// Package a exercises the tracker analyzer.
//
// The analyzer inspects every comment, including the ones that carry the
// analysistest expectations and these explanations, so the explanations never
// spell a marker, and each expectation either shares the marker's comment or
// spells the marker with a regexp wildcard.
package a

// Good: comments without a marker are ignored.

// Good: owner and description.
// TODO(alice): implement retry logic for transient failures

// Good: team owner on the second marker kind.
// FIXME(@team-platform): this breaks when input exceeds 1MB

// Good: tracker reference as the owner.
// TODO(jira-PROJ-123): add caching layer

// Good: the match is case-insensitive and tolerates spaces.
// todo (bob) : lower-case markers are accepted too

// Bad: no owner at all.
// TODO fix this // want "TODO without owner"

// Bad: the bare second marker kind.
// FIXME // want "FIXME without owner"

// Bad: a dash instead of an owner.
// TODO - make this better // want "TODO without owner"

// Bad: a lower-case marker is reported with the upper-case name.
// fixme later // want "FIXME without owner"

// Bad: owner but no description.
// TODO(alice) implement retries // want "TODO without description"

// Bad: owner but no description, second marker kind.
// FIXME(bob) handle overflow // want "FIXME without description"

// Bad: empty owner.
// TODO(): add metrics // want "TODO appears malformed"

// Bad: a colon before the owner.
// TODO: see helper(x) // want "TODO appears malformed"

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
