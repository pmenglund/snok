---
title: "Snok Framework Specification"
status: Draft
owners:
  - "Snok maintainers"
last_updated: 2026-06-11
compatibility_target: deterministic-behavioral
source_baseline:
  project: "Rune framework core"
  revision: "839ede33ce3644d3b60d5d5bd161a3217a425fda"
---

# Snok Framework Specification

Status: Draft

Purpose: Snok is a Cobra-based CLI framework layer that makes command-line tools easy for both humans and AI agents to understand, operate, and test. It preserves Rune's framework-core behavior at the observable level while using idiomatic Go and Cobra implementation mechanics.

## 1. Goals

- Make a CLI's command tree visible from project structure and metadata without requiring users or agents to reverse-engineer custom wiring.
- Provide a single command definition model that covers command metadata, positional arguments, options, output modes, structured errors, hooks, local runtime values, stdin, and tests.
- Ensure human-readable and machine-readable execution paths share the same parsing, validation, defaults, hooks, and command logic.
- Support deterministic conformance scenarios so independent implementations can agree on routing, parsing, help, output, and failure behavior.

## 2. Non-Goals

- Snok does not need to reproduce Rune's source language, package layout, build pipeline, scaffolding tool, or internal implementation.
- Snok does not define Cobra internals or replace Cobra's command execution engine; it constrains the wrapper behavior exposed to Snok users.
- Snok does not require byte-for-byte help text parity with Rune, except where a conformance fixture explicitly marks text as exact.
- Snok does not require a specific schema library. Schema-backed validation is expressed as a portable validator contract.

## 3. Context

Rune is an agent-native CLI framework built around file-based command routing, declarative command definitions, generated help, structured JSON output, structured errors, lifecycle hooks, locals, stdin abstraction, and in-process command tests. Snok adopts those framework-core principles for Go applications built on Cobra.

### Compatibility Target

- Target: deterministic behavioral parity.
- Source baseline: Rune framework core at revision `839ede33ce3644d3b60d5d5bd161a3217a425fda`.
- Observable parity boundary: route resolution, command metadata, field validation, argument and option parsing, defaults, help data, output modes, structured failures, hook lifecycle, stdin consumption, and in-process test results.
- Versioned parity profile: none. There are no model, prompt, or provider choices in scope.

## 4. Users and Use Cases

### Primary Users

- Go developers building CLIs on Cobra who want a higher-level, agent-friendly command contract.
- AI coding agents that need to inspect, modify, and test CLIs safely from visible structure and portable metadata.
- Test suites that need to execute command logic in-process with controlled argv, cwd, env, stdin, stdout, and stderr.

### Hero Use Case

A developer adds a new command by creating a visibly named command unit, declaring metadata, args, options, and a run function, then tests it in process. Snok automatically exposes the command in routing and help, parses user input into typed context values, supports structured output for agents, and reports structured failures consistently.

### Additional Use Cases

- Define global options, lifecycle hooks, and local runtime values once and apply them to every executable command.
- Emit normal text for humans and structured JSON or JSON Lines for automation from the same command logic.
- Generate help for root commands, groups, executable commands, and unknown commands without loading unrelated leaf command implementation.
- Validate CLI behavior with portable fixtures before or during a Go implementation.

## 5. Product and Behavior Contract

### Agent-Friendly CLI Shape

- The public command tree must be discoverable from explicit command metadata and a visible project structure.
- A command group may contain subcommands without being executable itself.
- An executable command may also have child commands; when invoked with no child segment, its own command logic runs.
- Private helpers and colocated tests must not become public commands.
- At runtime, command dispatch must load or initialize only the matched executable command and the metadata required to route and render help.

### Command Definitions

Snok must expose a command definition concept equivalent to Rune's `defineCommand`.

