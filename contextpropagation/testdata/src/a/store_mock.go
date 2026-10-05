package a

import "context"

// Good: files ending in _mock.go are skipped
func fakeGet(ctx context.Context) {}
