package cli

import (
	"testing"
)

func TestNewCLI(t *testing.T) {
	cli := NewCLI()
	if cli == nil {
		t.Fatal("CLI is nil")
	}

	if len(cli.commands) == 0 {
		t.Error("CLI should have default commands")
	}

	requiredCommands := []string{"help", "version", "join", "leave", "status", "peers"}
	for _, cmdName := range requiredCommands {
		if _, exists := cli.commands[cmdName]; !exists {
			t.Errorf("Missing required command: %s", cmdName)
		}
	}

	t.Logf("CLI created successfully with %d commands", len(cli.commands))
}

func TestCLIHelp(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"help"})
	if err != nil {
		t.Errorf("Help command failed: %v", err)
	}

	t.Logf("Help command executed successfully")
}

func TestCLIHelpSpecificCommand(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"help", "join"})
	if err != nil {
		t.Errorf("Help for join command failed: %v", err)
	}

	t.Logf("Help for specific command executed successfully")
}

func TestCLIHelpUnknownCommand(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"help", "unknown"})
	if err == nil {
		t.Error("Expected error for unknown command")
	}

	t.Logf("Help for unknown command correctly returned error")
}

func TestCLIVersion(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"version"})
	if err != nil {
		t.Errorf("Version command failed: %v", err)
	}

	t.Logf("Version command executed successfully")
}

func TestCLIJoin(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"join"})
	if err != nil {
		t.Errorf("Join command failed: %v", err)
	}

	t.Logf("Join command executed successfully")
}

func TestCLIJoinWithArguments(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"join", "--server", "example.com:8080", "--name", "test-peer"})
	if err != nil {
		t.Errorf("Join command with arguments failed: %v", err)
	}

	t.Logf("Join command with arguments executed successfully")
}

func TestCLIJoinInvalidArguments(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"join", "--server"})
	if err == nil {
		t.Error("Expected error for missing server value")
	}

	t.Logf("Join command correctly handled invalid arguments")
}

func TestCLILeave(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"leave"})
	if err != nil {
		t.Errorf("Leave command failed: %v", err)
	}

	t.Logf("Leave command executed successfully")
}

func TestCLIStatus(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"status"})
	if err != nil {
		t.Errorf("Status command failed: %v", err)
	}

	t.Logf("Status command executed successfully")
}

func TestCLIPeers(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"peers"})
	if err != nil {
		t.Errorf("Peers command failed: %v", err)
	}

	t.Logf("Peers command executed successfully")
}

func TestCLIUnknownCommand(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{"unknown"})
	if err == nil {
		t.Error("Expected error for unknown command")
	}

	t.Logf("Unknown command correctly returned error")
}

func TestCLINoArguments(t *testing.T) {
	cli := NewCLI()

	err := cli.Run([]string{})
	if err != nil {
		t.Errorf("No arguments should show help: %v", err)
	}

	t.Logf("No arguments correctly showed help")
}

func TestGeneratePeerID(t *testing.T) {
	peerID := generatePeerID()
	if peerID == "" {
		t.Error("Generated peer ID should not be empty")
	}

	if len(peerID) < 5 {
		t.Error("Generated peer ID should be at least 5 characters")
	}

	t.Logf("Generated peer ID: %s", peerID)
}
