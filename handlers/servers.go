package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/middleware"
	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
)

// serverListItem is one row of GET /api/servers (and the whole body of
// GET /api/servers/:sid): the registry row plus this server's own live
// status, so the client can render the Servers list/card without a further
// per-server round trip -- see PLAN-multi-server.md D3/D5.
type serverListItem struct {
	types.Server
	Running bool       `json:"running"`
	PID     int        `json:"pid,omitempty"`
	Since   *time.Time `json:"since,omitempty"`
}

// statusItemFor folds rt's own live status into s for display. PID/Since
// are only populated when the runtime is authoritatively running
// (rt.IsServerRunning(), which already applies mc-supervisor's
// heartbeat-staleness rule -- see services/process.go) rather than trusting
// the raw status file's own "running" field, so a crashed container can
// never be reported with a stale PID/uptime.
func statusItemFor(s types.Server, rt *services.ServerRuntime) serverListItem {
	item := serverListItem{Server: s, Running: rt.IsServerRunning()}
	if item.Running {
		if st, ok := rt.ReadStatus(); ok {
			item.PID = st.PID
			since := st.Since
			item.Since = &since
		}
	}
	return item
}

// @Summary List servers
// @Description Lists every registered server with its live status
// @Tags servers
// @Produce json
// @Success 200 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Security BearerAuth
// @Router /api/servers [get]
//
// ListServersHandler is gated by nothing beyond the JWT every /api route
// already requires under main.go's `api` group -- the same gate GET
// /api/status uses, deliberately not e.g. players.view. This endpoint is
// really just N copies of /api/status (see statusItemFor): every caller
// needs it to even discover what a valid :sid is before any specific,
// separately permission-gated action (start/stop/console/players/...)
// becomes reachable, so tying it to one of THOSE permissions would be an
// arbitrary mismatch -- a players-only viewer would either wrongly see (if
// gated by players.view) or wrongly lose (if gated by anything else) the
// one list every role needs for basic navigation.
func ListServersHandler(c *gin.Context) {
	servers, err := services.ListServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: "failed to list servers"})
		return
	}

	items := make([]serverListItem, 0, len(servers))
	for _, s := range servers {
		rt, err := services.RuntimeForID(s.ID)
		if err != nil {
			// ListServers and RuntimeForID read the same table, so this row
			// should always resolve -- but report it as not-running rather
			// than fail the whole list over one unexpected row.
			items = append(items, serverListItem{Server: s})
			continue
		}
		items = append(items, statusItemFor(s, rt))
	}

	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: items})
}

// @Summary Get a server
// @Description Returns one registered server with its live status
// @Tags servers
// @Produce json
// @Param sid path string true "Server ID"
// @Success 200 {object} types.APIResponse
// @Failure 404 {object} types.APIResponse
// @Security BearerAuth
// @Router /api/servers/{sid} [get]
//
// GetServerHandler is mounted behind middleware.ResolveServer() in main.go,
// so an unknown :sid never reaches here -- ResolveServer already wrote the
// 404. The ok-check below is just defense in depth against a future caller
// mounting this handler without that middleware, not the primary guard.
func GetServerHandler(c *gin.Context) {
	rt, ok := middleware.RuntimeFromContext(c)
	if !ok {
		c.JSON(http.StatusNotFound, types.APIResponse{Error: "server not found"})
		return
	}

	s, err := services.GetServer(rt.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{Error: "server not found"})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: statusItemFor(s, rt)})
}
