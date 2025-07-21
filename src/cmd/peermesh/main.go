package main

import (
	"fmt"
	"github.com/defenzysoft/PeerMesh/src/internal/cli"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 && (args[0] == "-i" || args[0] == "--interactive") {
		interactiveCLI := cli.NewInteractiveCLI()
		interactiveCLI.Start()
		return
	}

	if len(args) == 0 {
		fmt.Println("Starting PeerMesh in interactive mode...")
		fmt.Println("Use 'peermesh -h' for command line usage")
		fmt.Println()

		interactiveCLI := cli.NewInteractiveCLI()
		interactiveCLI.Start()
		return
	}

	cliInstance := cli.NewCLI()
	err := cliInstance.Run(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
