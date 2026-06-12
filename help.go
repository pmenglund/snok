package snok

import (
	"fmt"
	"strings"
)

type HelpKind string

const (
	HelpCommand HelpKind = "command"
	HelpGroup   HelpKind = "group"
	HelpUnknown HelpKind = "unknown"
)

type HelpData struct {
	Kind                HelpKind
	CLIName             string
	Path                []string
	Description         string
	Options             []HelpOption
	Args                []HelpArg
	Examples            []string
	Children            []HelpChild
	JSONL               bool
	AttemptedPath       []string
	MatchedPath         []string
	UnknownSegment      string
	AvailableChildNames []string
	Suggestions         []string
	Version             string
}

type HelpOption struct {
	Name         string
	Short        string
	TypeHint     string
	Description  string
	DefaultLabel string
}

type HelpArg struct {
	Name        string
	TypeHint    string
	Description string
	Required    bool
}

type HelpChild struct {
	Name        string
	Aliases     []string
	Description string
}

func HelpDataForRoute(tree *Tree, route RouteResult) HelpData {
	name := tree.Config.Name
	if name == "" {
		name = "cli"
	}
	if route.Kind == RouteUnknown {
		return HelpData{
			Kind:                HelpUnknown,
			CLIName:             name,
			AttemptedPath:       append([]string(nil), route.AttemptedPath...),
			MatchedPath:         append([]string(nil), route.MatchedPath...),
			UnknownSegment:      route.UnknownSegment,
			AvailableChildNames: append([]string(nil), route.AvailableChildNames...),
			Suggestions:         append([]string(nil), route.Suggestions...),
			Version:             tree.Config.Version,
		}
	}
	node := route.Node
	data := HelpData{
		CLIName:     name,
		Path:        append([]string(nil), route.MatchedPath...),
		Description: node.Description,
		Examples:    append([]string(nil), node.Examples...),
		Version:     tree.Config.Version,
	}
	for _, child := range node.Children() {
		data.Children = append(data.Children, HelpChild{
			Name:        child.Name,
			Aliases:     append([]string(nil), child.Aliases...),
			Description: child.Description,
		})
	}
	if route.Kind == RouteCommand && node.Command != nil {
		data.Kind = HelpCommand
		data.JSONL = node.Command.JSONL
		for _, field := range append(cloneFields(tree.Config.Options), node.Command.Options...) {
			data.Options = append(data.Options, helpOption(field))
		}
		if node.Command.JSON {
			data.Options = append(data.Options, HelpOption{Name: "json", TypeHint: "boolean", Description: "Emit structured JSON output"})
		}
		for _, field := range node.Command.Args {
			data.Args = append(data.Args, HelpArg{
				Name:        field.Name,
				TypeHint:    typeHint(field),
				Description: field.Description,
				Required:    field.Required,
			})
		}
		return data
	}
	data.Kind = HelpGroup
	return data
}

func RenderHelp(data HelpData) string {
	switch data.Kind {
	case HelpUnknown:
		return renderUnknownHelp(data)
	case HelpGroup:
		return renderGroupHelp(data)
	default:
		return renderCommandHelp(data)
	}
}

func RenderResolvedHelp(tree *Tree, route RouteResult) string {
	data := HelpDataForRoute(tree, route)
	if route.Kind == RouteCommand && route.Node != nil && route.Node.Command != nil && route.Node.Command.Help != nil {
		if text, err := route.Node.Command.Help(data); err == nil {
			return text
		}
	}
	if tree.Config.Help != nil {
		if text, err := tree.Config.Help(data); err == nil {
			return text
		}
	}
	return RenderHelp(data)
}

