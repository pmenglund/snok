package snok

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

func (c *Context) ArgString(name string) string {
	return mustStringValue("argument", name, c.Args[name])
}

func (c *Context) OptionString(name string) string {
	return mustStringValue("option", name, c.Options[name])
}

func (c *Context) ArgNumber(name string) float64 {
	return mustNumberValue("argument", name, c.Args[name])
}

func (c *Context) OptionNumber(name string) float64 {
	return mustNumberValue("option", name, c.Options[name])
}

func (c *Context) ArgInt(name string) int {
	return mustIntValue("argument", name, c.Args[name])
}

func (c *Context) OptionInt(name string) int {
	return mustIntValue("option", name, c.Options[name])
}

func (c *Context) ArgBool(name string) bool {
	return mustBoolValue("argument", name, c.Args[name])
}

func (c *Context) OptionBool(name string) bool {
	return mustBoolValue("option", name, c.Options[name])
}

func mustStringValue(source, name string, value any) string {
	if value == nil {
		panic(contextValueError(source, name, value, "string"))
	}
	typed, ok := value.(string)
	if !ok {
		panic(contextValueError(source, name, value, "string"))
	}
	return typed
}

func mustNumberValue(source, name string, value any) float64 {
	switch typed := value.(type) {
	case float64:
		if math.IsInf(typed, 0) || math.IsNaN(typed) {
			break
		}
		return typed
	case float32:
		value := float64(typed)
		if math.IsInf(value, 0) || math.IsNaN(value) {
			break
		}
		return value
	case int:
		return float64(typed)
	case int8:
		return float64(typed)
	case int16:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint8:
		return float64(typed)
	case uint16:
		return float64(typed)
	case uint32:
		return float64(typed)
	case uint64:
		return float64(typed)
	case json.Number:
		parsed, err := typed.Float64()
		if err == nil && !math.IsInf(parsed, 0) && !math.IsNaN(parsed) {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err == nil && !math.IsInf(parsed, 0) && !math.IsNaN(parsed) {
			return parsed
		}
	}
	panic(contextValueError(source, name, value, "number"))
}

func mustIntValue(source, name string, value any) int {
	number := mustNumberValue(source, name, value)
	if math.Trunc(number) != number {
		panic(contextValueError(source, name, value, "integer"))
	}
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	if number < float64(minInt) || number > float64(maxInt) {
		panic(contextValueError(source, name, value, "integer"))
	}
	return int(number)
}

func mustBoolValue(source, name string, value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(typed)
		if err == nil {
			return parsed
		}
	}
	panic(contextValueError(source, name, value, "boolean"))
}

func contextValueError(source, name string, value any, expected string) *CommandError {
	return &CommandError{
		Kind:     "rune/invalid-context-value",
		Message:  fmt.Sprintf("Invalid %s %q: expected %s", source, name, expected),
		ExitCode: 1,
		Details: map[string]any{
			"source":   source,
			"name":     name,
			"expected": expected,
			"value":    value,
		},
	}
}
