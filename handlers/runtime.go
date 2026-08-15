package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/middleware"
	"github.com/lomokwa/mc-manager/services"
)

// runtimeFromRequest returns the *services.ServerRuntime this request should
// operate on. A namespaced /api/servers/:sid/... route runs
// middleware.ResolveServer first, which stores the resolved runtime in the
// gin context; a flat route (e.g. /api/players, /api/start) never mounts
// that middleware, so this falls back to services.DefaultRuntime() -- the
// exact runtime every one of these handlers operated on before Phase 3
// added namespacing. This is the one shared helper PLAN-multi-server.md D3
// calls for, so every handler below reads as a single line instead of
// repeating the same "context lookup, else default" branch a dozen times.
func runtimeFromRequest(c *gin.Context) *services.ServerRuntime {
	if rt, ok := middleware.RuntimeFromContext(c); ok {
		return rt
	}
	return services.DefaultRuntime()
}