func renderCommandHelp(data HelpData) string {
	var b strings.Builder
	if data.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", data.Description)
	}
	fmt.Fprintf(&b, "Usage:\n  %s", usagePath(data.CLIName, data.Path))
	for _, arg := range data.Args {
		if arg.Required {
			fmt.Fprintf(&b, " <%s>", arg.Name)
		} else {
			fmt.Fprintf(&b, " [%s]", arg.Name)
		}
	}
	b.WriteString(" [options]\n")
	if len(data.Options) > 0 {
		b.WriteString("\nOptions:\n")
		for _, opt := range data.Options {
			renderOption(&b, opt)
		}
	}
	if len(data.Args) > 0 {
		b.WriteString("\nArguments:\n")
		for _, arg := range data.Args {
			required := "optional"
			if arg.Required {
				required = "required"
			}
			fmt.Fprintf(&b, "  %s", arg.Name)
			if arg.TypeHint != "" {
				fmt.Fprintf(&b, " <%s>", arg.TypeHint)
			}
			if arg.Description != "" {
				fmt.Fprintf(&b, "  %s", arg.Description)
			}
			fmt.Fprintf(&b, " (%s)\n", required)
		}
	}
	if len(data.Children) > 0 {
		b.WriteString("\nCommands:\n")
		for _, child := range data.Children {
			renderChild(&b, child)
		}
	}
	if data.JSONL {
		b.WriteString("\nOutput:\n  Emits one compact JSON value per line.\n")
	}
	if len(data.Examples) > 0 {
		b.WriteString("\nExamples:\n")
		for _, example := range data.Examples {
			fmt.Fprintf(&b, "  %s\n", example)
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func renderGroupHelp(data HelpData) string {
	var b strings.Builder
	if data.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", data.Description)
	}
	fmt.Fprintf(&b, "Usage:\n  %s <command> [options]\n", usagePath(data.CLIName, data.Path))
	if data.Version != "" && len(data.Path) == 0 {
		b.WriteString("\nOptions:\n  --version  Show version\n")
	}
	if len(data.Children) > 0 {
		b.WriteString("\nCommands:\n")
		for _, child := range data.Children {
			renderChild(&b, child)
		}
	}
	if len(data.Examples) > 0 {
		b.WriteString("\nExamples:\n")
		for _, example := range data.Examples {
			fmt.Fprintf(&b, "  %s\n", example)
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func renderUnknownHelp(data HelpData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Unknown command: %s\n", strings.Join(data.AttemptedPath, " "))
	if len(data.AvailableChildNames) > 0 {
		b.WriteString("\nAvailable commands:\n")
		for _, name := range data.AvailableChildNames {
			fmt.Fprintf(&b, "  %s\n", name)
		}
	}
	if len(data.Suggestions) > 0 {
		b.WriteString("\nDid you mean:\n")
		for _, suggestion := range data.Suggestions {
			fmt.Fprintf(&b, "  %s\n", suggestion)
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func renderOption(b *strings.Builder, opt HelpOption) {
	b.WriteString("  ")
	if opt.Short != "" {
		fmt.Fprintf(b, "-%s, ", opt.Short)
	}
	fmt.Fprintf(b, "--%s", opt.Name)
	if opt.TypeHint != "" && opt.TypeHint != "boolean" {
		fmt.Fprintf(b, " <%s>", opt.TypeHint)
	}
	if opt.Description != "" {
		fmt.Fprintf(b, "  %s", opt.Description)
	}
	if opt.DefaultLabel != "" {
		fmt.Fprintf(b, " (default %s)", opt.DefaultLabel)
	}
	b.WriteString("\n")
}

func renderChild(b *strings.Builder, child HelpChild) {
	fmt.Fprintf(b, "  %s", child.Name)
	if len(child.Aliases) > 0 {
		fmt.Fprintf(b, " (%s)", strings.Join(child.Aliases, ", "))
	}
	if child.Description != "" {
		fmt.Fprintf(b, "  %s", child.Description)
	}
	b.WriteString("\n")
}

func helpOption(field Field) HelpOption {
	return HelpOption{
		Name:         field.Name,
		Short:        field.Short,
		TypeHint:     typeHint(field),
		Description:  field.Description,
		DefaultLabel: defaultLabel(field.Default),
	}
}

func typeHint(field Field) string {
	if field.TypeHint != "" {
		return field.TypeHint
	}
	switch field.Kind {
	case FieldEnum:
		var parts []string
		for _, value := range field.Values {
			parts = append(parts, fmt.Sprint(value))
		}
		return strings.Join(parts, "|")
	case FieldSchema:
		if field.Flag {
			return "boolean"
		}
		return "value"
	default:
		if field.Type == "" {
			return string(TypeString)
		}
		return string(field.Type)
	}
}

func defaultLabel(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return fmt.Sprintf("%q", typed)
	default:
		return fmt.Sprint(typed)
	}
}

func usagePath(cliName string, path []string) string {
	parts := append([]string{cliName}, path...)
	return strings.Join(parts, " ")
}
