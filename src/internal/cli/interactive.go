package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// InteractiveCLI represents an interactive command line interface
type InteractiveCLI struct {
	cli     *CLI
	scanner *bufio.Scanner
	running bool
	prompt  string
}

// NewInteractiveCLI creates a new interactive CLI
func NewInteractiveCLI() *InteractiveCLI {
	return &InteractiveCLI{
		cli:     NewCLI(),
		scanner: bufio.NewScanner(os.Stdin),
		running: true,
		prompt:  "peermesh> ",
	}
}

// Start starts the interactive CLI
func (i *InteractiveCLI) Start() {
	fmt.Println("PeerMesh Interactive CLI")
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	for i.running {
		fmt.Print(i.prompt)

		if !i.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(i.scanner.Text())
		if input == "" {
			continue
		}

		args := i.parseCommand(input)
		if len(args) == 0 {
			continue
		}

		if args[0] == "exit" || args[0] == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if args[0] == "clear" {
			i.clearScreen()
			continue
		}

		err := i.cli.Run(args)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		fmt.Println()
	}
}

// parseCommand parses a command string into arguments
func (i *InteractiveCLI) parseCommand(input string) []string {
	args := strings.Fields(input)
	return args
}

// clearScreen clears the terminal screen
func (i *InteractiveCLI) clearScreen() {
	for j := 0; j < 50; j++ {
		fmt.Println()
	}
}

// SetPrompt sets the CLI prompt
func (i *InteractiveCLI) SetPrompt(prompt string) {
	i.prompt = prompt
}

// Stop stops the interactive CLI
func (i *InteractiveCLI) Stop() {
	i.running = false
}
