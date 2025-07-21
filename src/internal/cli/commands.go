package cli

import (
	"fmt"
	"os"
	"strings"
)

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Usage       string
	Handler     func(args []string) error
}

// CLI represents the command line interface
type CLI struct {
	commands map[string]*Command
}

// NewCLI creates a new CLI instance
func NewCLI() *CLI {
	cli := &CLI{
		commands: make(map[string]*Command),
	}

	cli.registerDefaultCommands()

	return cli
}

// RegisterCommand registers a new command
func (c *CLI) RegisterCommand(cmd *Command) {
	c.commands[cmd.Name] = cmd
}

// registerDefaultCommands registers the default commands
func (c *CLI) registerDefaultCommands() {
	c.RegisterCommand(&Command{
		Name:        "help",
		Description: "Show help information",
		Usage:       "help [command]",
		Handler:     c.handleHelp,
	})

	c.RegisterCommand(&Command{
		Name:        "version",
		Description: "Show version information",
		Usage:       "version",
		Handler:     c.handleVersion,
	})

	c.RegisterCommand(&Command{
		Name:        "join",
		Description: "Join the peer-to-peer network",
		Usage:       "join [--server <discovery-server>] [--name <peer-name>]",
		Handler:     c.handleJoin,
	})

	c.RegisterCommand(&Command{
		Name:        "leave",
		Description: "Leave the peer-to-peer network",
		Usage:       "leave",
		Handler:     c.handleLeave,
	})

	c.RegisterCommand(&Command{
		Name:        "status",
		Description: "Show network status",
		Usage:       "status",
		Handler:     c.handleStatus,
	})

	c.RegisterCommand(&Command{
		Name:        "peers",
		Description: "List connected peers",
		Usage:       "peers",
		Handler:     c.handlePeers,
	})
}

// Run executes the CLI with the given arguments
func (c *CLI) Run(args []string) error {
	if len(args) == 0 {
		return c.handleHelp([]string{})
	}

	commandName := args[0]
	commandArgs := args[1:]

	cmd, exists := c.commands[commandName]
	if !exists {
		return fmt.Errorf("unknown command: %s\nUse 'help' for available commands", commandName)
	}

	return cmd.Handler(commandArgs)
}

// handleHelp handles the help command
func (c *CLI) handleHelp(args []string) error {
	if len(args) > 0 {
		commandName := args[0]
		cmd, exists := c.commands[commandName]
		if !exists {
			return fmt.Errorf("unknown command: %s", commandName)
		}

		fmt.Printf("Usage: %s\n", cmd.Usage)
		fmt.Printf("Description: %s\n", cmd.Description)
		return nil
	}

	fmt.Println("PeerMesh - Zero-configuration P2P VPN")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println()

	for _, cmd := range c.commands {
		fmt.Printf("  %-15s %s\n", cmd.Name, cmd.Description)
	}

	fmt.Println()
	fmt.Println("Use 'help <command>' for detailed information about a command")
	return nil
}

// handleVersion handles the version command
func (c *CLI) handleVersion(args []string) error {
	fmt.Println("PeerMesh v0.1.0")
	fmt.Println("Zero-configuration peer-to-peer VPN")
	return nil
}

// handleJoin handles the join command
func (c *CLI) handleJoin(args []string) error {
	fmt.Println("Joining peer-to-peer network...")

	serverAddr := "localhost:8080"
	peerName := "peer-" + generatePeerID()

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--server":
			if i+1 < len(args) {
				serverAddr = args[i+1]
				i++
			} else {
				return fmt.Errorf("--server requires a value")
			}
		case "--name":
			if i+1 < len(args) {
				peerName = args[i+1]
				i++
			} else {
				return fmt.Errorf("--name requires a value")
			}
		}
	}

	fmt.Printf("Discovery server: %s\n", serverAddr)
	fmt.Printf("Peer name: %s\n", peerName)

	// TODO: Implement actual join logic
	fmt.Println("Join functionality will be implemented in the next phase")

	return nil
}

// handleLeave handles the leave command
func (c *CLI) handleLeave(args []string) error {
	fmt.Println("Leaving peer-to-peer network...")

	// TODO: Implement actual leave logic
	fmt.Println("Leave functionality will be implemented in the next phase")

	return nil
}

// handleStatus handles the status command
func (c *CLI) handleStatus(args []string) error {
	fmt.Println("Network Status:")
	fmt.Println("===============")
	fmt.Println("Status: Not connected")
	fmt.Println("Discovery Server: None")
	fmt.Println("Connected Peers: 0")
	fmt.Println("VPN Interface: Down")

	// TODO: Implement actual status logic
	fmt.Println("\nStatus functionality will be implemented in the next phase")

	return nil
}

// handlePeers handles the peers command
func (c *CLI) handlePeers(args []string) error {
	fmt.Println("Connected Peers:")
	fmt.Println("================")
	fmt.Println("No peers connected")

	// TODO: Implement actual peers logic
	fmt.Println("\nPeers functionality will be implemented in the next phase")

	return nil
}

// generatePeerID generates a simple peer ID
func generatePeerID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	hash := 0
	for _, char := range hostname {
		hash = (hash*31 + int(char)) % 10000
	}

	return fmt.Sprintf("%s-%04d", strings.ToLower(hostname), hash)
}
