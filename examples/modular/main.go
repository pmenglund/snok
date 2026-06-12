package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/pmenglund/snok"
)

type commandRegistration func(*snok.Tree) error

var commandRegistrations []commandRegistration

func registerCommand(register commandRegistration) {
	commandRegistrations = append(commandRegistrations, register)
}

func main() {
	tree := snok.NewTree(snok.Config{
		Name:    "snok-modular",
		Version: "0.1.0",
	})
	for _, register := range commandRegistrations {
		if err := register(tree); err != nil {
			fatal(err)
		}
	}
	execute(tree)
}

func execute(tree *snok.Tree) {
	if err := snok.CobraCommand(tree).Execute(); err != nil {
		var commandErr *snok.CommandError
		if errors.As(err, &commandErr) {
			fmt.Fprintln(os.Stderr, commandErr.Message)
			os.Exit(commandErr.ExitCode)
		}
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
