package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	must(tree.AddCommand([]string{"init"}, snok.CommandDefinition{
		Description: "Initialize Snok agent instructions in the current repository.",
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			result, err := initAgentsFile(ctx.CWD, os.Stdin, os.Stdout)
			if err != nil {
				return nil, err
			}
			ctx.Output.Log("%s", result)
			return nil, nil
		},
	}))
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

const snokAgentsHeading = "## Snok Command Authoring"

func initAgentsFile(dir string, in io.Reader, promptOut io.Writer) (string, error) {
	path := filepath.Join(dir, "AGENTS.md")
	content, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if err := os.WriteFile(path, []byte(renderAgentsFile()), 0o644); err != nil {
			return "", err
		}
		return "created AGENTS.md", nil
	}
	if hasSnokAgentsSection(string(content)) {
		return "AGENTS.md already includes Snok command-authoring instructions", nil
	}
	if !promptAppendAgents(in, promptOut) {
		return "left AGENTS.md unchanged", nil
	}
	updated := appendSnokAgentsSection(string(content))
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return "updated AGENTS.md", nil
}

func promptAppendAgents(in io.Reader, out io.Writer) bool {
	if out != nil {
		fmt.Fprint(out, "AGENTS.md already exists. Append Snok command-authoring instructions? [y/N] ")
	}
	if in == nil {
		return false
	}
	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && answer != "") {
		return false
	}
	return isYes(answer)
}

func isYes(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func hasSnokAgentsSection(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == snokAgentsHeading {
			return true
		}
	}
	return false
}

func renderAgentsFile() string {
	return strings.TrimPrefix(`# AGENTS.md

Instructions for coding agents working in this repository.

`+renderSnokAgentsSection(), "\n")
}

func appendSnokAgentsSection(content string) string {
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return renderAgentsFile()
	}
	return trimmed + "\n\n" + renderSnokAgentsSection()
}

func renderSnokAgentsSection() string {
	return `## Snok Command Authoring

- Before editing commands, inspect the app's existing ` + "`snok.NewTree`" + ` setup and follow the local registration pattern.
- Use ` + "`AddGroup`" + ` for non-executable namespaces and ` + "`AddCommand`" + ` for executable routes.
- Define positional arguments and options with ` + "`snok.Field`" + `, including descriptions for user-facing help.
- Use typed context accessors such as ` + "`ctx.ArgString`" + `, ` + "`ctx.OptionBool`" + `, ` + "`ctx.OptionNumber`" + `, and ` + "`ctx.OptionInt`" + ` instead of reparsing argv.
- Use ` + "`ctx.Output.Log`" + ` for human stdout and ` + "`ctx.Output.Error`" + ` for human stderr.
- Return ` + "`*snok.CommandError`" + ` for predictable user-facing failures that need stable kind, message, hint, details, or exit code fields.
- Test focused command behavior with ` + "`snok.RunCommand`" + `; add integration coverage when registration, routing, help text, or Cobra wiring changes.
`
}
