package snok

import (
	"context"
	"fmt"
	"math"
	"testing"
)

func FuzzCommandOptions(f *testing.F) {
	seeds := []struct {
		name          string
		short         string
		env           string
		kindChoice    int
		typeChoice    int
		defaultChoice int
		valuesChoice  int
		required      bool
		multiple      bool
		flag          bool
	}{
		{name: "name", kindChoice: 0, typeChoice: 0},
		{name: "count", kindChoice: 0, typeChoice: 1, defaultChoice: 2},
		{name: "force", short: "f", kindChoice: 0, typeChoice: 2},
		{name: "mode", kindChoice: 1, valuesChoice: 1},
		{name: "schema", kindChoice: 2, valuesChoice: 1},
		{name: "", kindChoice: 0, typeChoice: 0},
		{name: "bad", kindChoice: 99, typeChoice: 99, defaultChoice: 99, valuesChoice: 99, required: true, multiple: true, flag: true},
	}
	for _, seed := range seeds {
		f.Add(seed.name, seed.short, seed.env, seed.kindChoice, seed.typeChoice, seed.defaultChoice, seed.valuesChoice, seed.required, seed.multiple, seed.flag)
	}

	f.Fuzz(func(t *testing.T, name, short, env string, kindChoice, typeChoice, defaultChoice, valuesChoice int, required, multiple, flag bool) {
		field := fuzzOptionField(name, short, env, kindChoice, typeChoice, defaultChoice, valuesChoice, required, multiple, flag)
		def := CommandDefinition{
			Options: []Field{field},
			Run: func(_ context.Context, ctx *Context) (any, error) {
				if _, ok := ctx.Options[field.Name]; !ok {
					return nil, nil
				}
				if field.Multiple {
					return nil, nil
				}
				switch field.Kind {
				case FieldPrimitive:
					switch field.Type {
					case TypeString:
						_ = ctx.OptionString(field.Name)
					case TypeNumber:
						_ = ctx.OptionNumber(field.Name)
					case TypeBoolean:
						_ = ctx.OptionBool(field.Name)
					}
				}
				return nil, nil
			},
		}

		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("option definition panicked instead of rejecting or running cleanly: field=%#v panic=%v", field, recovered)
			}
		}()

		tree := NewTree(Config{})
		if err := tree.AddCommand([]string{"cmd"}, def); err != nil {
			return
		}

		result := RunCommand(context.Background(), def, ExecuteOptions{Argv: fuzzOptionArgv(field)})
		if result.Error != nil && result.Error.Kind == "rune/invalid-context-value" {
			t.Fatalf("accepted option caused context invariant failure: field=%#v err=%#v", field, result.Error)
		}
	})
}

func fuzzOptionField(name, short, env string, kindChoice, typeChoice, defaultChoice, valuesChoice int, required, multiple, flag bool) Field {
	field := Field{
		Name:     name,
		Short:    short,
		Env:      env,
		Required: required,
		Multiple: multiple,
		Flag:     flag,
		Default:  fuzzDefault(defaultChoice),
		Values:   fuzzValues(valuesChoice),
	}
	switch boundedChoice(kindChoice, 5) {
	case 0:
		field.Kind = FieldPrimitive
	case 1:
		field.Kind = FieldEnum
	case 2:
		field.Kind = FieldSchema
		if boundedChoice(valuesChoice, 3) != 0 {
			field.Validator = fuzzSchemaValidator
		}
	case 3:
		field.Kind = FieldKind("invalid")
	default:
		field.Kind = ""
	}
	switch boundedChoice(typeChoice, 5) {
	case 0:
		field.Type = TypeString
	case 1:
		field.Type = TypeNumber
	case 2:
		field.Type = TypeBoolean
	case 3:
		field.Type = PrimitiveType("invalid")
	default:
		field.Type = ""
	}
	return field
}

func fuzzDefault(choice int) any {
	switch boundedChoice(choice, 8) {
	case 0:
		return nil
	case 1:
		return "value"
	case 2:
		return float64(2)
	case 3:
		return true
	case 4:
		return []string{"a"}
	case 5:
		return math.NaN()
	case 6:
		return math.Inf(1)
	default:
		return map[string]any{"bad": true}
	}
}

func fuzzValues(choice int) []any {
	switch boundedChoice(choice, 8) {
	case 0:
		return nil
	case 1:
		return []any{"dev", "prod"}
	case 2:
		return []any{float64(1), float64(2)}
	case 3:
		return []any{true}
	case 4:
		return []any{"dup", "dup"}
	case 5:
		return []any{math.NaN()}
	case 6:
		return []any{math.Inf(1)}
	default:
		return []any{map[string]any{"bad": true}}
	}
}

func fuzzSchemaValidator(input ValidationInput) (any, error) {
	if input.Omitted {
		return nil, nil
	}
	if input.Flag {
		return input.Present, nil
	}
	if len(input.RawValues) > 0 {
		values := make([]any, 0, len(input.RawValues))
		for _, value := range input.RawValues {
			values = append(values, value)
		}
		return values, nil
	}
	return input.Raw, nil
}

func fuzzOptionArgv(field Field) []string {
	if !validFieldName(field.Name) || field.Name == "help" {
		return nil
	}
	switch field.Kind {
	case FieldPrimitive:
		switch field.Type {
		case TypeBoolean:
			return []string{"--" + field.Name}
		case TypeNumber:
			return repeatedOptionArgv(field, "2")
		default:
			return repeatedOptionArgv(field, "value")
		}
	case FieldEnum:
		if len(field.Values) == 0 {
			return nil
		}
		return repeatedOptionArgv(field, fmt.Sprint(field.Values[0]))
	case FieldSchema:
		if field.Flag {
			return []string{"--" + field.Name}
		}
		return repeatedOptionArgv(field, "value")
	default:
		return nil
	}
}

func repeatedOptionArgv(field Field, value string) []string {
	if field.Multiple {
		return []string{"--" + field.Name, value, "--" + field.Name, value}
	}
	return []string{"--" + field.Name, value}
}

func boundedChoice(value, size int) int {
	if size <= 0 {
		return 0
	}
	if value < 0 {
		value = -value
		if value < 0 {
			return 0
		}
	}
	return value % size
}
