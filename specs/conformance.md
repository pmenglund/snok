# Snok Conformance Cases

The following cases are normative unless marked evidence-only in the fixture itself.

## Routing

- `route_nested_commands`: derives root, command, and group nodes from a logical command tree.
- `route_ignored_units`: ignores private helpers, group metadata, and colocated tests.
- `route_alias_chain`: resolves command and group aliases while reporting canonical matched paths.
- `route_help_before_terminator`: treats `--help` and `-h` before `--` as framework help requests.
- `route_help_after_terminator`: treats help-looking tokens after `--` as command argv.
- `route_unknown_scoped_suggestion`: suggests only sibling names and aliases for unknown segments.

## Fields And Parsing

- `parse_primitives_and_defaults`: parses required positional args, scalar options, short flags, boolean flags, and defaults.
- `parse_env_precedence`: applies CLI value before env value before default.
- `parse_invalid_env`: invalid env values fail instead of falling back.
- `parse_repeatable_order`: repeatable option values preserve user-supplied order.
- `parse_enum_matching`: enum values match by strict string conversion.
- `parse_negatable_boolean`: default-true booleans support `--no-name` and reject positive/negative conflicts.
- `parse_schema_contract`: schema-backed values receive raw input and may transform or reject it.
- `parse_failures`: missing required fields, unknown options, duplicate scalar options, and extra positionals fail before command logic.

## Execution And Output

- `execute_text_context`: command context includes parsed args/options, cwd, rawArgs, output, stdin, and locals.
- `execute_json_mode`: active JSON mode suppresses human stdout and returns a structured document.
- `execute_json_flag_terminator`: `--json` after `--` is not framework-managed.
- `execute_jsonl_stream`: JSON Lines records are emitted one compact JSON value per line and retained in order.
- `execute_jsonl_midstream_failure`: emitted records before failure remain observable.
- `execute_structured_error`: structured command failures preserve kind, message, hint, details, and normalized exit code.
- `execute_hooks_lifecycle`: locals, hooks, command logic, and error hooks run in the required order.
- `execute_stdin_single_consume`: stdin text or bytes can be consumed once; second consumption fails.

## Help Rendering

- `help_command_sections`: command help includes description, usage, options, arguments, subcommands, output contract, and examples as applicable.
- `help_group_sections`: group help includes description, usage, child commands, aliases, version option for root when configured, and examples.
- `help_unknown_command`: unknown-command help reports attempted path, available sibling commands, and scoped canonical suggestions.
- `help_renderer_precedence`: command renderer wins over global renderer, global renderer wins over default, and renderer failure falls back to default.
