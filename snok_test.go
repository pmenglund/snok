package snok

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestRoutingFixtures(t *testing.T) {
	tree := NewTree(Config{Name: "my-cli"})
	must(t, tree.AddGroup(nil, GroupDefinition{}))
	must(t, tree.AddCommand([]string{"hello"}, CommandDefinition{Description: "Say hello"}))
	must(t, tree.AddCommand([]string{"project"}, CommandDefinition{Description: "Project commands"}))
	must(t, tree.AddCommand([]string{"project", "create"}, CommandDefinition{Description: "Create a project"}))
	must(t, tree.AddCommand([]string{"project", "list"}, CommandDefinition{Description: "List projects"}))
	must(t, tree.AddGroup([]string{"user"}, GroupDefinition{}))
	must(t, tree.AddCommand([]string{"user", "delete"}, CommandDefinition{Description: "Delete a user"}))

	route := tree.Resolve([]string{"project", "create", "--help"})
	assertEqual(t, route.Kind, RouteCommand)
	assertEqual(t, route.MatchedPath, []string{"project", "create"})
	assertEqual(t, route.RemainingArgs, []string{"--help"})
	assertEqual(t, route.HelpRequested, true)

	aliasTree := NewTree(Config{})
	must(t, aliasTree.AddGroup([]string{"project"}, GroupDefinition{Aliases: []string{"p"}}))
	must(t, aliasTree.AddCommand([]string{"project", "create"}, CommandDefinition{Aliases: []string{"c"}}))
	route = aliasTree.Resolve([]string{"p", "c"})
	assertEqual(t, route.Kind, RouteCommand)
	assertEqual(t, route.MatchedPath, []string{"project", "create"})

	route = tree.Resolve([]string{"project", "create", "--", "--help"})
	assertEqual(t, route.HelpRequested, false)
	assertEqual(t, route.RemainingArgs, []string{"--", "--help"})

	unknownTree := NewTree(Config{})
	must(t, unknownTree.AddGroup([]string{"project"}, GroupDefinition{}))
	must(t, unknownTree.AddCommand([]string{"project", "create"}, CommandDefinition{Aliases: []string{"c"}}))
	must(t, unknownTree.AddCommand([]string{"project", "list"}, CommandDefinition{}))
	must(t, unknownTree.AddGroup([]string{"user"}, GroupDefinition{}))
	must(t, unknownTree.AddCommand([]string{"user", "delete"}, CommandDefinition{}))
	route = unknownTree.Resolve([]string{"project", "cretae"})
	assertEqual(t, route.Kind, RouteUnknown)
	assertEqual(t, route.MatchedPath, []string{"project"})
	assertEqual(t, route.AttemptedPath, []string{"project", "cretae"})
	assertEqual(t, route.AvailableChildNames, []string{"create", "list"})
	assertEqual(t, route.Suggestions, []string{"create"})
}

func TestDerivePublicTreeIgnoresPrivateAndTests(t *testing.T) {
	tree := DerivePublicTree([]string{
		"commands/deploy",
		"commands/deploy.test",
		"commands/_deploy-logic",
		"commands/project/_group",
		"commands/project/_schema",
		"commands/project/create",
		"commands/project/create.spec",
	})
	assertEqual(t, tree.Commands, [][]string{{"deploy"}, {"project", "create"}})
	assertEqual(t, tree.Groups, [][]string{{"project"}})
}

