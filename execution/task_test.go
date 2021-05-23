package execution

import (
	"strings"
	"testing"
)

func TestTaskExec(t *testing.T) {
	te := &TaskExecution{
		Command:      "echo agent",
		Timeout:      0,
		Dependencies: nil,
	}
	output, err := te.Execute()
	if err != nil {
		t.Fatalf("error executing task for test: %v", err)
	}
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput != "agent" {
		t.Fatalf("error executing command for test. Expected output to be 'agent' but it is '%s'", trimmedOutput)
	}
}
