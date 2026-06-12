package snok

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Output struct {
	stdout      *bytes.Buffer
	stderr      *bytes.Buffer
	suppressLog bool
}

func newOutput(suppressLog bool) *Output {
	return &Output{
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
		suppressLog: suppressLog,
	}
}

func (o *Output) Log(format string, args ...any) {
	if o.suppressLog {
		return
	}
	fmt.Fprintln(o.stdout, fmt.Sprintf(format, args...))
}

func (o *Output) Error(format string, args ...any) {
	fmt.Fprintln(o.stderr, fmt.Sprintf(format, args...))
}

type Stdin struct {
	reader      io.Reader
	consumed    bool
	piped       bool
	interactive bool
}

func NewStdin(reader io.Reader, piped bool) *Stdin {
	if reader == nil {
		reader = strings.NewReader("")
	}
	return &Stdin{
		reader:      reader,
		piped:       piped,
		interactive: !piped,
	}
}

func (s *Stdin) IsPiped() bool {
	return s.piped
}

func (s *Stdin) IsInteractive() bool {
	return s.interactive
}

func (s *Stdin) Text() (string, error) {
	data, err := s.Bytes()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Stdin) Bytes() ([]byte, error) {
	if s.consumed {
		return nil, &CommandError{
			Kind:     "rune/stdin-consumed",
			Message:  "stdin has already been consumed",
			ExitCode: 1,
		}
	}
	s.consumed = true
	data, err := io.ReadAll(s.reader)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func RunCommand(ctx context.Context, def CommandDefinition, opts ExecuteOptions) RunResult {
	def = DefineCommand(def)
	config := DefineConfig(opts.Config)
	jsonMode, parseArgv, jsonFlagErr := prepareOutputMode(def, config, opts)
	if jsonFlagErr != nil {
		return failureResult(jsonMode, false, nil, nil, jsonFlagErr)
	}
	parse := ParseCommand(def, parseArgv, opts.Env, config.Options)
	if !parse.OK {
		return failureResult(jsonMode, def.JSONL, nil, nil, parse.Error)
	}
	output := newOutput(jsonMode || def.JSONL)
	invocation := Invocation{Argv: append([]string(nil), opts.Argv...), CWD: cwdOrDefault(opts.CWD), Env: cloneMap(opts.Env)}
	commandCtx := &Context{
		Args:    parse.Args,
		Options: parse.Options,
		Locals:  map[string]any{},
		CWD:     cwdOrDefault(opts.CWD),
		RawArgs: parse.RawArgs,
		Output:  output,
		Stdin:   NewStdin(opts.Stdin, opts.Piped),
	}
	if def.JSON {
		commandCtx.Options["json"] = jsonMode
	}
	if config.Locals != nil {
		locals, err := config.Locals(ctx, invocation)
		if err != nil {
			return runError(ctx, config, commandCtx, output, jsonMode, def.JSONL, nil, unexpectedError(err))
		}
		if locals != nil {
			commandCtx.Locals = locals
		}
	}
	if config.Hooks.BeforeRun != nil {
		if err := config.Hooks.BeforeRun(ctx, commandCtx); err != nil {
			return runError(ctx, config, commandCtx, output, jsonMode, def.JSONL, nil, unexpectedError(err))
		}
	}
	var value any
	var err error
	if def.Run != nil {
		value, err = runCommandFunc(ctx, commandCtx, def.Run)
	}
	if err != nil {
		return runError(ctx, config, commandCtx, output, jsonMode, def.JSONL, nil, unexpectedError(err))
	}
	result := OutputResult{Kind: OutputText}
	if def.JSONL {
		jsonlResult, streamErr := emitJSONL(output, value)
		result = jsonlResult
		if streamErr != nil {
			return runError(ctx, config, commandCtx, output, false, true, result.Records, unexpectedError(streamErr))
		}
	} else if jsonMode {
		result = OutputResult{Kind: OutputJSON, Document: value}
	}
	if config.Hooks.AfterRun != nil {
		if err := config.Hooks.AfterRun(ctx, commandCtx, result); err != nil {
			return runError(ctx, config, commandCtx, output, jsonMode, def.JSONL, result.Records, unexpectedError(err))
		}
	}
	return RunResult{
		ExitCode:  0,
		Stdout:    output.stdout.String(),
		Stderr:    output.stderr.String(),
		Output:    result,
		JSONMode:  jsonMode,
		JSONLMode: def.JSONL,
	}
}

func runCommandFunc(ctx context.Context, commandCtx *Context, run RunFunc) (value any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			switch typed := recovered.(type) {
			case *CommandError:
				err = typed
			case error:
				err = typed
			default:
				err = fmt.Errorf("panic: %v", recovered)
			}
		}
	}()
	return run(ctx, commandCtx)
}

