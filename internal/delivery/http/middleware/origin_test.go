package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const allowed = "https://frontend.example.com"
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, origin := range []string{"", "null", "https://untrusted.example.com", allowed + ".attacker.com", allowed + "/", "http://frontend.example.com", allowed + ":444", allowed} {
			t.Run(method+"/"+origin, func(t *testing.T) {
				router := gin.New()
				router.Use(ErrorHandler(), ValidateOrigin([]string{allowed}))
				called := false
				router.Handle(method, "/", func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) })
				req := httptest.NewRequest(method, "/", nil)
				if origin != "" {
					req.Header.Set("Origin", origin)
				}
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, req)
				want := http.StatusForbidden
				if origin == allowed {
					want = http.StatusNoContent
				}
				if recorder.Code != want || called != (origin == allowed) {
					t.Fatalf("status=%d, handler called=%v; want status=%d", recorder.Code, called, want)
				}
			})
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		router := gin.New()
		router.Use(ValidateOrigin(nil))
		router.Handle(method, "/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, "/", nil))
		if recorder.Code != http.StatusNoContent {
			t.Errorf("safe method %s without Origin was rejected", method)
		}
	}
}
