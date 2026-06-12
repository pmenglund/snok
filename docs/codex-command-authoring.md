# Codex Command Authoring Guide

Use this guide when adding or restructuring commands in a Go CLI built with
`github.com/pmenglund/snok`. Keep the app's existing shape unless there is a
clear reason to change it.

## Read First

Before editing, inspect the local app:

- Find where the app creates its command tree with `snok.NewTree`.
- Find the existing command registration pattern and follow it.
- Check whether commands are grouped with `AddGroup` or registered directly
  with `AddCommand`.
- Read nearby tests that use `snok.RunCommand`, `go run`, or the app binary.
- Check examples and docs that mention the command surface you are changing.

If the app has no established structure yet, use the structure below.

## Default Structure

Prefer one registration function per command file:

```go
func registerGreetCommand(tree *snok.Tree) error {
	return tree.AddCommand([]string{"greet"}, snok.CommandDefinition{
		Description: "Print a greeting.",
		Args: []snok.Field{
			{Name: "name", Kind: snok.FieldPrimitive, Type: snok.TypeString, Required: true},
		},
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			ctx.Output.Log("Hello, %s!", ctx.ArgString("name"))
			return nil, nil
		},
	})
}
```

Keep tree construction central:

```go
func newCommand() interface{ Execute() error } {
	tree := snok.NewTree(snok.Config{
		Name:    "my-cli",
		Version: "0.1.0",
	})
	must(registerGreetCommand(tree))
	return snok.CobraCommand(tree)
}
```

For larger apps, collect registration functions and apply them from the central
builder. The `examples/modular` package shows this pattern with `init`,
`registerCommand`, and one self-contained command file.

Use `AddGroup` for non-executable namespaces that organize child commands. Use
`AddCommand` for executable routes. A command path such as
`[]string{"project", "create"}` may rely on Snok to create missing parent groups,
but explicit `AddGroup` calls are clearer when the group has a description,
aliases, examples, or many child commands.

## Command Rules

- Choose stable route segments. They become user-facing command names.
- Add aliases only when they are intentional public API, not as temporary
  compatibility guesses.
- Declare positional arguments and options with `snok.Field`; include
  descriptions for user-facing help.
- Use `snok.FieldPrimitive` with `snok.TypeString`, `snok.TypeNumber`, or
  `snok.TypeBoolean` unless the app already uses enum or schema fields.
- Use typed context accessors such as `ctx.ArgString`, `ctx.OptionBool`,
  `ctx.OptionNumber`, and `ctx.OptionInt` instead of reparsing raw argv.
- Use `ctx.Output.Log` for human stdout and `ctx.Output.Error` for human stderr.
- Read stdin through `ctx.Stdin.Text()` or `ctx.Stdin.Bytes()` when needed; stdin
  is single-use.
- For JSON commands, set `CommandDefinition.JSON` and return the document value
  from `Run`. Do not manually print JSON through `ctx.Output.Log`.
- For JSON Lines commands, set `CommandDefinition.JSONL` and return
  `snok.JSONLStream` or records for Snok to serialize.
- Return `*snok.CommandError` for predictable failures that need stable
  `Kind`, `Message`, `Hint`, `Details`, or `ExitCode` fields.

## Testing

Use `snok.RunCommand` for focused command behavior tests:

```go
result := snok.RunCommand(context.Background(), def, snok.ExecuteOptions{
	Argv: []string{"Ada", "--shout"},
})
if result.ExitCode != 0 {
	t.Fatalf("command failed: %v", result.Error)
}
```

Cover the behavior the command owns:

- successful text output;
- parse errors for missing args, bad option values, unknown options, and
  unexpected args;
- defaults and environment fallbacks when fields use `Default` or `Env`;
- JSON or JSONL output mode when enabled;
- structured error fields for expected user-facing failures.

Add or update integration tests when registration, routing, help text, or Cobra
wiring changes. Example tests may run `go run ./examples/...`; app tests may run
the app binary or command package.

## Documentation

Keep user-facing docs practical and aligned with real commands. If a change
adds or changes public command behavior, update the app's README or examples as
needed. When working in this repository, keep `README.md`, `examples/`, and
`cmd/snok` aligned as required by `AGENTS.md`.
