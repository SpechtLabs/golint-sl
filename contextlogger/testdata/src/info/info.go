package info

import (
	"context"
	stdlog "log"

	"example.com/log"
)

func FromContext(ctx context.Context) *log.Logger { return log.FromContext(ctx) }

func IntoContext(ctx context.Context, l *log.Logger) context.Context { return ctx }

func use(ctx context.Context) {
	a := FromContext(ctx)
	a.Info("a")
	b := log.FromContext(ctx)
	b.Info("b")
	log.Info("c")
	log.Error("d")
	stdlog.Printf("e")
	other()
}

func other() {}
