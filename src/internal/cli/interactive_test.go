package cli

import (
	"testing"
)

func TestNewInteractiveCLI(t *testing.T) {
	interactiveCLI := NewInteractiveCLI()
	if interactiveCLI == nil {
		t.Fatal("Interactive CLI is nil")
	}

	if interactiveCLI.cli == nil {
		t.Error("Interactive CLI should have a CLI instance")
	}

	if interactiveCLI.scanner == nil {
		t.Error("Interactive CLI should have a scanner")
	}

	if !interactiveCLI.running {
		t.Error("Interactive CLI should be running initially")
	}

	if interactiveCLI.prompt != "peermesh> " {
		t.Errorf("Expected prompt 'peermesh> ', got '%s'", interactiveCLI.prompt)
	}

	t.Logf("Interactive CLI created successfully")
}

func TestInteractiveCLISetPrompt(t *testing.T) {
	interactiveCLI := NewInteractiveCLI()

	newPrompt := "test> "
	interactiveCLI.SetPrompt(newPrompt)

	if interactiveCLI.prompt != newPrompt {
		t.Errorf("Expected prompt '%s', got '%s'", newPrompt, interactiveCLI.prompt)
	}

	t.Logf("Prompt set successfully")
}

func TestInteractiveCLIStop(t *testing.T) {
	interactiveCLI := NewInteractiveCLI()

	if !interactiveCLI.running {
		t.Error("Interactive CLI should be running initially")
	}

	interactiveCLI.Stop()

	if interactiveCLI.running {
		t.Error("Interactive CLI should be stopped")
	}

	t.Logf("Interactive CLI stopped successfully")
}

func TestInteractiveCLIParseCommand(t *testing.T) {
	interactiveCLI := NewInteractiveCLI()

	args := interactiveCLI.parseCommand("help")
	if len(args) != 1 || args[0] != "help" {
		t.Errorf("Expected ['help'], got %v", args)
	}

	args = interactiveCLI.parseCommand("join --server example.com --name test")
	if len(args) != 4 {
		t.Errorf("Expected 4 arguments, got %d", len(args))
	}

	args = interactiveCLI.parseCommand("")
	if len(args) != 0 {
		t.Errorf("Expected empty args, got %v", args)
	}

	args = interactiveCLI.parseCommand("   ")
	if len(args) != 0 {
		t.Errorf("Expected empty args for whitespace, got %v", args)
	}

	t.Logf("Command parsing tests passed")
}

func TestInteractiveCLIClearScreen(t *testing.T) {
	interactiveCLI := NewInteractiveCLI()

	interactiveCLI.clearScreen()

	t.Logf("Clear screen function works")
}
