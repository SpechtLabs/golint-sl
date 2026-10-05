package a

import "errors"

var errBad = errors.New("bad")

// Good: early returns keep the function flat.
func GetItem(m map[string]int, id string) (int, error) {
	v, ok := m[id]
	if !ok {
		return 0, errBad
	}

	if v < 0 {
		return 0, errBad
	}

	return v, nil
}

// Good: depth of exactly 3 is allowed (if > for > switch).
func depthThree(xs []int) {
	if len(xs) > 0 {
		for _, x := range xs {
			switch x {
			case 1:
			}
		}
	}
}

// Bad: if > for > switch > if is depth 4.
func depthFour(xs []int) { // want `function "depthFour" has nesting depth of 4 \(max 3\); use early returns to flatten the code`
	if len(xs) > 0 {
		for _, x := range xs {
			switch x {
			case 1:
				if x > 0 {
					println(x)
				}
			}
		}
	}
}

// Bad: the depth counts through for, type switch, select and else branches.
func depthFive(v any, ch chan int) { // want `function "depthFive" has nesting depth of 5 \(max 3\)`
	for i := 0; i < 1; i++ {
		switch v.(type) {
		case int:
			select {
			case <-ch:
				if i == 0 {
					println(i)
				} else {
					for {
						break
					}
				}
			}
		}
	}
}

// Bad: an else-if branch is at the same depth as its if, and the deepest
// branch wins even when an earlier sibling is shallower.
func depthElseIf(a, b int) { // want `function "depthElseIf" has nesting depth of 4 \(max 3\)`
	for {
		if a > 0 {
			println(a)
		} else if b > 0 {
			for range a {
				for range b {
				}
			}
		} else if a < b {
		}
		if a == b {
		}
		break
	}
}

// Good: function literals do not add to the enclosing depth.
func withClosure(xs []int) {
	if len(xs) > 0 {
		f := func() {
			for range xs {
				for range xs {
					for range xs {
					}
				}
			}
		}
		f()
	}
}

// Good: depth diagnostic suppressed by nolint.
//
//nolint:nestingdepth
func suppressedDepth(xs []int) {
	for range xs {
		for range xs {
			for range xs {
				for range xs {
				}
			}
		}
	}
}

// Bad: three branches where the first ends in return.
func chainThree(x int) int {
	if x > 10 { // want `if-else chain with 3 branches; consider using early returns to flatten`
		return 10
	} else if x > 5 {
		return 5
	} else {
		return 0
	}
}

// Good: three branches where the first does not end in return.
func chainNoReturn(x int) int {
	y := 0
	if x > 10 {
		y = 10
	} else if x > 5 {
		y = 5
	} else {
		y = 1
	}
	return y
}

// Good: three branches with an empty first body.
func chainEmptyBody(x int) {
	if x > 10 {
	} else if x > 5 {
		println(x)
	} else {
		println(-x)
	}
}

// Good: two branches are within the limit.
func chainTwo(x int) int {
	if x > 0 {
		return 1
	} else {
		return 0
	}
}

// Bad: a nested if that is the only statement of its parent.
func nestedIf(a, b bool) {
	if a {
		if b { // want `nested if statements could be combined with && operator`
			println("both")
		}
	}
}

// Good: the outer init is fine; the inner init blocks combination.
func nestedInnerInit(m map[string]int) {
	if len(m) > 0 {
		if v, ok := m["k"]; ok {
			println(v)
		}
	}
}

// Good: an else on either level blocks combination.
func nestedElse(a, b bool) {
	if a {
		if b {
			println("b")
		} else {
			println("not b")
		}
	}
	if a {
		if b {
			println("b")
		}
	} else {
		println("not a")
	}
}

// Good: two statements in the outer body.
func nestedTwoStmts(a, b bool) {
	if a {
		println("a")
		if b {
			println("b")
		}
	}
}

// Good: nested-if diagnostic suppressed by nolint.
func nestedSuppressed(a, b bool) {
	if a {
		if b { //nolint:nestingdepth
			println("both")
		}
	}
}

func step() error { return nil }

// Good: five error checks are within the limit; non-binary and
// non-err conditions are not counted.
func fiveErrChecks(ok bool, n int) error {
	if err := step(); err != nil {
		return err
	}
	if err := step(); nil != err {
		return err
	}
	if err := step(); err != nil {
		return err
	}
	if err := step(); err != nil {
		return err
	}
	if err := step(); err != nil {
		return err
	}
	if ok {
		return nil
	}
	if n > 0 {
		return nil
	}
	return nil
}

// Bad: six error checks, with err on either side of the comparison.
func sixErrChecks() error { // want `function "sixErrChecks" has 6 error checks; consider extracting helper functions`
	if err := step(); err != nil {
		return err
	}
	if err := step(); nil != err {
		return err
	}
	if err := step(); err != nil {
		return err
	}
	if err := step(); err != nil {
		return err
	}
	if err := step(); err != nil {
		return err
	}
	if err := step(); err != nil {
		return err
	}
	return nil
}
