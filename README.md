# snok

Snok is a Go CLI framework layer built on top of Cobra. It gives command-line
tools a visible command tree, declarative command metadata, predictable parsing,
structured output modes, and in-process test helpers that are easy for both
humans and coding agents to inspect.

Snok has two surfaces:

- A Go package, `github.com/pmenglund/snok`, for building Cobra-backed CLIs.
- A `snok` executable for hot-running linked Go command packages during local
  development.

## Install

Add Snok to a Go module:

```sh
go get github.com/pmenglund/snok
```

Install the `snok` CLI from this repository:

```sh
go install ./cmd/snok
```

Or install it from the module path:

```sh
go install github.com/pmenglund/snok/cmd/snok@latest
```

Then make sure your Go install directory, usually `$(go env GOPATH)/bin`, is on
`PATH` and run:

```sh
snok --help
```

## Build a CLI

Create a command tree, add commands or groups, then expose the tree as a Cobra
command:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/pmenglund/snok"
)

func main() {
	tree := snok.NewTree(snok.Config{
		Name:    "hello",
		Version: "0.1.0",
	})

	err := tree.AddCommand([]string{"greet"}, snok.CommandDefinition{
		Description: "Print a greeting.",
		Examples: []string{
			"hello greet Ada --title Dr",
		},
		Args: []snok.Field{
			{
				Name:        "name",
				Kind:        snok.FieldPrimitive,
				Type:        snok.TypeString,
				Required:    true,
				Description: "Person to greet",
			},
		},
		Options: []snok.Field{
			{
				Name:        "title",
				Kind:        snok.FieldPrimitive,
				Type:        snok.TypeString,
				Default:     "",
				Description: "Optional title",
			},
		},
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			name := ctx.ArgString("name")
			if title := ctx.OptionString("title"); title != "" {
				name = title + " " + name
			}
			ctx.Output.Log("Hello, %s!", name)
			return nil, nil
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := snok.CobraCommand(tree).Execute(); err != nil {
		var commandErr *snok.CommandError
		if errors.As(err, &commandErr) {
			fmt.Fprintln(os.Stderr, commandErr.Message)
			os.Exit(commandErr.ExitCode)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## Core Concepts

- `Tree` is the public command tree. Use `NewTree` with `Config`, then add
  commands and groups with `AddCommand` and `AddGroup`.
- `CommandDefinition` describes one executable command: description, aliases,
  examples, positional args, options, output mode, help renderer, and run
  function.
- `GroupDefinition` describes a non-executable command group that can contain
  child commands.
- `Field` declares a positional arg or option. Snok supports primitive string,
  number, and boolean fields; enum fields; schema-backed validator fields;
  defaults; environment fallbacks; repeatable options; and short flags.
- `Context` gives command logic typed access to parsed args and options, cwd,
  raw argv, locals, stdin, and framework-owned stdout/stderr through
  `ctx.Output`.
- `CommandError` is the structured error type Snok uses for predictable
  failures and exit codes.
- `RunCommand` executes one command in process for tests with injected argv,
  env, cwd, stdin, globals, hooks, and locals.
- `CobraCommand` exposes the Snok tree through Cobra while keeping Snok routing,
  parsing, help, and execution behavior as the source of truth.

See `SPEC.md` for the full behavior contract and `specs/fixtures/*.json` for
portable conformance scenarios.

## Output Modes

Commands are text-first by default. Use `ctx.Output.Log` for human stdout and
`ctx.Output.Error` for human stderr.

Set `CommandDefinition.JSON` to enable Snok's framework-managed `--json` mode.
When JSON mode is active, `ctx.Output.Log` is suppressed and the command return
value becomes the JSON document.

Set `CommandDefinition.JSONL` for JSON Lines commands. JSONL commands return a
`snok.JSONLStream` or records that Snok serializes as one compact JSON document
per line.

## Hooks, Locals, and Stdin

`Config` can define:

- Global options available to every executable command.
- `Locals`, a per-invocation factory for runtime values.
- `Hooks.BeforeRun`, `Hooks.AfterRun`, and `Hooks.OnRunError`.
- `AutoJSON` and `DisableAutoJSON` for agent-aware JSON activation.

Command logic can read stdin through `ctx.Stdin.Text()` or `ctx.Stdin.Bytes()`.
Stdin is intentionally single-use; a second read returns a structured
`rune/stdin-consumed` error.

## Examples

Run the checked-in examples from this repository:

```sh
go run ./examples/text greet Ada
go run ./examples/text greet Ada --title Dr --shout
go run ./examples/json sum 2 3 --json
go run ./examples/jsonl events --count 3
```

## The `snok` CLI

The repository also includes a small `snok` executable that can link command
names to local Go package directories. Linked commands are symlinks to the
`snok` launcher. When invoked through a linked name, the launcher rebuilds the
source package when its non-test Go files change and then execs the cached
binary.

List the CLI commands:

```sh
snok --help
```

Link a local Go command package:

```sh
snok link my-command ./path/to/package
```

Then add the reported Snok bin directory to `PATH` if needed and run:

```sh
my-command
```

List and remove links:

```sh
snok links
snok unlink my-command
```

By default, link metadata and cached builds live under `~/.snok`. Set
`SNOK_HOME` to use a different directory.

## Development

Run the full test suite:

```sh
go test ./...
```

Run focused tests while working:

```sh
go test .
go test ./internal/hotrun
go test ./cmd/snok
```

Keep documentation examples aligned with `examples/`, `cmd/snok`, and the
public API in this repository.
