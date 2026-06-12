package snok

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// CobraCommand exposes a Snok tree through Cobra while keeping Snok's router,
// parser, help, and execution pipeline as the behavioral source of truth.
func CobraCommand(tree *Tree) *cobra.Command {
	name := tree.Config.Name
	if name == "" {
		name = "cli"
	}
	cmd := &cobra.Command{
		Use:                name,
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(c *cobra.Command, args []string) error {
			argv := append([]string(nil), args...)
			route := tree.Resolve(argv)
			if route.HelpRequested || route.Kind == RouteGroup || route.Kind == RouteUnknown {
				_, err := io.WriteString(c.OutOrStdout(), RenderResolvedHelp(tree, route))
				return err
			}
			result := RunCommand(context.Background(), *route.Node.Command, ExecuteOptions{
				Argv:   route.RemainingArgs,
				Env:    envMap(),
				CWD:    "",
				Stdin:  c.InOrStdin(),
				Piped:  false,
				Config: tree.Config,
			})
			if result.Stdout != "" {
				if _, err := io.WriteString(c.OutOrStdout(), result.Stdout); err != nil {
					return err
				}
			}
			if result.JSONMode && result.Output.Kind == OutputJSON {
				data, err := json.Marshal(result.Output.Document)
				if err != nil {
					return err
				}
				if _, err := c.OutOrStdout().Write(append(data, '\n')); err != nil {
					return err
				}
			}
			if result.Stderr != "" {
				if _, err := io.WriteString(c.ErrOrStderr(), result.Stderr); err != nil {
					return err
				}
			}
			if result.Error != nil {
				return result.Error
			}
			return nil
		},
	}
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		route := tree.Resolve([]string{"--help"})
		_, _ = io.WriteString(c.OutOrStdout(), RenderResolvedHelp(tree, route))
	})
	return cmd
}

func envMap() map[string]string {
	out := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			out[key] = value
		}
	}
	return out
}