func TestParseFixtures(t *testing.T) {
	def := CommandDefinition{
		Options: []Field{
			{Name: "name", Kind: FieldPrimitive, Type: TypeString, Required: true},
			{Name: "force", Kind: FieldPrimitive, Type: TypeBoolean, Short: "f"},
			{Name: "count", Kind: FieldPrimitive, Type: TypeNumber, Default: float64(1)},
		},
		Args: []Field{{Name: "id", Kind: FieldPrimitive, Type: TypeString, Required: true}},
	}
	result := ParseCommand(def, []string{"123", "--name", "snok", "-f"}, nil, nil)
	assertOK(t, result)
	assertEqual(t, result.Args, map[string]any{"id": "123"})
	assertEqual(t, result.Options, map[string]any{"name": "snok", "force": true, "count": float64(1)})
	assertEqual(t, result.RawArgs, []string{"123", "--name", "snok", "-f"})

	def = CommandDefinition{Options: []Field{{Name: "port", Kind: FieldPrimitive, Type: TypeNumber, Env: "PORT", Default: float64(3000)}}}
	assertOption(t, ParseCommand(def, []string{"--port", "4000"}, map[string]string{"PORT": "5000"}, nil), "port", float64(4000))
	assertOption(t, ParseCommand(def, nil, map[string]string{"PORT": "5000"}, nil), "port", float64(5000))
	assertOption(t, ParseCommand(def, nil, nil, nil), "port", float64(3000))
	assertParseContains(t, ParseCommand(def, nil, map[string]string{"PORT": "bad"}, nil), "Invalid value for option --port", "environment variable PORT", "Expected number")

	def = CommandDefinition{Options: []Field{
		{Name: "tag", Kind: FieldPrimitive, Type: TypeString, Multiple: true},
		{Name: "level", Kind: FieldPrimitive, Type: TypeNumber, Multiple: true, Short: "l"},
	}}
	result = ParseCommand(def, []string{"--tag", "alpha", "--tag", "beta", "-l", "1", "-l", "2"}, nil, nil)
	assertOK(t, result)
	assertEqual(t, result.Options["tag"], []any{"alpha", "beta"})
	assertEqual(t, result.Options["level"], []any{float64(1), float64(2)})

	def = CommandDefinition{Options: []Field{
		{Name: "mode", Kind: FieldEnum, Values: []any{"dev", "prod"}, Default: "dev"},
		{Name: "shard", Kind: FieldEnum, Values: []any{float64(1), float64(2)}, Required: true},
	}}
	assertOption(t, ParseCommand(def, []string{"--shard", "1"}, nil, nil), "shard", float64(1))
	assertParseContains(t, ParseCommand(def, []string{"--shard", "1.0"}, nil, nil), "Expected one of", "1, 2")

	def = CommandDefinition{Options: []Field{{Name: "color", Kind: FieldPrimitive, Type: TypeBoolean, Default: true}}}
	assertOption(t, ParseCommand(def, nil, nil, nil), "color", true)
	assertOption(t, ParseCommand(def, []string{"--no-color"}, nil, nil), "color", false)
	assertParseContains(t, ParseCommand(def, []string{"--color", "--no-color"}, nil, nil), "Conflicting options", "--color", "--no-color")

	def = CommandDefinition{
		Options: []Field{
			{Name: "count", Kind: FieldSchema, Validator: positiveInteger},
			{Name: "verbose", Kind: FieldSchema, Flag: true, Validator: optionalBoolean},
		},
		Args: []Field{{Name: "id", Kind: FieldSchema, Validator: uuidString}},
	}
	result = ParseCommand(def, []string{"550e8400-e29b-41d4-a716-446655440000", "--count", "2", "--verbose"}, nil, nil)
	assertOK(t, result)
	assertEqual(t, result.Args["id"], "550e8400-e29b-41d4-a716-446655440000")
	assertEqual(t, result.Options["count"], 2)
	assertEqual(t, result.Options["verbose"], true)

	def = CommandDefinition{
		Options: []Field{{Name: "name", Kind: FieldPrimitive, Type: TypeString, Required: true}, {Name: "force", Kind: FieldPrimitive, Type: TypeBoolean}},
		Args:    []Field{{Name: "id", Kind: FieldPrimitive, Type: TypeString, Required: true}},
	}
	assertParseContains(t, ParseCommand(def, []string{"123"}, nil, nil), "Missing required option", "--name")
	assertParseContains(t, ParseCommand(def, []string{"123", "--name", "snok", "--name", "again"}, nil, nil), "Duplicate option", "--name")
	assertParseContains(t, ParseCommand(def, []string{"123", "--name", "snok", "--unknown"}, nil, nil), "Unknown option")
	assertParseContains(t, ParseCommand(def, []string{"123", "extra", "--name", "snok"}, nil, nil), "Unexpected argument", "extra")
}

