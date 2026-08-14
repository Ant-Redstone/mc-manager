package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
)

// capturedResolve records what a probe handler mounted behind ResolveServer
// observed, so tests can assert both "did the request reach past the
// middleware" and "which runtime, if any, did it see".
type capturedResolve struct {
	reached bool
	rt      *services.ServerRuntime
}

func newResolveServerRouter(capture *capturedResolve) *gin.Engine {
	r := gin.New()
	r.GET("/servers/:sid/probe", ResolveServer(), func(c *gin.Context) {
		capture.reached = true
		if rt, ok := RuntimeFromContext(c); ok {
			capture.rt = rt
		}
		c.JSON(http.StatusOK, types.APIResponse{Success: true})
	})
	return r
}

func TestResolveServer_UnknownID_404sWithStandardShape(t *testing.T) {
	setupTestDB(t) // defined in permissions_test.go, same package

	var capture capturedResolve
	r := newResolveServerRouter(&capture)

	req := httptest.NewRequest(http.MethodGet, "/servers/does-not-exist/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}
	if capture.reached {
		t.Error("expected the downstream handler to never run for an unknown id")
	}

	var resp types.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == "" {
		t.Error("expected a non-empty error message in the standard APIResponse shape")
	}
}

func TestResolveServer_KnownID_ReachesHandlerWithRuntimeInContext(t *testing.T) {
	setupTestDB(t)
	if err := services.EnsureDefaultServer(); err != nil {
		t.Fatalf("failed to seed default server: %v", err)
	}

	var capture capturedResolve
	r := newResolveServerRouter(&capture)

	req := httptest.NewRequest(http.MethodGet, "/servers/"+services.DefaultServerID+"/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !capture.reached {
		t.Fatal("expected the downstream handler to run for a known id")
	}
	if capture.rt == nil {
		t.Fatal("expected a runtime to be stored in the context")
	}
	if capture.rt.ID != services.DefaultServerID {
		t.Errorf("expected resolved runtime id %q, got %q", services.DefaultServerID, capture.rt.ID)
	}
	if capture.rt.Dir != services.ServerDir {
		t.Errorf("expected resolved runtime dir %q (from the registry row, not the request), got %q", services.ServerDir, capture.rt.Dir)
	}
}

// TestResolveServer_TraversalLikeID_NeverResolves is the regression test
// PLAN-multi-server.md D4's "path traversal via :sid" risk row calls for:
// no matter what string a :sid param holds, ResolveServer must treat it
// purely as a services.GetServer lookup key, never as a filesystem path. A
// traversal-shaped value simply fails to match any registry row, so it
// 404s exactly like any other unknown id -- there is no code path where
// such a string is ever joined onto rt.Dir or opened. Params are set
// directly on the context here (rather than relying on how a specific raw
// HTTP request line happens to get tokenized by gin's router) so this
// proves the middleware's OWN logic is safe regardless of how a given
// string ends up as :sid.
func TestResolveServer_TraversalLikeID_NeverResolves(t *testing.T) {
	setupTestDB(t)
	if err := services.EnsureDefaultServer(); err != nil {
		t.Fatalf("failed to seed default server: %v", err)
	}

	traversalIDs := []string{
		"../../etc",
		"..%2f..",
		"../../../etc/passwd",
		"..",
		"/etc/passwd",
		"....//....//etc",
		"..\\..\\windows",
	}

	for _, id := range traversalIDs {
		t.Run(id, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/servers/x/probe", nil)
			// This is the property under test: whatever string gin's router
			// hands ResolveServer as :sid, the middleware must never treat it
			// as anything but a database lookup key.
			c.Params = gin.Params{{Key: "sid", Value: id}}

			ResolveServer()(c)

			if w.Code != http.StatusNotFound {
				t.Errorf("id %q: expected 404, got %d", id, w.Code)
			}
			if rt, ok := RuntimeFromContext(c); ok {
				t.Errorf("id %q: expected no runtime to be resolved, got %+v", id, rt)
			}
		})
	}
}

// TestResolveServer_DotDotSegment_ViaRealHTTPRequest exercises the same
// property as TestResolveServer_TraversalLikeID_NeverResolves but through
// an actual HTTP request parsed by Go's net/http and matched by gin's real
// router, rather than a hand-built gin.Context: "/servers/../probe" parses
// to the single path segment ".." for :sid (gin's router does not collapse
// "." / ".." segments the way a filesystem or a browser resolving a
// relative link would), which is just a string that isn't a registry id.
func TestResolveServer_DotDotSegment_ViaRealHTTPRequest(t *testing.T) {
	setupTestDB(t)
	if err := services.EnsureDefaultServer(); err != nil {
		t.Fatalf("failed to seed default server: %v", err)
	}

	var capture capturedResolve
	r := newResolveServerRouter(&capture)

	req := httptest.NewRequest(http.MethodGet, "/servers/../probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}
	if capture.reached {
		t.Error("expected the downstream handler to never run")
	}
}

func TestRuntimeFromContext_NotSet(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if rt, ok := RuntimeFromContext(c); ok {
		t.Errorf("expected ok=false when ResolveServer never ran, got %+v", rt)
	}
}

func TestRuntimeFromContext_WrongType(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(runtimeContextKey, "not-a-runtime")
	if rt, ok := RuntimeFromContext(c); ok {
		t.Errorf("expected ok=false for a non-*ServerRuntime value, got %+v", rt)
	}
}
