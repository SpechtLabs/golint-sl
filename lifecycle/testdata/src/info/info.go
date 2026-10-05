package info // want `^run=\[both runOnly\] stop=\[both stopOnly\] missing=\[runOnly\] both=\[both\]$`

import "context"

// both has Run and Stop.
type both struct{}

func (b *both) Run(ctx context.Context) error { return nil }

func (b *both) Stop() {}

// runOnly is missing a stop method.
type runOnly struct{}

func (r runOnly) Start(ctx context.Context) error { return nil }

// stopOnly only has a stop method.
type stopOnly struct{}

func (s *stopOnly) Close() error { return nil }

// Plain functions are skipped.
func Run() {}
