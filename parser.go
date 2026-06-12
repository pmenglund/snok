package snok

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type ParseResult struct {
	OK      bool
	Args    map[string]any
	Options map[string]any
	RawArgs []string
	Error   *CommandError
}

var (
	optionNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*$`)
	segmentPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

func ParseCommand(def CommandDefinition, argv []string, env map[string]string, globalOptions []Field) ParseResult {
	def = DefineCommand(def)
	options := append(cloneFields(globalOptions), def.Options...)
	if err := validateFields(def.Args, options); err != nil {
		return ParseResult{OK: false, Error: parseError("%s", err.Error()), RawArgs: append([]string(nil), argv...)}
	}
	if env == nil {
		env = map[string]string{}
	}
	state := parserState{
		optionDefs:   map[string]Field{},
		shortToLong:  map[string]string{},
		optionValues: map[string][]string{},
		optionSource: map[string]string{},
		positiveBool: map[string]bool{},
		negativeBool: map[string]bool{},
	}
	for _, field := range options {
		state.optionDefs[field.Name] = field
		if field.Short != "" {
			state.shortToLong[field.Short] = field.Name
		}
	}
	positionals, err := state.scan(argv)
	if err != nil {
		return ParseResult{OK: false, Error: err, RawArgs: append([]string(nil), argv...)}
	}
	parsedArgs, err := parsePositionals(def.Args, positionals)
	if err != nil {
		return ParseResult{OK: false, Error: err, RawArgs: append([]string(nil), argv...)}
	}
	parsedOptions, err := state.finalizeOptions(options, env)
	if err != nil {
		return ParseResult{OK: false, Error: err, RawArgs: append([]string(nil), argv...)}
	}
	return ParseResult{
		OK:      true,
		Args:    parsedArgs,
		Options: parsedOptions,
		RawArgs: append([]string(nil), argv...),
	}
}

type parserState struct {
	optionDefs   map[string]Field
	shortToLong  map[string]string
	optionValues map[string][]string
	optionSource map[string]string
	positiveBool map[string]bool
	negativeBool map[string]bool
}

func (p *parserState) scan(argv []string) ([]string, *CommandError) {
	var positionals []string
	afterTerminator := false
	for i := 0; i < len(argv); i++ {
		token := argv[i]
		if afterTerminator {
			positionals = append(positionals, token)
			continue
		}
		if token == "--" {
			afterTerminator = true
			continue
		}
		if strings.HasPrefix(token, "--") && token != "--" {
			nameValue := strings.TrimPrefix(token, "--")
			value := ""
			hasInlineValue := false
			if before, after, ok := strings.Cut(nameValue, "="); ok {
				nameValue = before
				value = after
				hasInlineValue = true
			}
			negated := false
			if strings.HasPrefix(nameValue, "no-") {
				candidate := strings.TrimPrefix(nameValue, "no-")
				field, ok := p.optionDefs[candidate]
				if ok && field.Kind == FieldPrimitive && field.Type == TypeBoolean && boolDefault(field.Default) {
					nameValue = candidate
					negated = true
				}
			}
			field, ok := p.optionDefs[nameValue]
			if !ok {
				return nil, parseError("Unknown option --%s", nameValue)
			}
			if negated {
				if hasInlineValue {
					return nil, parseError("Option --no-%s does not accept a value", nameValue)
				}
				if p.positiveBool[nameValue] {
					return nil, parseError("Conflicting options --%s and --no-%s", nameValue, nameValue)
				}
				p.negativeBool[nameValue] = true
				p.optionValues[nameValue] = []string{"false"}
				p.optionSource[nameValue] = "cli"
				continue
			}
			if isValueLessFlag(field) {
				if hasInlineValue {
					return nil, parseError("Option --%s does not accept a value", nameValue)
				}
				if p.negativeBool[nameValue] {
					return nil, parseError("Conflicting options --%s and --no-%s", nameValue, nameValue)
				}
				p.positiveBool[nameValue] = true
				if field.Multiple {
					p.optionValues[nameValue] = append(p.optionValues[nameValue], "true")
				} else if _, seen := p.optionValues[nameValue]; seen {
					return nil, parseError("Duplicate option --%s", nameValue)
				} else {
					p.optionValues[nameValue] = []string{"true"}
				}
				p.optionSource[nameValue] = "cli"
				continue
			}
			if !hasInlineValue {
				i++
				if i >= len(argv) {
					return nil, parseError("Missing value for option --%s", nameValue)
				}
				value = argv[i]
			}
			if field.Multiple {
				p.optionValues[nameValue] = append(p.optionValues[nameValue], value)
			} else if _, seen := p.optionValues[nameValue]; seen {
				return nil, parseError("Duplicate option --%s", nameValue)
			} else {
				p.optionValues[nameValue] = []string{value}
			}
			p.optionSource[nameValue] = "cli"
			continue
		}
		if strings.HasPrefix(token, "-") && token != "-" {
			short := strings.TrimPrefix(token, "-")
			if len(short) != 1 {
				return nil, parseError("Unknown option -%s", short)
			}
			name, ok := p.shortToLong[short]
			if !ok {
				return nil, parseError("Unknown option -%s", short)
			}
			field := p.optionDefs[name]
			if isValueLessFlag(field) {
				if _, seen := p.optionValues[name]; seen && !field.Multiple {
					return nil, parseError("Duplicate option --%s", name)
				}
				p.positiveBool[name] = true
				p.optionValues[name] = append(p.optionValues[name], "true")
				p.optionSource[name] = "cli"
				continue
			}
			i++
			if i >= len(argv) {
				return nil, parseError("Missing value for option -%s", short)
			}
			if field.Multiple {
				p.optionValues[name] = append(p.optionValues[name], argv[i])
			} else if _, seen := p.optionValues[name]; seen {
				return nil, parseError("Duplicate option --%s", name)
			} else {
				p.optionValues[name] = []string{argv[i]}
			}
			p.optionSource[name] = "cli"
			continue
		}
		positionals = append(positionals, token)
	}
	return positionals, nil
}

func parsePositionals(fields []Field, tokens []string) (map[string]any, *CommandError) {
	out := map[string]any{}
	for i, field := range fields {
		if i >= len(tokens) {
			if field.Default != nil {
				out[field.Name] = field.Default
				continue
			}
			if field.Required {
				return nil, parseError("Missing required argument %s", field.Name)
			}
			if field.Kind == FieldSchema && field.Validator != nil {
				value, err := field.Validator(ValidationInput{Omitted: true})
				if err != nil {
					return nil, parseError("Invalid value for argument %s: %s", field.Name, err)
				}
				if value != nil {
					out[field.Name] = value
				}
			}
			continue
		}
		value, err := parseFieldValue(field, []string{tokens[i]}, true)
		if err != nil {
			return nil, parseError("Invalid value for argument %s: %s", field.Name, err)
		}
		out[field.Name] = value
	}
	if len(tokens) > len(fields) {
		return nil, parseError("Unexpected argument %s", tokens[len(fields)])
	}
	return out, nil
}

func (p *parserState) finalizeOptions(fields []Field, env map[string]string) (map[string]any, *CommandError) {
	out := map[string]any{}
	for _, field := range fields {
		values, hasCLI := p.optionValues[field.Name]
		source := "cli"
		if !hasCLI && field.Env != "" {
			if value, ok := env[field.Env]; ok {
				values = []string{value}
				source = "environment variable " + field.Env
			}
		}
		if len(values) > 0 {
			value, err := parseFieldValue(field, values, false)
			if err != nil {
				return nil, parseError("Invalid value for option --%s from %s: %s", field.Name, source, err)
			}
			out[field.Name] = value
			continue
		}
		if field.Default != nil {
			out[field.Name] = field.Default
			continue
		}
		if field.Kind == FieldPrimitive && field.Type == TypeBoolean {
			out[field.Name] = false
			continue
		}
		if field.Kind == FieldSchema && field.Flag {
			if field.Validator != nil {
				value, err := field.Validator(ValidationInput{Omitted: true, Flag: true})
				if err != nil {
					return nil, parseError("Invalid value for option --%s: %s", field.Name, err)
				}
				if value != nil {
					out[field.Name] = value
				}
			}
			continue
		}
		if field.Required {
			return nil, parseError("Missing required option --%s", field.Name)
		}
		if field.Kind == FieldSchema && field.Validator != nil {
			value, err := field.Validator(ValidationInput{Omitted: true})
			if err != nil {
				return nil, parseError("Invalid value for option --%s: %s", field.Name, err)
			}
			if value != nil {
				out[field.Name] = value
			}
		}
	}
	return out, nil
}

func parseFieldValue(field Field, raw []string, positional bool) (any, error) {
	if field.Multiple {
		if field.Kind == FieldSchema {
			if field.Validator == nil {
				return append([]string(nil), raw...), nil
			}
			return field.Validator(ValidationInput{RawValues: append([]string(nil), raw...), Present: true})
		}
		out := make([]any, 0, len(raw))
		single := field
		single.Multiple = false
		for _, token := range raw {
			value, err := parseFieldValue(single, []string{token}, positional)
			if err != nil {
				return nil, err
			}
			out = append(out, value)
		}
		return out, nil
	}
	token := ""
	if len(raw) > 0 {
		token = raw[0]
	}
	switch field.Kind {
	case "", FieldPrimitive:
		return parsePrimitive(field.Type, token)
	case FieldEnum:
		for _, value := range field.Values {
			if fmt.Sprint(value) == token {
				return value, nil
			}
		}
		var allowed []string
		for _, value := range field.Values {
			allowed = append(allowed, fmt.Sprint(value))
		}
		return nil, fmt.Errorf("Expected one of %s", strings.Join(allowed, ", "))
	case FieldSchema:
		if field.Validator == nil {
			if field.Flag {
				return true, nil
			}
			return token, nil
		}
		return field.Validator(ValidationInput{Raw: token, Present: true, Flag: field.Flag})
	default:
		return nil, fmt.Errorf("unknown field kind %q", field.Kind)
	}
}

func parsePrimitive(kind PrimitiveType, token string) (any, error) {
	switch kind {
	case "", TypeString:
		return token, nil
	case TypeNumber:
		value, err := strconv.ParseFloat(token, 64)
		if err != nil || math.IsInf(value, 0) || math.IsNaN(value) {
			return nil, fmt.Errorf("Expected number")
		}
		return value, nil
	case TypeBoolean:
		if token == "true" {
			return true, nil
		}
		if token == "false" {
			return false, nil
		}
		return nil, fmt.Errorf("Expected boolean")
	default:
		return nil, fmt.Errorf("unknown primitive type %q", kind)
	}
}

func isValueLessFlag(field Field) bool {
	return (field.Kind == FieldPrimitive && field.Type == TypeBoolean) || (field.Kind == FieldSchema && field.Flag)
}

func boolDefault(value any) bool {
	boolValue, ok := value.(bool)
	return ok && boolValue
}

func validateCommandDefinition(path []string, def CommandDefinition, globals []Field) error {
	if len(path) == 0 && len(def.Aliases) > 0 {
		return fmt.Errorf("root command must not have aliases")
	}
	if def.JSON && def.JSONL {
		return fmt.Errorf("command %q cannot enable both json and jsonl", strings.Join(path, " "))
	}
	for _, field := range globals {
		if field.Name == "json" {
			return fmt.Errorf("global option --json is reserved")
		}
	}
	if def.JSON || def.JSONL {
		for _, field := range def.Options {
			if field.Name == "json" {
				return fmt.Errorf("command option --json is reserved when structured output is enabled")
			}
		}
	}
	if err := validateAliases(def.Aliases); err != nil {
		return err
	}
	return validateFields(def.Args, append(cloneFields(globals), def.Options...))
}

func validateAliases(aliases []string) error {
	seen := map[string]bool{}
	for _, alias := range aliases {
		if !segmentPattern.MatchString(alias) {
			return fmt.Errorf("invalid alias %q", alias)
		}
		if seen[alias] {
			return fmt.Errorf("duplicate alias %q", alias)
		}
		seen[alias] = true
	}
	return nil
}

func validateSegment(segment string) error {
	if !segmentPattern.MatchString(segment) || strings.Contains(segment, "--") || strings.HasSuffix(segment, "-") {
		return fmt.Errorf("invalid route segment %q", segment)
	}
	return nil
}

func validateFields(args, options []Field) error {
	seenOptionalArg := false
	argNames := map[string]bool{}
	for _, arg := range args {
		if err := validateField(arg, "argument", false); err != nil {
			return err
		}
		if arg.Name == "" || !validFieldName(arg.Name) {
			return fmt.Errorf("invalid argument name %q", arg.Name)
		}
		if argNames[arg.Name] {
			return fmt.Errorf("duplicate argument %q", arg.Name)
		}
		argNames[arg.Name] = true
		if seenOptionalArg && arg.Required {
			return fmt.Errorf("required argument %q follows optional argument", arg.Name)
		}
		if !arg.Required {
			seenOptionalArg = true
		}
	}
	optionNames := map[string]bool{}
	shortNames := map[string]bool{}
	for _, opt := range options {
		if err := validateField(opt, "option", true); err != nil {
			return err
		}
		if !validFieldName(opt.Name) {
			return fmt.Errorf("invalid option name %q", opt.Name)
		}
		if opt.Name == "help" {
			return fmt.Errorf("option --help is reserved")
		}
		if optionNames[opt.Name] {
			return fmt.Errorf("duplicate option --%s", opt.Name)
		}
		optionNames[opt.Name] = true
		if opt.Short != "" {
			if len(opt.Short) != 1 || !regexp.MustCompile(`^[A-Za-z]$`).MatchString(opt.Short) {
				return fmt.Errorf("invalid short option -%s", opt.Short)
			}
			if opt.Short == "h" {
				return fmt.Errorf("short option -h is reserved")
			}
			if shortNames[opt.Short] {
				return fmt.Errorf("duplicate short option -%s", opt.Short)
			}
			shortNames[opt.Short] = true
		}
		if opt.Multiple && opt.Env != "" {
			return fmt.Errorf("repeatable option --%s must not use environment fallback", opt.Name)
		}
		if opt.Required && opt.Default != nil {
			return fmt.Errorf("required option --%s must not have a default", opt.Name)
		}
	}
	return nil
}

func validateField(field Field, role string, option bool) error {
	if field.Kind != FieldPrimitive && field.Kind != FieldEnum && field.Kind != FieldSchema {
		return fmt.Errorf("invalid %s %q kind %q", role, field.Name, field.Kind)
	}
	if !option {
		if field.Short != "" {
			return fmt.Errorf("argument %q must not define a short option", field.Name)
		}
		if field.Env != "" {
			return fmt.Errorf("argument %q must not define an environment fallback", field.Name)
		}
		if field.Multiple {
			return fmt.Errorf("argument %q must not be repeatable", field.Name)
		}
		if field.Flag {
			return fmt.Errorf("argument %q must not be a flag", field.Name)
		}
	}
	if option && field.Multiple && field.Default != nil {
		return fmt.Errorf("repeatable option --%s must not have a default", field.Name)
	}
	switch field.Kind {
	case FieldPrimitive:
		if field.Type != TypeString && field.Type != TypeNumber && field.Type != TypeBoolean {
			return fmt.Errorf("invalid %s %q primitive type %q", role, field.Name, field.Type)
		}
		if len(field.Values) > 0 {
			return fmt.Errorf("primitive %s %q must not define enum values", role, field.Name)
		}
		if field.Validator != nil {
			return fmt.Errorf("primitive %s %q must not define a validator", role, field.Name)
		}
		if field.Flag {
			return fmt.Errorf("primitive %s %q must not define schema flag behavior", role, field.Name)
		}
		if option && field.Type == TypeBoolean && field.Required {
			return fmt.Errorf("boolean option --%s must not be required", field.Name)
		}
		if field.Default != nil {
			if err := validatePrimitiveDefault(field); err != nil {
				return fmt.Errorf("invalid default for %s %q: %w", role, field.Name, err)
			}
		}
	case FieldEnum:
		if field.Type != "" {
			return fmt.Errorf("enum %s %q must not define a primitive type", role, field.Name)
		}
		if field.Validator != nil {
			return fmt.Errorf("enum %s %q must not define a validator", role, field.Name)
		}
		if field.Flag {
			return fmt.Errorf("enum %s %q must not define schema flag behavior", role, field.Name)
		}
		if len(field.Values) == 0 {
			return fmt.Errorf("enum %s %q must define at least one value", role, field.Name)
		}
		seen := map[string]bool{}
		for _, value := range field.Values {
			key, ok := enumValueKey(value)
			if !ok {
				return fmt.Errorf("enum %s %q contains non-string/non-number value %T", role, field.Name, value)
			}
			if seen[key] {
				return fmt.Errorf("enum %s %q contains duplicate value %s", role, field.Name, key)
			}
			seen[key] = true
		}
		if field.Default != nil {
			key, ok := enumValueKey(field.Default)
			if !ok || !seen[key] {
				return fmt.Errorf("invalid default for enum %s %q", role, field.Name)
			}
		}
	case FieldSchema:
		if field.Type != "" {
			return fmt.Errorf("schema %s %q must not define a primitive type", role, field.Name)
		}
		if len(field.Values) > 0 {
			return fmt.Errorf("schema %s %q must not define enum values", role, field.Name)
		}
		if field.Validator == nil {
			return fmt.Errorf("schema %s %q must define a validator", role, field.Name)
		}
		if field.Default != nil {
			return fmt.Errorf("schema %s %q must not define a default; use validator omitted behavior", role, field.Name)
		}
		if field.Flag && !option {
			return fmt.Errorf("schema argument %q must not be a flag", field.Name)
		}
		if field.Flag && field.Required {
			return fmt.Errorf("schema flag option --%s must not be required", field.Name)
		}
	}
	return nil
}

func validatePrimitiveDefault(field Field) error {
	switch field.Type {
	case TypeString:
		if _, ok := field.Default.(string); !ok {
			return fmt.Errorf("expected string")
		}
	case TypeNumber:
		if _, ok := numericValueKey(field.Default); !ok {
			return fmt.Errorf("expected finite number")
		}
	case TypeBoolean:
		if _, ok := field.Default.(bool); !ok {
			return fmt.Errorf("expected boolean")
		}
	}
	return nil
}

func enumValueKey(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return "s:" + typed, true
	default:
		key, ok := numericValueKey(value)
		if !ok {
			return "", false
		}
		return "n:" + key, true
	}
}

func numericValueKey(value any) (string, bool) {
	switch typed := value.(type) {
	case float64:
		if math.IsInf(typed, 0) || math.IsNaN(typed) {
			return "", false
		}
		return strconv.FormatFloat(typed, 'g', -1, 64), true
	case float32:
		value := float64(typed)
		if math.IsInf(value, 0) || math.IsNaN(value) {
			return "", false
		}
		return strconv.FormatFloat(value, 'g', -1, 32), true
	case int:
		return strconv.FormatInt(int64(typed), 10), true
	case int8:
		return strconv.FormatInt(int64(typed), 10), true
	case int16:
		return strconv.FormatInt(int64(typed), 10), true
	case int32:
		return strconv.FormatInt(int64(typed), 10), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case uint:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint8:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint16:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint32:
		return strconv.FormatUint(uint64(typed), 10), true
	case uint64:
		return strconv.FormatUint(typed, 10), true
	default:
		return "", false
	}
}

func validFieldName(name string) bool {
	return optionNamePattern.MatchString(name) && !strings.Contains(name, "--") && !strings.HasSuffix(name, "-")
}
