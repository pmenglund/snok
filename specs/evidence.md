# Evidence Ledger

Source baseline: Rune framework core at revision `839ede33ce3644d3b60d5d5bd161a3217a425fda`.

## Public Documentation

- `https://rune-cli.org/`: observed and required continuity for the agent-native principles, file-based routing, type-safe command definitions, Standard Schema support, test utilities, JSON output, help generation, structured errors, and agent skills.
- `https://rune-cli.org/guides/routing/`: observed and required continuity for command tree derivation, ignored private/test units, group metadata, root commands, and matched-command-only loading.
- `https://rune-cli.org/guides/commands/`: observed and required continuity for command definitions, groups, args, options, enum fields, repeatable options, env fallback, kebab-case names, negatable booleans, and aliases.
- `https://rune-cli.org/guides/json/`: observed and required continuity for JSON mode, JSON Lines mode, stdout suppression, stderr preservation, and structured error output.
- `https://rune-cli.org/guides/testing/`: observed and required continuity for in-process command testing, captured outputs, structured errors, JSON outputs, injected cwd/env/stdin, and config-backed helpers.
- `https://rune-cli.org/guides/standard-schema/`: observed and required continuity for schema-backed fields, omitted/default behavior, schema flags, display labels, and error behavior.
- `https://rune-cli.org/guides/help-customization/`: observed and required continuity for default, command-level, and global help rendering.

## Exported Interfaces

- `packages/rune/src/index.ts`: required continuity for public landmarks: `defineCommand`, `defineConfig`, `defineGroup`, `CommandError`, `renderDefaultHelp`, command context types, hooks, and help data.
- `packages/rune/src/test.ts`: required continuity for `runCommand` and `createRunCommand`.
- `packages/rune/src/core/command-types.ts`: observed behavior for command input fields, normalized command definitions, context values, JSON/JSONL modes, and inferred output categories.
- `packages/rune/src/core/field-types.ts`: observed behavior for primitive, enum, schema, flag, repeatable, env-backed, defaulted, required, and short option fields.

## Implementation Evidence

- `packages/rune/src/core/define-command.ts`: observed behavior for definition validation, copied arrays, reserved names, aliases, required-arg ordering, JSON/JSONL exclusivity, and branding.
- `packages/rune/src/core/define-config.ts`: observed behavior for config metadata, global options, hooks, locals, config option validation, and reserved global option names.
- `packages/rune/src/core/define-group.ts`: observed behavior for group metadata and alias validation.
- `packages/rune/src/core/parse-command-args.ts`: observed behavior for argv parsing, env precedence, defaults, schema validation contract, enum matching, repeatable options, negation, duplicate detection, parse failures, and camel-case aliases.
- `packages/rune/src/core/run-command-pipeline.ts`: observed behavior for execution context, JSON flag extraction, agent JSON activation, output suppression, JSONL streaming, broken pipes, structured failure normalization, hooks, locals, and lifecycle stages.
- `packages/rune/src/routing/resolve-command-route.ts`: observed behavior for route matching, help detection, aliases, unknown commands, and suggestions.
- `packages/rune/src/help/*`: observed behavior for structured help data, default rendering, custom renderer precedence, and safe fallback.
- `packages/rune/src/test-utils/run-command.ts`: observed behavior for in-process command tests, output capture, injected context, JSON documents, JSON Lines records, and error envelopes.

## Test Evidence

- `packages/rune/tests/core/define-command.test.ts`: required continuity for command normalization and definition-time validation categories.
- `packages/rune/tests/core/define-config.test.ts`: required continuity for global options, hooks, locals, and config validation.
- `packages/rune/tests/core/parse-command-args.test.ts`: required continuity for parsing, defaults, env, schemas, repeats, negation, failures, and aliases.
- `packages/rune/tests/core/run-command-pipeline.test.ts`: required continuity for execution context, output modes, structured errors, hooks, locals, stdin, JSONL, and broken pipes.
- `packages/rune/tests/routing/resolve-command-route.test.ts`: required continuity for route results, aliases, help flags, unknown-command suggestions, and argument passthrough.
- `packages/rune/tests/help/*.test.ts`: required continuity for help data, default help sections, group help, command help, unknown-command help, custom renderer precedence, and safe fallback.
- `packages/rune/tests/test-utils/*.test.ts`: required continuity for in-process test runner behavior.
- `packages/rune/tests/manifest/generate/*.test.ts`: evidence for source tree discovery. Snok keeps the logical behavior and excludes source-language file extension mechanics.

## Excluded Or Rewritten Assets

- TypeScript syntax, type inference machinery, symbol branding, package exports, runtime version constraints, test runner APIs, and build commands are incidental implementation and excluded from the portable core.
- Exact Rune help byte strings are rewritten into semantic section and substring checks because Snok will render through Cobra-compatible conventions.
- Rune source file extensions and package-specific directories are rewritten into logical command tree units.
- `create-rune-app`, Rune build/sync commands, generated type files, and package manager workflows are out of scope for framework-core parity.
- Internal prompts are not present and generative-output parity is not applicable.