func prepareOutputMode(def CommandDefinition, config Config, opts ExecuteOptions) (bool, []string, *CommandError) {
	jsonMode := false
	var parsed []string
	afterTerminator := false
	for _, arg := range opts.Argv {
		if afterTerminator {
			parsed = append(parsed, arg)
			continue
		}
		if arg == "--" {
			afterTerminator = true
			parsed = append(parsed, arg)
			continue
		}
		if arg == "--json" {
			if def.JSONL {
				return false, nil, parseError("Option --json is not available for JSON Lines commands")
			}
			if def.JSON {
				jsonMode = true
				continue
			}
		}
		parsed = append(parsed, arg)
	}
	if def.JSON && !jsonMode && !config.DisableAutoJSON && config.AutoJSON != nil {
		jsonMode = config.AutoJSON(Invocation{Argv: append([]string(nil), opts.Argv...), CWD: opts.CWD, Env: cloneMap(opts.Env)})
	}
	return jsonMode, parsed, nil
}

func emitJSONL(output *Output, value any) (OutputResult, error) {
	result := OutputResult{Kind: OutputJSONL}
	var records []any
	var streamErr error
	switch typed := value.(type) {
	case JSONLStream:
		records = typed.Records
		streamErr = typed.Err
	case *JSONLStream:
		if typed != nil {
			records = typed.Records
			streamErr = typed.Err
		}
	case []any:
		records = typed
	default:
		return result, &CommandError{Kind: "rune/invalid-command-result", Message: "JSON Lines command returned an invalid result", ExitCode: 1}
	}
	for _, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			return result, &CommandError{Kind: "rune/serialization-failed", Message: err.Error(), ExitCode: 1}
		}
		if _, err := output.stdout.Write(append(data, '\n')); err != nil {
			if errors.Is(err, io.ErrClosedPipe) {
				return OutputResult{Kind: OutputJSONL, Records: result.Records}, nil
			}
			return result, err
		}
		result.Records = append(result.Records, record)
	}
	if streamErr != nil {
		return result, streamErr
	}
	return result, nil
}

func runError(ctx context.Context, config Config, commandCtx *Context, output *Output, jsonMode, jsonlMode bool, records []any, commandErr *CommandError) RunResult {
	commandErr = commandErr.normalized()
	if config.Hooks.OnRunError != nil {
		commandErr = combineHookError(commandErr, config.Hooks.OnRunError(ctx, commandCtx, commandErr))
	}
	return failureResult(jsonMode, jsonlMode, output, records, commandErr)
}

func failureResult(jsonMode, jsonlMode bool, output *Output, records []any, commandErr *CommandError) RunResult {
	commandErr = commandErr.normalized()
	if output == nil {
		output = newOutput(jsonMode || jsonlMode)
	}
	if jsonMode && commandErr != nil {
		data, err := json.Marshal(map[string]any{"error": commandErr})
		if err == nil {
			output.stderr.Write(append(data, '\n'))
		}
	}
	result := OutputResult{Kind: OutputText}
	if jsonMode {
		result.Kind = OutputJSON
	}
	if jsonlMode {
		result.Kind = OutputJSONL
		result.Records = append([]any(nil), records...)
	}
	exitCode := 1
	if commandErr != nil {
		exitCode = commandErr.ExitCode
	}
	return RunResult{
		ExitCode:  exitCode,
		Stdout:    output.stdout.String(),
		Stderr:    output.stderr.String(),
		Error:     commandErr,
		Output:    result,
		JSONMode:  jsonMode,
		JSONLMode: jsonlMode,
	}
}

func cwdOrDefault(cwd string) string {
	if cwd != "" {
		return cwd
	}
	current, err := os.Getwd()
	if err != nil {
		return ""
	}
	return current
}

func cloneMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}
