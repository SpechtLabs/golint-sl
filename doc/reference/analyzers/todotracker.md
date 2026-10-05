---
title: todotracker
permalink: /reference/analyzers/todotracker
createTime: 2025/01/16 10:00:00
---

Ensures TODO and FIXME comments have owners.

## Category

Architecture

## What It Checks

This analyzer detects TODO and FIXME comments without attribution.

A marker is the upper-case word `TODO` or `FIXME` at the start of a comment
line, including the continuation lines of a block comment. Words that only
contain the letters, such as `Mastodon`, lower-case prose, a marker in the
middle of a sentence and identifiers such as `context.TODO` are left alone.

## Why It Matters

Anonymous TODOs:

- Have no accountability
- Are often forgotten
- Accumulate over time
- Make triage difficult

## Examples

### Bad: Anonymous TODO

```go
// TODO: fix this later
func Process() {
    // FIXME: handle errors properly
}
```

### Good: Attributed TODO

```go
// TODO(alice): fix this later - JIRA-1234
func Process() {
    // FIXME(bob): handle errors properly
}
```

### Accepted Formats

```go
// TODO(username): description
// TODO(username): description - TICKET-123
// FIXME(username): description
```

## Why Attribution Matters

1. **Accountability**: Someone is responsible
2. **Contact**: Know who to ask about it
3. **Tracking**: Link to issue trackers
4. **Cleanup**: Easier to triage and prioritize

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  todotracker: true  # enabled by default
```

## When to Disable

- Personal projects
- Early prototyping
- Projects using external TODO tracking

```yaml
analyzers:
  todotracker: false
```

## Related Analyzers

- [exporteddoc](/reference/analyzers/exporteddoc) - Documentation requirements
