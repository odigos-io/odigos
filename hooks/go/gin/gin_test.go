package gin

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// The wrapper runs the middleware chain on a goroutine it spawns itself, so a
// panic anywhere in that chain unwinds a stack that gin's Recovery middleware
// cannot see. Left unhandled it terminates the whole application process, which
// no `recover` inside this test binary can observe — hence the child process.
const scenarioEnvVar = "ODIGOS_GIN_HOOKS_TEST_SCENARIO"

func TestPanicInHandlerDoesNotCrashTheProcess(t *testing.T) {
	runScenarioInChildProcess(t, "handler", "TestPanicInHandlerDoesNotCrashTheProcess")
}

func TestPanicInWrappedMiddlewareDoesNotCrashTheProcess(t *testing.T) {
	runScenarioInChildProcess(t, "middleware", "TestPanicInWrappedMiddlewareDoesNotCrashTheProcess")
}

func runScenarioInChildProcess(t *testing.T, scenario string, testName string) {
	t.Helper()

	if os.Getenv(scenarioEnvVar) == scenario {
		servePanickingRequest(scenario)
		return
	}

	command := exec.Command(os.Args[0], "-test.run=^"+testName+"$", "-test.timeout=60s")
	command.Env = append(os.Environ(), scenarioEnvVar+"="+scenario)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("the application process did not survive a panic in the %s: %v\n%s", scenario, err, output)
	}
	if !bytes.Contains(output, []byte("SCENARIO_STATUS=500")) {
		t.Fatalf("expected gin's Recovery middleware to answer 500, got:\n%s", output)
	}
}

// servePanickingRequest drives one request whose chain panics, through an engine
// wrapped exactly the way the Go hooks documentation describes.
func servePanickingRequest(scenario string) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	middleware := func(c *gin.Context) { c.Next() }
	if scenario == "middleware" {
		middleware = func(c *gin.Context) { panic("middleware panicked") }
	}
	engine.Use(OdigosGinMiddleware(middleware)...)

	engine.GET("/panic", func(c *gin.Context) {
		if scenario == "handler" {
			var attributes map[string]string
			attributes["key"] = "value"
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))
	fmt.Printf("SCENARIO_STATUS=%d\n", recorder.Code)
}

func TestOdigosGinMiddlewareRunsEveryMiddlewareInOrder(t *testing.T) {
	var calls []string
	appendCall := func(name string) gin.HandlerFunc {
		return func(c *gin.Context) {
			calls = append(calls, name)
			c.Next()
		}
	}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(OdigosGinMiddleware(appendCall("first"), appendCall("second"))...)
	engine.GET("/", func(c *gin.Context) {
		calls = append(calls, "handler")
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if got := strings.Join(calls, ","); got != "first,second,handler" {
		t.Fatalf("expected the wrapped middlewares to run in order, got %q", got)
	}
}

func TestOdigosGinMiddlewareKeepsAbortWorking(t *testing.T) {
	handlerCalled := false

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(OdigosGinMiddleware(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	})...)
	engine.GET("/", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
	if handlerCalled {
		t.Fatal("expected the aborted chain to skip the route handler")
	}
}

func TestOdigosGinMiddlewareWrapsEveryMiddleware(t *testing.T) {
	wrapped := OdigosGinMiddleware(func(c *gin.Context) {}, func(c *gin.Context) {})
	if len(wrapped) != 2 {
		t.Fatalf("expected 2 wrapped middlewares, got %d", len(wrapped))
	}
}
