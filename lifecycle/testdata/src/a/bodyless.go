package a

import "context"

// Good: Run without a body (implemented in assembly) skips the context check.
type asm struct{}

func (a *asm) Run(ctx context.Context) error

func (a *asm) Close() error { return nil }
