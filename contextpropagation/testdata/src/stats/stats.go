package stats

import "context"

func a(ctx context.Context) { _ = ctx }
func b(_ context.Context)   {}
func c()                    {}
func d(n int)               {}
func e(context.Context)     {}
