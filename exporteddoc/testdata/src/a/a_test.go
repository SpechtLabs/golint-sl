package a

// Good: test files are skipped entirely, so none of these undocumented
// exported symbols are reported.

func Helper() {}

type Fixture struct{}

var Golden = "x"
