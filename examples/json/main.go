package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/pmenglund/snok"
)

func main() {
	tree := snok.NewTree(snok.Config{Name: "snok-json"})
	err := tree.AddCommand([]string{"sum"}, snok.CommandDefinition{
		Description: "Sum two numbers.",
		JSON:        true,
		Examples: []string{
			"snok-json sum 2 3 --json",
		},
		Args: []snok.Field{
			{Name: "left", Kind: snok.FieldPrimitive, Type: snok.TypeNumber, Required: true},
			{Name: "right", Kind: snok.FieldPrimitive, Type: snok.TypeNumber, Required: true},
		},
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			left := ctx.ArgNumber("left")
			right := ctx.ArgNumber("right")
			if ctx.OptionBool("json") {
				return map[string]any{
					"left":  left,
					"right": right,
					"sum":   left + right,
				}, nil
			}
			ctx.Output.Log("%g", left+right)
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
