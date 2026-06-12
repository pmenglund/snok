package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pmenglund/snok"
	"github.com/pmenglund/snok/internal/hotrun"
)

func main() {
	name := filepath.Base(os.Args[0])
	if name != "snok" {
		manager, err := hotrun.NewManager()
		if err != nil {
			fatal(err)
		}
		if err := manager.Exec(name, os.Args[1:]); err != nil {
			fatal(err)
		}
		return
	}
	if err := snokCommand().Execute(); err != nil {
		var commandErr *snok.CommandError
		if errors.As(err, &commandErr) {
			fmt.Fprintln(os.Stderr, commandErr.Message)
			os.Exit(commandErr.ExitCode)
		}
		fatal(err)
	}
}

func snokCommand() interface{ Execute() error } {
	tree := snok.NewTree(snok.Config{
		Name:    "snok",
		Version: "0.1.0",
	})
	must(tree.AddCommand([]string{"link"}, snok.CommandDefinition{
		Description: "Link a command name to a Go source package.",
		Args: []snok.Field{
			{Name: "name", Kind: snok.FieldPrimitive, Type: snok.TypeString, Required: true, Description: "Command name to link"},
			{Name: "source", Kind: snok.FieldPrimitive, Type: snok.TypeString, Required: true, Description: "Go package directory to build"},
		},
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			manager, err := hotrun.NewManager()
			if err != nil {
				return nil, err
			}
			launcher, err := os.Executable()
			if err != nil {
				return nil, err
			}
			result, err := manager.Link(ctx.ArgString("name"), ctx.ArgString("source"), launcher)
			if err != nil {
				return nil, err
			}
			ctx.Output.Log("linked %s -> %s", result.Name, result.Source)
			ctx.Output.Log("created %s", result.LinkPath)
			if result.PathHint != "" {
				ctx.Output.Log("%s", result.PathHint)
			}
			return nil, nil
		},
	}))
	must(tree.AddCommand([]string{"unlink"}, snok.CommandDefinition{
		Description: "Remove a linked command.",
		Args: []snok.Field{
			{Name: "name", Kind: snok.FieldPrimitive, Type: snok.TypeString, Required: true, Description: "Command name to unlink"},
		},
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			manager, err := hotrun.NewManager()
			if err != nil {
				return nil, err
			}
			name := ctx.ArgString("name")
			if err := manager.Unlink(name); err != nil {
				return nil, err
			}
			ctx.Output.Log("unlinked %s", name)
			return nil, nil
		},
	}))
	must(tree.AddCommand([]string{"links"}, snok.CommandDefinition{
		Description: "List linked commands.",
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			manager, err := hotrun.NewManager()
			if err != nil {
				return nil, err
			}
			links, err := manager.Links()
			if err != nil {
				return nil, err
			}
			if len(links) == 0 {
				ctx.Output.Log("no links")
				return nil, nil
			}
			for _, link := range links {
				ctx.Output.Log("%s -> %s", link.Name, link.Source)
			}
			return nil, nil
		},
	}))
	return snok.CobraCommand(tree)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