func TestAddCommandRejectsInvalidFieldDefinitions(t *testing.T) {
	optionCases := []struct {
		name  string
		field Field
		want  string
	}{
		{
			name:  "missing primitive type",
			field: Field{Name: "value", Kind: FieldPrimitive},
			want:  "primitive type",
		},
		{
			name:  "unknown kind",
			field: Field{Name: "value", Kind: FieldKind("wat")},
			want:  "kind",
		},
		{
			name:  "number default has wrong type",
			field: Field{Name: "count", Kind: FieldPrimitive, Type: TypeNumber, Default: "oops"},
			want:  "expected finite number",
		},
		{
			name:  "required boolean option",
			field: Field{Name: "force", Kind: FieldPrimitive, Type: TypeBoolean, Required: true},
			want:  "must not be required",
		},
		{
			name:  "enum without values",
			field: Field{Name: "mode", Kind: FieldEnum},
			want:  "at least one value",
		},
		{
			name:  "enum with non scalar value",
			field: Field{Name: "mode", Kind: FieldEnum, Values: []any{true}},
			want:  "non-string/non-number",
		},
		{
			name:  "enum default outside values",
			field: Field{Name: "mode", Kind: FieldEnum, Values: []any{"dev"}, Default: "prod"},
			want:  "invalid default",
		},
		{
			name:  "schema without validator",
			field: Field{Name: "count", Kind: FieldSchema},
			want:  "must define a validator",
		},
		{
			name:  "schema with default",
			field: Field{Name: "count", Kind: FieldSchema, Validator: positiveInteger, Default: 1},
			want:  "must not define a default",
		},
		{
			name:  "repeatable with default",
			field: Field{Name: "tag", Kind: FieldPrimitive, Type: TypeString, Multiple: true, Default: []string{"a"}},
			want:  "must not have a default",
		},
	}
	for _, tc := range optionCases {
		t.Run(tc.name, func(t *testing.T) {
			tree := NewTree(Config{})
			err := tree.AddCommand([]string{"cmd"}, CommandDefinition{Options: []Field{tc.field}})
			if err == nil {
				t.Fatal("expected AddCommand to reject invalid option")
			}
			assertContains(t, err.Error(), tc.want)
		})
	}

	argCases := []struct {
		name string
		args []Field
		want string
	}{
		{
			name: "argument with option short",
			args: []Field{{Name: "id", Kind: FieldPrimitive, Type: TypeString, Short: "i"}},
			want: "must not define a short option",
		},
		{
			name: "repeatable argument",
			args: []Field{{Name: "id", Kind: FieldPrimitive, Type: TypeString, Multiple: true}},
			want: "must not be repeatable",
		},
		{
			name: "required after defaulted optional",
			args: []Field{
				{Name: "first", Kind: FieldPrimitive, Type: TypeString, Default: "x"},
				{Name: "second", Kind: FieldPrimitive, Type: TypeString, Required: true},
			},
			want: "follows optional",
		},
	}
	for _, tc := range argCases {
		t.Run(tc.name, func(t *testing.T) {
			tree := NewTree(Config{})
			err := tree.AddCommand([]string{"cmd"}, CommandDefinition{Args: tc.args})
			if err == nil {
				t.Fatal("expected AddCommand to reject invalid argument")
			}
			assertContains(t, err.Error(), tc.want)
		})
	}
}