| Canonical field | Required behavior |
|---|---|
| `description` | Optional one-line summary shown in help. |
| `aliases` | Optional alternative path segments. Aliases must be lowercase kebab-case names made from letters, digits, and single internal hyphens. The root command must not have aliases. |
| `examples` | Optional ordered list of full command invocation examples shown in help. |
| `args` | Ordered positional fields. Required positional fields must not follow optional positional fields. |
| `options` | Ordered named fields exposed as long flags, with optional single-letter short flags. |
| `json` | When enabled, the framework owns a `--json` flag and can emit structured success and error documents. |
| `jsonl` | When enabled, the command emits one compact JSON record per line and must not also enable `json`. |
| `help` | Optional command-level help renderer. It takes precedence over global and default help rendering. |
| `run` | Executable command function called after routing, parsing, locals creation, and before-run hooks succeed. |

Command definitions must copy user-supplied `args`, `options`, `aliases`, and `examples` so later caller mutation does not change command behavior.

### Group Definitions

Snok must expose a group definition concept equivalent to Rune's `defineGroup`.

| Canonical field | Required behavior |
|---|---|
| `description` | Required one-line summary shown in group help. |
| `aliases` | Optional alternative path segments using the same alias rules as commands. |
| `examples` | Optional ordered examples shown in group help. |

Group definitions are metadata only and do not run command logic.

### Configuration

Snok must expose a project configuration concept equivalent to Rune's `defineConfig`.

| Canonical field | Required behavior |
|---|---|
| `name` | Optional display name used in help and version output. If omitted, implementations may derive a project name from host conventions. |
| `version` | Optional display version used in help and version output. |
| `help` | Optional global help renderer used for groups, unknown commands, and commands without command-level renderers. |
| `options` | Optional global options available to every executable command. Global options must be optional and must not use framework-reserved names. |
| `hooks` | Optional lifecycle hooks around executable command runs. |
| `locals` | Optional factory that creates invocation-local values once per successful command invocation. |

Global options are parsed with the same rules as command options and are visible to command logic and hooks. Command-level options override or conflict with globals only where an implementation explicitly rejects duplicate names; silent shadowing is not allowed.

### Field Model

Snok must support positional argument fields and option fields. A field must use exactly one of primitive, enum, or schema-backed validation.

| Field kind | Required behavior |
|---|---|
| Primitive string | Accepts any provided string token. |
| Primitive number | Converts the token to a finite number or fails validation. |
| Primitive boolean | Accepts `true` or `false` as positional/env values; boolean option flags are value-less and become true when present. |
| Enum | Accepts only listed string or number choices. Matching uses `String(value) == raw_token`; number enum value `1` accepts token `1` but not `1.0` or `007`. |
| Schema-backed value | Passes the raw token, omitted value, or repeated raw token collection into a validator contract that may validate and transform the value. |
| Schema-backed flag | Passes true when the flag is present and an omitted value when absent. |

Field names are public keys in parsed `args` and `options`. Option names must start with a letter and contain only letters, digits, and single internal hyphens. Hyphenated argument names must follow the same kebab-case rule. Empty argument names are invalid.

### Option Behavior

- Long options use `--name`; short options use `-x`.
- A short option must be exactly one ASCII letter and unique within the effective option set.
- The framework reserves `--help` and `-h` everywhere.
- Global options reserve `--json`; command options reserve `--json` only when JSON or JSON Lines behavior is enabled.
- Scalar options may declare an environment variable fallback. Resolution order is CLI value, then environment value, then default, then omitted or missing-required behavior.
- Environment values are parsed through the same validation path as CLI values. An invalid environment value fails the invocation and must not fall back to a default.
- Empty environment strings count as provided values.
- Repeatable options may be supplied multiple times; parsed primitive and enum values are arrays in user-supplied order.
- Repeatable schema-backed value options pass the collected raw string array to the schema validator.
- Repeatable options must not use environment fallback.
- Primitive boolean options omitted by the user default to false.
- Primitive boolean options with default true must expose a `--no-name` negated form. Supplying both positive and negative forms is a parse failure.
- Schema-backed flags do not receive an automatic negated form.

### Positional Arguments

- Positional arguments are matched by declaration order.
- Missing required positional arguments fail parsing before the command runs.
- Optional positional arguments without defaults are omitted from parsed `args`.
- Positional defaults are applied when the user omits the argument.
- Extra positional tokens fail parsing unless they occur after the resolved command path and belong to a command that deliberately accepts passthrough through its own field model.

