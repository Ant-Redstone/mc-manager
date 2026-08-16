package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/automation"
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
