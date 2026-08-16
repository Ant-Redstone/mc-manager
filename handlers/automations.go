package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/automation"
	"github.com/lomokwa/mc-manager/middleware"
	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
)

// firingHistoryLimit bounds what one request can pull back. The table has no
// retention policy yet (deliberately out of scope in the design), so an
// unbounded list would eventually be the slowest endpoint in the app.
const firingHistoryLimit = 100

// idParam parses :id, rejecting anything non-numeric before it reaches a query.
// `what` names the thing so a webhook route does not report "invalid automation
// id". Returns false having already written the response.
func idParam(c *gin.Context, what string) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: "invalid " + what + " id"})
		return 0, false
	}
	return id, true
}

// ListAutomationsHandler returns every rule, enabled or not -- the management
// screen has to show the disabled ones too, unlike the engine.
func ListAutomationsHandler(c *gin.Context) {
	rules, err := automation.ListRules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: rules})
}

func GetAutomationHandler(c *gin.Context) {
	id, ok := idParam(c, "automation")
	if !ok {
		return
	}
	rule, err := automation.GetRule(id)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, types.APIResponse{Error: "automation not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: rule})
}

// ListAutomationWebhooksHandler returns destinations WITHOUT their URLs.
// automation.ListWebhooks does not select the column and Webhook.URL is
// json:"-", so the credential cannot be included here by accident -- the types
// do the remembering rather than the reviewer.
func ListAutomationWebhooksHandler(c *gin.Context) {
	hooks, err := automation.ListWebhooks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: hooks})
}

func ListAutomationFiringsHandler(c *gin.Context) {
	id, ok := idParam(c, "automation")
	if !ok {
		return
	}
	firings, err := automation.ListFirings(id, firingHistoryLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: firings})
}

// automationRequest is the write shape. Deliberately NOT automation.Rule:
// binding straight onto the storage type would let a client set ID,
// LastFiredAt or CreatedAt -- and last_fired_at is what the cooldown reads, so
// a client that could post it could bypass every cooldown in the system.
type automationRequest struct {
	ServerID          string              `json:"server_id"`
	Name              string              `json:"name"`
	Enabled           *bool               `json:"enabled"`
	TriggerKind       string              `json:"trigger_kind"`
	TriggerConfig     map[string]any      `json:"trigger_config"`
	Actions           []automation.Action `json:"actions"`
	CooldownSeconds   int                 `json:"cooldown_seconds"`
	StopOnFailure     *bool               `json:"stop_on_failure"`
	DeafWindowSeconds int                 `json:"deaf_window_seconds"`
}

// engineReloader is set at boot so a write makes the engine re-read its rules
// immediately. Nil in tests that do not care.
var engineReloader func() error

// SetEngineReloader wires the running engine to the handlers. Called once from
// main; tests use it to observe that writes trigger a reload.
func SetEngineReloader(fn func() error) { engineReloader = fn }

func reloadEngine() {
	if engineReloader == nil {
		return
	}
	if err := engineReloader(); err != nil {
		slog.Error("automations: failed to reload rules after a write", "err", err)
	}
}

// toRule validates a request and converts it. Every rejection is a message the
// builder can show verbatim, because the person who typed it is the person who
// can fix it.
func (r automationRequest) toRule() (automation.Rule, error) {
	if strings.TrimSpace(r.Name) == "" {
		return automation.Rule{}, errors.New("an automation needs a name")
	}
	if _, err := services.RuntimeForID(r.ServerID); err != nil {
		return automation.Rule{}, fmt.Errorf("unknown server %q", r.ServerID)
	}
	if !automation.IsKnownTrigger(r.TriggerKind) {
		return automation.Rule{}, fmt.Errorf("unknown trigger %q", r.TriggerKind)
	}
	if err := automation.ValidateActions(r.Actions); err != nil {
		return automation.Rule{}, err
	}
	if r.CooldownSeconds < 0 || r.DeafWindowSeconds < 0 {
		return automation.Rule{}, errors.New("cooldown and deaf window cannot be negative")
	}

	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	// The safe default, matching the design: [warn on Discord] -> [restart]
	// with a dead webhook must not disconnect anyone unwarned.
	stopOnFailure := true
	if r.StopOnFailure != nil {
		stopOnFailure = *r.StopOnFailure
	}
	cfg := r.TriggerConfig
	if cfg == nil {
		cfg = map[string]any{}
	}

	return automation.Rule{
		ServerID: r.ServerID, Name: strings.TrimSpace(r.Name), Enabled: enabled,
		TriggerKind: r.TriggerKind, TriggerConfig: cfg, Actions: r.Actions,
		CooldownSeconds: r.CooldownSeconds, StopOnFailure: stopOnFailure,
		DeafWindowSeconds: r.DeafWindowSeconds,
	}, nil
}

func CreateAutomationHandler(c *gin.Context) {
	var req automationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: "invalid request body"})
		return
	}
	rule, err := req.toRule()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: err.Error()})
		return
	}
	if uid, ok := middleware.UserIDFromContext(c); ok {
		rule.CreatedBy = &uid
	}

	id, err := automation.CreateRule(rule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: err.Error()})
		return
	}
	reloadEngine()

	rule.ID = id
	c.JSON(http.StatusCreated, types.APIResponse{Success: true, Data: rule})
}

func UpdateAutomationHandler(c *gin.Context) {
	id, ok := idParam(c, "automation")
	if !ok {
		return
	}
	if _, err := automation.GetRule(id); errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, types.APIResponse{Error: "automation not found"})
		return
	}

	var req automationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: "invalid request body"})
		return
	}
	rule, err := req.toRule()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: err.Error()})
		return
	}
	rule.ID = id

	if err := automation.UpdateRule(rule); err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: err.Error()})
		return
	}
	reloadEngine()
	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: rule})
}
