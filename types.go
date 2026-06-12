package snok

import (
	"context"
	"io"
)

type FieldKind string

const (
	FieldPrimitive FieldKind = "primitive"
	FieldEnum      FieldKind = "enum"
	FieldSchema    FieldKind = "schema"
)

type PrimitiveType string

const (
	TypeString  PrimitiveType = "string"
	TypeNumber  PrimitiveType = "number"
	TypeBoolean PrimitiveType = "boolean"
)

// Field declares either a positional argument or an option.
type Field struct {
	Name        string
	Description string
	Kind        FieldKind
	Type        PrimitiveType
	Values      []any
	Required    bool
	Default     any
	Short       string
	Env         string
	Multiple    bool
	Flag        bool
	Validator   Validator
	TypeHint    string
}

type ValidationInput struct {
	Raw       string
	RawValues []string
	Present   bool
	Flag      bool
	Omitted   bool
}

type Validator func(ValidationInput) (any, error)

type RunFunc func(context.Context, *Context) (any, error)

type HelpRenderer func(HelpData) (string, error)

type CommandDefinition struct {
	Description string
	Aliases     []string
	Examples    []string
	Args        []Field
	Options     []Field
	JSON        bool
	JSONL       bool
	Help        HelpRenderer
	Run         RunFunc
}

type GroupDefinition struct {
	Description string
	Aliases     []string
	Examples    []string
}

type Config struct {
	Name            string
	Version         string
	Help            HelpRenderer
	Options         []Field
	Hooks           Hooks
	Locals          LocalsFunc
	AutoJSON        func(Invocation) bool
	DisableAutoJSON bool
}

type Hooks struct {
	BeforeRun  func(context.Context, *Context) error
	AfterRun   func(context.Context, *Context, OutputResult) error
	OnRunError func(context.Context, *Context, *CommandError) error
}

type LocalsFunc func(context.Context, Invocation) (map[string]any, error)

type Invocation struct {
	Argv []string
	CWD  string
	Env  map[string]string
}

type Context struct {
	Args    map[string]any
	Options map[string]any
	Locals  map[string]any
	CWD     string
	RawArgs []string
	Output  *Output
	Stdin   *Stdin
}

type ExecuteOptions struct {
	Argv   []string
	Env    map[string]string
	CWD    string
	Stdin  io.Reader
	Piped  bool
	Config Config
}

type RunResult struct {
	ExitCode  int
	Stdout    string
	Stderr    string
	Error     *CommandError
	Output    OutputResult
	JSONMode  bool
	JSONLMode bool
}

type OutputKind string

const (
	OutputText  OutputKind = "text"
	OutputJSON  OutputKind = "json"
	OutputJSONL OutputKind = "jsonl"
)

type OutputResult struct {
	Kind     OutputKind
	Document any
	Records  []any
}

type JSONLStream struct {
	Records []any
	Err     error
}

func DefineCommand(def CommandDefinition) CommandDefinition {
	return cloneCommand(def)
}

func DefineGroup(def GroupDefinition) GroupDefinition {
	return cloneGroup(def)
}

func DefineConfig(config Config) Config {
	copy := config
	copy.Options = cloneFields(config.Options)
	return copy
}

func cloneCommand(def CommandDefinition) CommandDefinition {
	copy := def
	copy.Aliases = append([]string(nil), def.Aliases...)
	copy.Examples = append([]string(nil), def.Examples...)
	copy.Args = cloneFields(def.Args)
	copy.Options = cloneFields(def.Options)
	return copy
}

func cloneGroup(def GroupDefinition) GroupDefinition {
	copy := def
	copy.Aliases = append([]string(nil), def.Aliases...)
	copy.Examples = append([]string(nil), def.Examples...)
	return copy
}

func cloneFields(fields []Field) []Field {
	out := make([]Field, len(fields))
	for i, field := range fields {
		out[i] = field
		out[i].Values = append([]any(nil), field.Values...)
	}
	return out
}