### Routing

- The root command path is the empty segment list.
- A visible command unit at a path segment creates an executable command for that path.
- A visible group unit creates a non-executable group for that path.
- A visible command unit may also have children.
- Names beginning with `_` are ignored by routing, except for the reserved group metadata unit.
- Colocated test units ending in `.test` or `.spec` are ignored by routing.
- A file-like command and a directory-like command with the same public segment at the same level are invalid.
- Route resolution consumes argv segments while they match child command names or aliases.
- Once an executable command is matched, unmatched later tokens are passed to that command as remaining argv.
- When a group is matched and the next token is unknown, route resolution returns an unknown-command result scoped to that group.
- `--help` and `-h` before the argument terminator request help for the resolved command or group. Help flags after `--` are normal remaining argv.
- Unknown-command suggestions are scoped to sibling command and alias names, and should include close adjacent transposition matches. Suggestions must report canonical command names, not aliases.

### Execution Context

When command logic runs, the context must expose:

| Context field | Required behavior |
|---|---|
| `args` | Parsed, validated positional values keyed by canonical field name. |
| `options` | Parsed, validated option values keyed by canonical field name, including effective global options. |
| `locals` | Values returned by the project locals factory, or an empty record when no factory exists. |
| `cwd` | Caller working directory for the invocation. |
| `rawArgs` | Original argv tokens passed to the resolved command before framework-managed token removal. |
| `output` | Framework-owned stdout/stderr API. |
| `stdin` | Framework-owned stdin API. |

Implementations may expose idiomatic Go field accessors instead of map-like access, but the observable keys and values must match the contract.

### Lifecycle

- Routing and argument parsing happen before locals and hooks.
- The locals factory runs once after routing and parsing succeed and before `beforeRun`.
- `beforeRun` runs after locals creation and before command logic.
- `afterRun` runs after successful command logic and receives the result kind: text, JSON document, or JSON Lines records.
- `onRunError` runs when locals, `beforeRun`, command logic, JSONL record validation/serialization, or `afterRun` fails.
- Hooks do not run for help, version output, unknown commands, group help, or parse failures.
- If `beforeRun` fails, command logic must not run.
- If `afterRun` fails, the invocation fails at the after-run stage.
- If `onRunError` itself fails, the reported failure must preserve both the original failure and the hook failure.

### Output Modes

#### Text Mode

- `output.log` writes human stdout with a trailing newline.
- `output.error` writes human stderr with a trailing newline.
- A command return value is ignored unless JSON or JSON Lines mode is enabled.

#### JSON Mode

- JSON mode is available only for commands that enable `json`.
- The framework-managed `--json` flag before `--` activates JSON mode and is removed before user option parsing. `--json` after `--` is not framework-managed.
- When JSON mode is active, human stdout from `output.log` is suppressed.
- `output.error` still writes stderr.
- Successful command return data is the JSON document result.
- JSON-enabled command context includes an option value indicating whether JSON mode is active.
- Agent-environment auto-activation may enable JSON mode without `--json`; it must be suppressible by configuration or environment so tests can be deterministic.
- In JSON mode, structured errors must be representable as compact JSON error envelopes on stderr.

#### JSON Lines Mode

- JSON Lines mode is available only for commands that enable `jsonl`.
- JSON Lines commands must not accept the framework `--json` flag before `--`; doing so fails before user parsing.
- Command logic must return an ordered iterable or stream of records.
- Each record is serialized as one compact JSON value followed by LF.
- `output.log` is suppressed and `output.error` still writes stderr.
- The result must retain records successfully emitted before a mid-stream failure.
- If a downstream stdout pipe closes while emitting JSON Lines, the invocation exits successfully as a normal early stop and must not run `afterRun` or `onRunError`.
- Records that cannot be serialized fail with structured kind `rune/serialization-failed`.

### Stdin

- Stdin exposes whether input is piped and whether the stream is interactive.
- Stdin can be consumed as text or bytes.
- Stdin may be consumed only once. A second consumption attempt fails with structured kind `rune/stdin-consumed`.
- Text decoding uses UTF-8.

### Structured Errors