func TestExecutionOutputFixtures(t *testing.T) {
	def := CommandDefinition{
		Options: []Field{{Name: "name", Kind: FieldPrimitive, Type: TypeString, Required: true}},
		Args:    []Field{{Name: "id", Kind: FieldPrimitive, Type: TypeString, Required: true}},
		Run: func(_ context.Context, ctx *Context) (any, error) {
			id := ctx.ArgString("id")
			name := ctx.OptionString("name")
			text, err := ctx.Stdin.Text()
			if err != nil {
				return nil, err
			}
			ctx.Output.Log("id=%s name=%s cwd=%s raw=%s stdin=%s", id, name, ctx.CWD, strings.Join(ctx.RawArgs, ","), strings.TrimSpace(text))
			return nil, nil
		},
	}
	result := RunCommand(context.Background(), def, ExecuteOptions{
		Argv:  []string{"123", "--name", "snok"},
		CWD:   "/workspace/project",
		Stdin: strings.NewReader("hello\n"),
		Piped: true,
	})
	assertRunOK(t, result)
	assertEqual(t, result.Stdout, "id=123 name=snok cwd=/workspace/project raw=123,--name,snok stdin=hello\n")
	assertEqual(t, result.Stderr, "")
	assertEqual(t, result.Output.Kind, OutputText)

	def = CommandDefinition{
		JSON: true,
		Run: func(_ context.Context, ctx *Context) (any, error) {
			ctx.Output.Log("suppressed")
			ctx.Output.Error("diagnostic")
			return map[string]any{"items": []any{float64(1), float64(2), float64(3)}}, nil
		},
	}
	result = RunCommand(context.Background(), def, ExecuteOptions{Argv: []string{"--json"}})
	assertRunOK(t, result)
	assertEqual(t, result.Stdout, "")
	assertEqual(t, result.Stderr, "diagnostic\n")
	assertEqual(t, result.JSONMode, true)
	assertEqual(t, result.Output.Kind, OutputJSON)

	result = RunCommand(context.Background(), CommandDefinition{
		JSON: true,
		Args: []Field{{Name: "rest", Kind: FieldPrimitive, Type: TypeString}},
		Run: func(_ context.Context, ctx *Context) (any, error) {
			return ctx.RawArgs, nil
		},
	}, ExecuteOptions{Argv: []string{"--", "--json"}})
	assertRunOK(t, result)
	assertEqual(t, result.JSONMode, false)

	def = CommandDefinition{JSONL: true, Run: func(context.Context, *Context) (any, error) {
		return []any{map[string]any{"id": 1}, map[string]any{"id": 2}}, nil
	}}
	result = RunCommand(context.Background(), def, ExecuteOptions{})
	assertRunOK(t, result)
	assertEqual(t, result.Stdout, "{\"id\":1}\n{\"id\":2}\n")
	assertEqual(t, result.Output.Kind, OutputJSONL)

	def = CommandDefinition{JSONL: true, Run: func(context.Context, *Context) (any, error) {
		return JSONLStream{Records: []any{map[string]any{"id": 1}}, Err: &CommandError{Kind: "app/failed", Message: "stream failed", ExitCode: 4}}, nil
	}}
	result = RunCommand(context.Background(), def, ExecuteOptions{})
	assertEqual(t, result.ExitCode, 4)
	assertEqual(t, result.Stdout, "{\"id\":1}\n")
	assertEqual(t, result.Output.Records, []any{map[string]any{"id": 1}})

	def = CommandDefinition{JSON: true, Run: func(context.Context, *Context) (any, error) {
		return nil, &CommandError{Kind: "config/not-found", Message: "Config file was not found", Hint: "Create a config file", Details: map[string]any{"path": "snok.config"}, ExitCode: 7}
	}}
	result = RunCommand(context.Background(), def, ExecuteOptions{Argv: []string{"--json"}})
	assertEqual(t, result.ExitCode, 7)
	assertEqual(t, result.Error.Kind, "config/not-found")
	assertContains(t, result.Stderr, `"kind":"config/not-found"`)

	order := []string{}
	config := Config{
		Options: []Field{{Name: "profile", Kind: FieldPrimitive, Type: TypeString, Default: "prod"}},
		Locals: func(context.Context, Invocation) (map[string]any, error) {
			order = append(order, "locals")
			return map[string]any{"workspace": "prod"}, nil
		},
		Hooks: Hooks{
			BeforeRun: func(context.Context, *Context) error {
				order = append(order, "beforeRun")
				return nil
			},
			AfterRun: func(context.Context, *Context, OutputResult) error {
				order = append(order, "afterRun")
				return nil
			},
			OnRunError: func(context.Context, *Context, *CommandError) error {
				order = append(order, "onRunError")
				return nil
			},
		},
	}
	def = CommandDefinition{Run: func(context.Context, *Context) (any, error) {
		order = append(order, "run")
		return nil, nil
	}}
	order = append(order, "parse")
	result = RunCommand(context.Background(), def, ExecuteOptions{Config: config})
	assertRunOK(t, result)
	assertEqual(t, order, []string{"parse", "locals", "beforeRun", "run", "afterRun"})

	def = CommandDefinition{Run: func(context.Context, *Context) (any, error) {
		_, err := NewStdin(strings.NewReader("hello\n"), true).Text()
		return nil, err
	}}
	stdin := NewStdin(strings.NewReader("hello\n"), true)
	text, err := stdin.Text()
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, text, "hello\n")
	_, err = stdin.Bytes()
	var commandErr *CommandError
	if !errors.As(err, &commandErr) || commandErr.Kind != "rune/stdin-consumed" {
		t.Fatalf("expected stdin consumed error, got %#v", err)
	}
}

