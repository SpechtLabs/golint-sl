// Package a exercises the todotracker analyzer.
//
// The analyzer inspects every comment, including the ones that carry the
// analysistest expectations and these explanations. A marker only counts as
// the first word of a comment line, so the explanations never start a line
// with one, and each expectation either shares the marker's comment or spells
// the marker with a regexp wildcard.
package a

import "context"

// Good: comments without a marker are ignored.

// Good: owner and description.
// TODO(alice): implement retry logic for transient failures

// Good: team owner on the second marker kind.
// FIXME(@team-platform): this breaks when input exceeds 1MB

// Good: tracker reference as the owner.
// TODO(jira-PROJ-123): add caching layer

// Good: spaces around the owner are tolerated.
// TODO (bob) : spaced out but complete

// Good: a word that merely contains the letters is not a marker.
// Mastodon client for the notification fan-out.

// Good: lower-case prose is not a marker.
// todo (bob) : lower-case words are not markers
// fixme later

// Good: a marker in the middle of a line is not a marker.
// Keep the TODO list short.

// Good: a longer word that starts with the letters is not a marker.
// TODOs pile up when nobody owns them.

// Good: a well-formed marker on a continuation line of a block comment.
/*
 * TODO(alice): add caching
 */

// Good: identifiers such as context.TODO are not markers.
// Fall back to context.TODO() until every caller passes a context.
func g() context.Context {
	return context.TODO() // context.TODO is fine here
}

// Bad: no owner at all.
// TODO fix this // want "TODO without owner"

// Bad: the bare second marker kind.
// FIXME // want "FIXME without owner"

// Bad: a dash instead of an owner.
// TODO - make this better // want "TODO without owner"

// Bad: owner but no description.
// TODO(alice) implement retries // want "TODO without description"

// Bad: owner but no description, second marker kind.
// FIXME(bob) handle overflow // want "FIXME without description"

// Bad: empty owner.
// TODO(): add metrics // want "TODO appears malformed"

// Bad: a colon before the owner.
// TODO: see helper(x) // want "TODO appears malformed"

// Bad: the closing marker of a block comment is not a description.
/* TODO(alice): */ // want "T.DO appears malformed"

// Bad: a marker on a continuation line of a block comment.
/* first line
   FIXME handle overflow */ // want "F.XME without owner"

// Good: suppressed by a nolint directive on the line before.
//nolint:todotracker
// TODO fix this later

// Good: suppressed by the catch-all nolint directive.
//nolint:golint-sl
// FIXME someday

// Good: suppressed by an inline nolint directive.
/* TODO inline */ //nolint:todotracker

func f() int {
	x := 1 /* TODO bump the default */ // want "T.DO without owner"
	return x
}