Snok must expose a structured command error concept equivalent to Rune's `CommandError`.

| Field | Required behavior |
|---|---|
| `kind` | Stable machine-readable category string. |
| `message` | Human-readable failure summary. |
| `hint` | Optional recovery hint. |
| `details` | Optional JSON-compatible diagnostic value. |
| `exitCode` | Optional process exit code; invalid or omitted values normalize to 1. |

Parse failures normalize to kind `rune/invalid-arguments` and exit code 1. Unexpected execution failures normalize to kind `rune/unexpected` and exit code 1. JSON Lines result-shape failures normalize to kind `rune/invalid-command-result`.

### Help

- Help is generated from command, group, config, route, args, options, examples, aliases, output mode, and available subcommands.
- Default help for commands includes description when present, usage, options, arguments, subcommands when present, output contract when JSON Lines mode is enabled, and examples when present.
- Default help for groups includes description when present, usage, child commands with descriptions and aliases, root version option when version is configured, and examples when present.
- Unknown-command help includes the attempted path, available sibling commands, and scoped suggestions when present.
- Command-level help renderer takes precedence over global help renderer.
- Global help renderer takes precedence over default renderer for groups, unknown commands, and commands without command-level renderers.
- If a custom renderer fails, the default renderer must be used instead of failing the help request.

### Test Utilities

Snok must provide in-process command execution utilities equivalent to Rune's `runCommand` and `createRunCommand`.

- The test utility executes one resolved command without spawning a child process.
- It runs the same command-level parse, validation, default, output, locals, hook, stdin, and error pipeline as a real invocation.
- It does not perform top-level command routing unless a separate test helper explicitly covers routing.
- It returns exit code, captured stdout, captured stderr, structured error when present, and a discriminated output result: text, JSON document, or JSON Lines records.
- Test context can inject cwd, env, stdin, globals, hooks, locals, and agent-detection behavior.
- Injected env is a complete replacement for host environment unless the caller explicitly merges host values.

## 6. Interfaces

### Canonical Components

| Component | Purpose |
|---|---|
| Command definition | Declares one executable command's metadata, fields, output mode, help renderer, and run logic. |
| Group definition | Declares one non-executable command group's metadata. |
| Project configuration | Declares global display metadata, options, help renderer, hooks, and locals. |
| Structured command error | Reports user-facing and machine-readable failures. |
| Default help renderer | Converts structured help data into human-readable help text. |
| In-process command runner | Executes a single resolved command in tests. |

### Files, Data Formats, and Generated Artifacts

- Public command tree units are represented by a logical path model. A Go implementation may map that model to directories, files, registration functions, or generated manifests, but the resulting command tree must be inspectable and deterministic.
- JSON documents and JSON Lines records use UTF-8 JSON with LF line endings. JSON Lines emits one serialized JSON value per line.
- Error details must be JSON-compatible when serialized for machine-readable output.

### Binding Profiles

No separate binding profile is included. Snok's Go/Cobra implementation may choose idiomatic names and types as long as the observable behavior and public conceptual landmarks remain recognizable.

## 7. Data, State, and Lifecycle

- Command definitions, group definitions, and configuration are framework-owned metadata once registered.
- Parsed args/options, locals, output buffers, stdin consumption state, and hook context are invocation-scoped.
- A command invocation must not mutate registered command definitions or global configuration.
- Generated routing or help metadata must be deterministic for the same command tree.
- Temporary working artifacts must not appear in durable output.

## 8. Configuration, Dependencies, and Permissions

- Snok depends on Cobra as the command execution foundation, but Cobra-specific mechanics are not part of this portable behavior contract.
- Environment variable fallback reads only explicitly declared variable names.
- Commands are responsible for their own application permissions and credentials; Snok only defines framework parsing and execution behavior.
- Machine-readable output must avoid mixing human stdout into JSON or JSON Lines stdout streams.

## 9. Error Handling and Edge Cases