func TestHelpFixtures(t *testing.T) {
	text := RenderHelp(HelpData{
		Kind:        HelpCommand,
		CLIName:     "my-cli",
		Path:        []string{"text", "transform"},
		Description: "Transform the input text",
		Options: []HelpOption{
			{Name: "mode", TypeHint: "upper|lower", Description: "Transformation to apply", DefaultLabel: "\"upper\""},
			{Name: "force", Short: "f", TypeHint: "boolean"},
		},
		Args:     []HelpArg{{Name: "input", TypeHint: "string", Required: true}},
		Examples: []string{"my-cli text transform hello --mode upper"},
	})
	assertInOrder(t, text, "Transform the input text", "Usage:", "Options:", "Arguments:", "Examples:")
	for _, want := range []string{"my-cli text transform", "--mode", "upper|lower", "Transformation to apply", "--force", "-f", "input", "my-cli text transform hello --mode upper"} {
		assertContains(t, text, want)
	}

	text = RenderHelp(HelpData{
		Kind:        HelpGroup,
		CLIName:     "my-cli",
		Path:        []string{"project"},
		Description: "Manage projects",
		Children: []HelpChild{
			{Name: "create", Aliases: []string{"c"}, Description: "Create a project"},
			{Name: "list", Description: "List projects"},
		},
		Examples: []string{"my-cli project list"},
	})
	assertInOrder(t, text, "Manage projects", "Usage:", "Commands:", "Examples:")
	for _, want := range []string{"my-cli project", "create", "c", "Create a project", "list", "List projects", "my-cli project list"} {
		assertContains(t, text, want)
	}

	text = RenderHelp(HelpData{
		Kind:                HelpUnknown,
		CLIName:             "my-cli",
		AttemptedPath:       []string{"project", "cretae"},
		AvailableChildNames: []string{"create", "list"},
		Suggestions:         []string{"create"},
	})
	for _, want := range []string{"Unknown command", "project cretae", "create", "list"} {
		assertContains(t, text, want)
	}
	if strings.Contains(text, "user delete") {
		t.Fatalf("unknown help leaked unrelated command: %s", text)
	}

	tree := NewTree(Config{Help: func(HelpData) (string, error) { return "global help", nil }})
	must(t, tree.AddCommand([]string{"cmd"}, CommandDefinition{Help: func(HelpData) (string, error) { return "command help", nil }}))
	assertEqual(t, RenderResolvedHelp(tree, tree.Resolve([]string{"cmd", "--help"})), "command help")
	assertEqual(t, RenderResolvedHelp(tree, tree.Resolve([]string{"--help"})), "global help")

	tree = NewTree(Config{})
	must(t, tree.AddCommand([]string{"cmd"}, CommandDefinition{Help: func(HelpData) (string, error) { return "", errors.New("boom") }}))
	assertContains(t, RenderResolvedHelp(tree, tree.Resolve([]string{"cmd", "--help"})), "Usage:")
}

