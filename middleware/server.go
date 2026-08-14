package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
)

// runtimeContextKey is the gin.Context key ResolveServer stores the resolved
// *services.ServerRuntime under. Unexported -- RuntimeFromContext is the
// only supported way to read it back from another package, the same way
// UserIDFromContext (auth.go) is the only supported way to read the
// "userID" key ValidateJWT sets.
const runtimeContextKey = "serverRuntime"

// ResolveServer resolves a route's :sid param to a *services.ServerRuntime
// and stores it in the gin context for downstream handlers to read via
// RuntimeFromContext. It must be mounted on routes that declare a :sid
// param (see main.go's /api/servers/:sid group). An unknown id 404s with
// the standard types.APIResponse shape, same as every other "not found" in
// this API, instead of falling through to a handler with no runtime to act
// on.
//
// Security (PLAN-multi-server.md D4's "Path traversal via :sid" risk row):
// :sid is NEVER turned into a filesystem path, here or anywhere downstream.
// services.RuntimeForID treats it purely as a lookup key into the servers
// table's parameterized query -- the directory a handler eventually touches
// always comes from that row's own dir column, never from this request. A
// traversal-shaped id (e.g. "../../etc") is therefore just a string that
// fails to match any row: it 404s exactly like any other unknown id rather
// than resolving anywhere. See server_test.go for the regression tests
// proving this.
func ResolveServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := c.Param("sid")

		rt, err := services.RuntimeForID(sid)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, types.APIResponse{Error: "server not found"})
			return
		}

		c.Set(runtimeContextKey, rt)
		c.Next()
	}
}

// RuntimeFromContext returns the *services.ServerRuntime ResolveServer
// stored for this request, mirroring UserIDFromContext's pattern (auth.go).
// ok is false whenever ResolveServer never ran -- which is exactly what
// happens on a flat route (e.g. /api/players): those routes have no :sid
// and never mount this middleware, so callers are expected to fall back to
// services.DefaultRuntime() in that case (see handlers/runtime.go's
// runtimeFromRequest, the one shared helper every handler that needs to
// support both a flat and a namespaced route goes through).
func RuntimeFromContext(c *gin.Context) (*services.ServerRuntime, bool) {
	raw, exists := c.Get(runtimeContextKey)
	if !exists {
		return nil, false
	}
	rt, ok := raw.(*services.ServerRuntime)
	return rt, ok
}
