package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pmenglund/snok"
)

func main() {
	tree := snok.NewTree(snok.Config{
		Name:    "snok-text",
		Version: "0.1.0",
	})
	err := tree.AddCommand([]string{"greet"}, snok.CommandDefinition{
		Description: "Print a greeting.",
		Examples: []string{
			"snok-text greet Ada --title Dr --shout",
		},
		Args: []snok.Field{
			{Name: "name", Kind: snok.FieldPrimitive, Type: snok.TypeString, Required: true, Description: "Person to greet"},
		},
		Options: []snok.Field{
			{Name: "title", Kind: snok.FieldPrimitive, Type: snok.TypeString, Default: "", Description: "Optional title"},
			{Name: "shout", Kind: snok.FieldPrimitive, Type: snok.TypeBoolean, Short: "s", Description: "Uppercase the greeting"},
		},
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			name := ctx.ArgString("name")
			title := ctx.OptionString("title")
			if title != "" {
				name = title + " " + name
			}
			message := "Hello, " + name + "!"
			if ctx.OptionBool("shout") {
				message = strings.ToUpper(message)
			}
			ctx.Output.Log("%s", message)
			return nil, nil
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	execute(tree)
}

func execute(tree *snok.Tree) {
	if err := snok.CobraCommand(tree).Execute(); err != nil {
		var commandErr *snok.CommandError
		if errors.As(err, &commandErr) {
			os.Exit(commandErr.ExitCode)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
