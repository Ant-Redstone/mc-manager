package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/automation"
	"github.com/lomokwa/mc-manager/db"
)

// seedDefaultServer inserts the row automation_rules.server_id points at.
// Without it every CreateRule here fails on the foreign key.
func seedDefaultServer(t *testing.T) {
	t.Helper()
	if _, err := db.DB.Exec(
		`INSERT INTO servers (id, name, dir, port) VALUES ('default', 'Default', './minecraft-server', 25565)`,
	); err != nil {
		t.Fatalf("failed to seed the server row: %v", err)
	}
}

func newRule(t *testing.T, name string) int {
	t.Helper()
	id, err := automation.CreateRule(automation.Rule{
		ServerID: "default", Name: name, Enabled: true,
		TriggerKind: "stop", TriggerConfig: map[string]any{},
		Actions: []automation.Action{{Type: "backup"}},
	})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	return id
}

// call runs a handler with the given params and body, the way the router would.
func call(h gin.HandlerFunc, method, path, body string, params gin.Params) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = params
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	c.Request = httptest.NewRequest(method, path, reader)
	if body != "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	h(c)
	return w
}

func TestListAutomations_ReturnsStoredRules(t *testing.T) {
	setupTestDB(t)
	seedDefaultServer(t)
	newRule(t, "avisa quando cair")

	w := call(ListAutomationsHandler, http.MethodGet, "/api/automations", "", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "avisa quando cair") {
		t.Errorf("expected the rule in the body, got %s", w.Body.String())
	}
}

// The credential must not reach the client. This is the single most important
// assertion in the whole REST layer.
func TestListAutomationWebhooks_NeverCarriesTheURL(t *testing.T) {
	setupTestDB(t)
	const secret = "https://discord.com/api/webhooks/1/SUPERSECRETTOKEN"
	if _, err := automation.CreateWebhook(automation.Webhook{Name: "alertas", URL: secret}); err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}

	w := call(ListAutomationWebhooksHandler, http.MethodGet, "/api/automation-webhooks", "", nil)

	body := w.Body.String()
	if !strings.Contains(body, "alertas") {
		t.Errorf("expected the webhook name, got %s", body)
	}
	for _, leak := range []string{"SUPERSECRETTOKEN", "discord.com/api/webhooks"} {
		if strings.Contains(body, leak) {
			t.Errorf("the webhook credential leaked into the response: %s", body)
		}
	}
}

func TestGetAutomation_404sForAnUnknownID(t *testing.T) {
	setupTestDB(t)

	w := call(GetAutomationHandler, http.MethodGet, "/api/automations/9999", "",
		gin.Params{{Key: "id", Value: "9999"}})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for an unknown rule, got %d", w.Code)
	}
}

// The id reaches a query, so anything non-numeric is rejected before it gets
// there rather than relied on to be harmless.
func TestGetAutomation_RejectsANonNumericID(t *testing.T) {
	setupTestDB(t)

	for _, bad := range []string{"../../etc/passwd", "1 OR 1=1", "", "-3", "abc"} {
		w := call(GetAutomationHandler, http.MethodGet, "/api/automations/x", "",
			gin.Params{{Key: "id", Value: bad}})
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for id %q, got %d", bad, w.Code)
		}
	}
}

func TestListFirings_ReturnsNewestFirst(t *testing.T) {
	setupTestDB(t)
	seedDefaultServer(t)
	id := newRule(t, "r")
	for _, trig := range []string{"primeiro", "segundo"} {
		if err := automation.RecordFiring(automation.Firing{RuleID: id, Trigger: trig, Outcome: "[]"}); err != nil {
			t.Fatalf("RecordFiring: %v", err)
		}
	}

	w := call(ListAutomationFiringsHandler, http.MethodGet, "/api/automations/1/firings", "",
		gin.Params{{Key: "id", Value: strconv.Itoa(id)}})

	var resp struct {
		Data []automation.Firing `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 2 || resp.Data[0].Trigger != "segundo" {
		t.Errorf("expected newest first, got %+v", resp.Data)
	}
}