func TestContextTypedHelpers(t *testing.T) {
	ctx := &Context{
		Args: map[string]any{
			"name":  "Ada",
			"count": float64(3),
		},
		Options: map[string]any{
			"json":  true,
			"limit": "5",
		},
	}

	name := ctx.ArgString("name")
	assertEqual(t, name, "Ada")

	count := ctx.ArgInt("count")
	assertEqual(t, count, 3)

	limit := ctx.OptionNumber("limit")
	assertEqual(t, limit, float64(5))

	jsonMode := ctx.OptionBool("json")
	assertEqual(t, jsonMode, true)

	assertContextValuePanic(t, func() { _ = ctx.ArgString("count") })
}

func TestRunCommandRecoversContextAccessorPanics(t *testing.T) {
	result := RunCommand(context.Background(), CommandDefinition{
		Run: func(_ context.Context, ctx *Context) (any, error) {
			_ = ctx.ArgString("missing")
			return nil, nil
		},
	}, ExecuteOptions{})

	if result.ExitCode != 1 || result.Error == nil {
		t.Fatalf("expected structured failure, got %#v", result)
	}
	assertEqual(t, result.Error.Kind, "rune/invalid-context-value")
	assertContains(t, result.Error.Message, `argument "missing"`)
}

func positiveInteger(input ValidationInput) (any, error) {
	if input.Omitted {
		return nil, nil
	}
	if input.Raw == "2" {
		return 2, nil
	}
	return nil, errors.New("expected positive integer")
}

func optionalBoolean(input ValidationInput) (any, error) {
	if input.Omitted {
		return nil, nil
	}
	return input.Present, nil
}

func uuidString(input ValidationInput) (any, error) {
	if strings.Count(input.Raw, "-") == 4 {
		return input.Raw, nil
	}
	return nil, errors.New("expected uuid")
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func assertOK(t *testing.T, result ParseResult) {
	t.Helper()
	if !result.OK {
		t.Fatalf("parse failed: %#v", result.Error)
	}
}

func assertRunOK(t *testing.T, result RunResult) {
	t.Helper()
	if result.ExitCode != 0 || result.Error != nil {
		t.Fatalf("run failed: exit=%d err=%#v stderr=%q", result.ExitCode, result.Error, result.Stderr)
	}
}

func assertOption(t *testing.T, result ParseResult, name string, value any) {
	t.Helper()
	assertOK(t, result)
	assertEqual(t, result.Options[name], value)
}

func assertParseContains(t *testing.T, result ParseResult, parts ...string) {
	t.Helper()
	if result.OK || result.Error == nil {
		t.Fatalf("expected parse failure, got %#v", result)
	}
	for _, part := range parts {
		assertContains(t, result.Error.Message, part)
	}
}

func assertContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q to contain %q", text, want)
	}
}

func assertInOrder(t *testing.T, text string, parts ...string) {
	t.Helper()
	offset := 0
	for _, part := range parts {
		index := strings.Index(text[offset:], part)
		if index < 0 {
			t.Fatalf("expected %q after offset %d in %q", part, offset, text)
		}
		offset += index + len(part)
	}
}

func assertContextValuePanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		var commandErr *CommandError
		if !errors.As(asError(recovered), &commandErr) || commandErr.Kind != "rune/invalid-context-value" {
			t.Fatalf("expected invalid context value panic, got %#v", recovered)
		}
	}()
	fn()
}

func asError(value any) error {
	if err, ok := value.(error); ok {
		return err
	}
	return fmt.Errorf("%v", value)
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
