package port

import (
	"encoding/json"
	"testing"
	"time"
)

func TestQueueTaskValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		task QueueTask
		want bool
	}{
		{name: "valid", task: QueueTask{Type: "cleanup", Payload: json.RawMessage(`{}`)}, want: true},
		{name: "blank type", task: QueueTask{Type: "  ", Payload: json.RawMessage(`{}`)}},
		{name: "missing payload", task: QueueTask{Type: "cleanup"}},
		{name: "invalid payload", task: QueueTask{Type: "cleanup", Payload: json.RawMessage(`{`)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.task.Validate()
			if (err == nil) != tc.want {
				t.Fatalf("Validate() error = %v, want success %v", err, tc.want)
			}
		})
	}
}

func TestDispatchOptionsValidation(t *testing.T) {
	valid := DispatchOptions{Queue: "maintenance", Timeout: time.Second, MaxRetries: 2, UniqueFor: time.Minute, Retention: time.Hour}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() rejected valid options: %v", err)
	}
	for _, invalid := range []DispatchOptions{
		{Queue: " ", Timeout: time.Second},
		{Queue: "default", Timeout: 0},
		{Queue: "default", Timeout: time.Second, MaxRetries: -1},
		{Queue: "default", Timeout: time.Second, UniqueFor: -time.Second},
		{Queue: "default", Timeout: time.Second, Retention: -time.Second},
	} {
		if err := invalid.Validate(); err == nil {
			t.Errorf("Validate() accepted invalid options: %+v", invalid)
		}
	}
}