- Definition-time validation failures must identify the invalid command, group, option, argument, alias, output mode, or conflict.
- Parse failures must prevent locals, hooks, and command logic from running.
- Duplicate scalar options are parse failures. Repeatable options allow duplicates and preserve order.
- Unknown options and unknown short options are parse failures.
- Unknown command segments must produce route-level unknown-command help or errors rather than falling through to unrelated commands.
- Help requests must not run command logic.
- Version output must not require a valid command tree when version metadata is available.
- Broken-pipe handling is special only for framework stdout writes during JSON Lines streaming; command-thrown broken-pipe-like errors are normal command failures.

## 10. Quality Attributes and Constraints

- Agent readability: command structure, metadata, and tests should be easy to inspect mechanically.
- Determinism: route ordering, help data ordering, parse results, defaults, JSONL record order, and conformance fixture comparisons must be stable.
- Portability: normative behavior must be implementable without copying Rune's source language, runtime, or repository layout.
- Testability: command behavior must be testable in process with isolated cwd, env, stdin, stdout, stderr, hooks, and locals.
- Human and machine compatibility: the same command logic must serve text and structured execution without requiring duplicate implementations.

## 11. Verification and Acceptance Criteria

### Acceptance Criteria

- A Snok command tree can be inspected to determine commands, groups, aliases, ignored helpers, and tests without executing every command.
- A command with string, number, boolean, enum, schema-like, repeatable, defaulted, required, env-backed, and negatable fields parses according to this spec.
- Text, JSON, and JSON Lines commands all execute through the same lifecycle and produce isolated stdout/stderr contracts.
- Structured errors expose stable kind, message, optional hint/details, and normalized exit code.
- In-process tests can assert stdout, stderr, structured error, JSON document, JSON Lines records, cwd, env, stdin, hooks, and locals without spawning a process.

### Verification

- `specs/fixtures/routing.json` covers command tree derivation, alias routing, help detection, and unknown-command suggestions.
- `specs/fixtures/fields-and-parsing.json` covers field parsing, defaults, env precedence, repeated options, negation, and parse failures.
- `specs/fixtures/execution-output.json` covers execution context, text output, JSON mode, JSON Lines, structured errors, hooks, locals, stdin, and broken pipe behavior.
- `specs/fixtures/help-rendering.json` covers semantic help sections and precedence.

### Conformance Assets

- Bundle: `specs/`.
- Normative assets: all fixture files under `specs/fixtures/`.
- Evidence-only assets: `specs/evidence.md`.
- Comparison model: structured JSON equality unless a case explicitly states substring, ordered semantic equality, or byte-exact text.
- Deterministic text: help rendering cases use required sections and substrings unless marked byte-exact.
- Generative behavior: not applicable.

## 12. Decisions and Open Questions

### Decisions

- Snok targets Rune framework-core behavior, not Rune's scaffolding, package, build, or sync tooling.
- Snok preserves source public landmarks as concepts, not as source-binding API spellings.
- Snok uses Cobra idiomatically while enforcing this higher-level behavior contract.
- The conformance bundle is normative for deterministic behaviors and evidence-only for source-test inventory.

### Evidence Conflicts

| Contract topic | Observed implementation | Documentation, tests, or guidance | Extracted decision |
|---|---|---|---|
| Runtime/toolchain details | Rune package declares runtime and build tooling details. | Snok is a Go/Cobra wrapper. | Exclude source runtime and package mechanics as incidental implementation. |
| Help byte layout | Rune tests assert exact help text in places. | Snok will render through Cobra-compatible idioms. | Require semantic help sections and stable ordering; avoid byte-exact Rune text except fixture-marked substrings. |
| Source file shape | Rune routes command files in its source binding. | Snok will be Go/Cobra-based. | Specify logical command tree rules rather than source file mechanics. |

### Open Questions

There are no open questions blocking this specification.

### Unresolved Divergences

| Decision surface | Plausible implementation choices | Required resolution or conformance asset |
|---|---|---|
| Schema validation API | Go implementations may use different validators. | Treat schemas as validator contracts with raw input, omitted input, and transformed output semantics. |
| Agent detection | Implementations may detect agents differently. | Conformance tests must inject deterministic agent-detection state. |
| Help formatting | Cobra conventions may differ from Rune. | Use semantic section/order/substr comparison in help fixtures unless exact text is explicitly required. |
