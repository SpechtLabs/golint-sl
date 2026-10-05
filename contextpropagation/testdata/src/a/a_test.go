package a

import "context"

// Good: test files are skipped entirely
func helperInTest(ctx context.Context) {
	_ = context.Background()
}
