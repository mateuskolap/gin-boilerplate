package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWritesConsoleLogsAndErrorOnlyJSONFile(t *testing.T) {
	for _, tc := range []struct {
		name        string
		environment string
		wantJSON    bool
	}{
		{name: "development text", environment: "development"},
		{name: "production json", environment: "production", wantJSON: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var console bytes.Buffer
			logPath := filepath.Join(t.TempDir(), "nested", "app.log")
			logger, file, err := New(tc.environment, &console, logPath)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			logger.Info("visible info")
			logger.Error("stored error")
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			fileData, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(console.String(), "visible info") || !strings.Contains(console.String(), "stored error") {
				t.Fatalf("console log output = %q", console.String())
			}
			if tc.wantJSON && !strings.Contains(console.String(), `"msg":"visible info"`) {
				t.Fatalf("production console output is not JSON: %q", console.String())
			}
			if strings.Contains(string(fileData), "visible info") || !strings.Contains(string(fileData), `"msg":"stored error"`) {
				t.Fatalf("error-only file output = %q", fileData)
			}
		})
	}
}
