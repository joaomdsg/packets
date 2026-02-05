package agent_test

import (
	"testing"
	"time"

	"github.com/joaomdsg/agntpr/internal/agent"
)

func TestNewOpenCodeRunner(t *testing.T) {
	runner := agent.NewOpenCodeRunner(30 * time.Second)

	if runner == nil {
		t.Error("expected non-nil runner")
	}
}
