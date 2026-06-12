package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/pmenglund/snok"
)

func main() {
	tree := snok.NewTree(snok.Config{Name: "snok-jsonl"})
	err := tree.AddCommand([]string{"events"}, snok.CommandDefinition{
		Description: "Emit sample events as JSON Lines.",
		JSONL:       true,
		Examples: []string{
			"snok-jsonl events --count 3",
		},
		Options: []snok.Field{
			{Name: "count", Kind: snok.FieldPrimitive, Type: snok.TypeNumber, Default: float64(2), Description: "Number of events to emit"},
		},
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			count := ctx.OptionInt("count")
			records := make([]any, 0, count)
			for i := 1; i <= count; i++ {
				records = append(records, map[string]any{
					"id":     i,
					"source": "snok-jsonl",
				})
			}
			return records, nil
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
