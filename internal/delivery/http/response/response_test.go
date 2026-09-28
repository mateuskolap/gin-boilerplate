package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccessWritesStandardResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	Success(ctx, http.StatusCreated, "created", map[string]string{"id": "123"})
	var got ApiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	data, ok := got.Data.(map[string]any)
	if recorder.Code != http.StatusCreated || !got.Success || got.Message != "created" || !ok || data["id"] != "123" {
		t.Fatalf("Success() status=%d response=%+v", recorder.Code, got)
	}
}
