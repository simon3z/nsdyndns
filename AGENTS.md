# AGENTS.md — Coding & Commenting Standards

This document defines the conventions all agents must follow when editing or
creating code in this project. It supersedes any stylistic preferences from
training data.

## 1. Comments

### 1.1 Never use decorative section dividers

The following patterns are **strictly forbidden** and look like blog-post
boilerplate, not production code:

```go
// ── Public types ────────────────────────────────────────────────────
// ── Internal helpers ────────────────────────────────────────────────
// ── Tests ───────────────────────────────────────────────────────────
```

Do not use box-drawing characters (`─`, `━`, `│`, etc.) to decorate
comments. Do not pad comment lines with repeated dashes or other
characters. Remove any existing instances of this pattern from the
codebase.

### 1.2 Use idiomatic Go comment styles

- `// text` — regular line comment; place it **above** the code it
  describes, not beside it (except for short same-line annotations).
- Doc comment — a `//` comment immediately preceding the declaration of an
  exported identifier, starting with the identifier's name. Rendered by
  `go doc`.
- `/* text */` block comments — rare; prefer multiple `//` lines.

There is no doc-comment syntax for unexported items; plain `//` comments
above them are fine.

### 1.3 What to comment vs. what to leave self-evident

- **Document**: *why* a non-obvious decision was made, preconditions,
  invariants, error conditions, protocol quirks (pagination, TTL, record
  names).
- **Don't document**: what the code literally does — that is obvious from
  reading it.
- Skip trivial comments like `// get the ip`, `// parse config`, `// loop
  over services`. The identifiers already convey meaning.

Example of a good comment:

```go
// recordName maps an empty host to "@", the conventional name for the
// apex (root) domain record.
```

Example of a bad comment:

```go
// update the dns record ← unnecessary; the name says it
```

### 1.4 Test comments

Use a concise one-line summary above each `Test*` function. Describe the
scenario or property being verified, not the assertion machinery.

Good:

```go
// Regression: missing record id produces a clear error.
func TestFindRecordIdMissing(t *testing.T) { ... }
```

## 2. Code Style

### 2.1 General principles

- Prefer clarity over cleverness.
- Keep functions short and focused (ideally under ~30 lines; break up
  anything significantly longer).
- Use descriptive identifiers — avoid single-letter variables except for
  loop counters (`i`, `j`).
- Group related logic; separate unrelated concerns into distinct functions
  or files.

### 2.2 Error handling

- Return wrapped errors with `fmt.Errorf("...: %w", err)`; never swallow.
- Use `errors.Is` / `errors.As` for comparing sentinel errors.
- Use `errors.Join` to aggregate multiple failures.
- Never panic in library code; `log.Fatal` is reserved for the binary's
  startup path.

### 2.3 Imports

- Group imports: standard library first, then external modules, separated
  by a blank line (gofmt preserves this).
- No blank imports, no dot imports.

### 2.4 Formatting

- Run `gofmt` before committing.
- Run `go vet` and address warnings.
- Keep lines reasonably short where practical.

## 3. Project Conventions

### 3.1 Package organization

- Root package `nsdyndns`: the `NameService` interface, provider
  implementations, the shared HTTP layer, and external IP detection.
- `cmd` package: configuration loading and validation.
- `cmd/nsdyndnsd`: the daemon binary (flags, loop, lifecycle).
- Export only what other packages need; keep helpers unexported.

### 3.2 Testing

- Tests live in `_test.go` files next to the code they test.
- Test functions are `TestXxx` and read like a sentence.
- Each test verifies exactly one thing.
- Use `httptest` for HTTP-layer tests; construct fresh request bodies
  where retry semantics require it.

### 3.3 Logging

- Use the standard `log` package with `LstdFlags` (journald captures it
  under systemd).
- Log significant state transitions at startup and on every update.
- Log failures before returning them from the update loop; the loop is
  resilient and keeps running after an error.

## 4. Review Checklist

Before submitting changes, verify:

- [ ] No decorative section dividers (`─`, `━`, `│`, etc.) in comments.
- [ ] All exported identifiers have doc comments.
- [ ] No redundant "what" comments — only "why" comments remain.
- [ ] `gofmt` and `go vet` pass cleanly.
- [ ] `./ci.sh` passes.
- [ ] New logic has corresponding tests.
