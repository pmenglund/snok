package main

import (
	"context"
	"strings"

	"github.com/pmenglund/snok"
)

func init() {
	registerCommand(registerGreetCommand)
}

func registerGreetCommand(tree *snok.Tree) error {
	return tree.AddCommand([]string{"greet"}, snok.CommandDefinition{
		Description: "Print a greeting.",
		Examples: []string{
			"snok-modular greet Ada --title Dr --shout",
		},
		Args: []snok.Field{
			{Name: "name", Kind: snok.FieldPrimitive, Type: snok.TypeString, Required: true, Description: "Person to greet"},
		},
		Options: []snok.Field{
			{Name: "title", Kind: snok.FieldPrimitive, Type: snok.TypeString, Default: "", Description: "Optional title"},
			{Name: "shout", Kind: snok.FieldPrimitive, Type: snok.TypeBoolean, Short: "s", Description: "Uppercase the greeting"},
		},
		Run: func(_ context.Context, ctx *snok.Context) (any, error) {
			name := ctx.ArgString("name")
			if title := ctx.OptionString("title"); title != "" {
				name = title + " " + name
			}
			message := "Hello, " + name + "!"
			if ctx.OptionBool("shout") {
				message = strings.ToUpper(message)
			}
			ctx.Output.Log("%s", message)
			return nil, nil
		},
	})
}
