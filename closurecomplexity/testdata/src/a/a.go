package a

import "fmt"

type Command struct {
	Use  string
	RunE func() error
	Run  func()
}

type Server struct {
	Handler func()
	Other   func()
}

// Visitor-style helpers whose callbacks are exempt by name.
func ForEach(xs []int, fn func(int)) {
	for _, x := range xs {
		fn(x)
	}
}

func Walk(fn func()) { fn() }

func cleanup() {}

// Good: a short closure
func Short(xs []int) int {
	sum := 0
	add := func(x int) {
		sum += x
	}
	for _, x := range xs {
		add(x)
	}
	return sum
}

// Bad: too many statements
func Long() {
	f := func() { // want `closure has \d+ statements \(max 15\); extract complex logic into a named function for testability`
		fmt.Println(1)
		fmt.Println(2)
		fmt.Println(3)
		fmt.Println(4)
		fmt.Println(5)
		fmt.Println(6)
		fmt.Println(7)
		fmt.Println(8)
		fmt.Println(9)
		fmt.Println(10)
		fmt.Println(11)
		fmt.Println(12)
		fmt.Println(13)
		fmt.Println(14)
		fmt.Println(15)
		fmt.Println(16)
		fmt.Println(17)
		fmt.Println(18)
	}
	f()
}

// Bad: if -> for -> if is three levels deep
func DeepFor(a bool) {
	f := func() { // want `closure has nesting depth of 3 \(max 2\); extract into a named function`
		if a {
			for {
				if a {
					break
				}
			}
		}
	}
	f()
}

// Bad: the third level is a range loop
func DeepRange(a bool, xs []int) {
	f := func() { // want `closure has nesting depth of 3 \(max 2\)`
		for {
			if a {
				for range xs {
				}
			}
			break
		}
	}
	f()
}

// Bad: the third level is a switch
func DeepSwitch(a bool, x int) {
	f := func() { // want `closure has nesting depth of 3 \(max 2\)`
		if a {
			if a {
				switch x {
				case 1:
				}
			}
		}
	}
	f()
}

// Bad: the third level is a type switch
func DeepTypeSwitch(a bool, v any) {
	f := func() { // want `closure has nesting depth of 3 \(max 2\)`
		if a {
			if a {
				switch v.(type) {
				case int:
				}
			}
		}
	}
	f()
}

// Bad: the third level is a select
func DeepSelect(a bool, ch chan int) {
	f := func() { // want `closure has nesting depth of 3 \(max 2\)`
		if a {
			if a {
				select {
				case <-ch:
				default:
				}
			}
		}
	}
	f()
}

// Bad: a bare block does not add a level, but its contents do
func DeepBlock(a bool) {
	f := func() { // want `closure has nesting depth of 3 \(max 2\)`
		{
			if a {
				if a {
					if a {
					}
				}
			}
		}
	}
	f()
}

// Bad: nesting under an else-if branch
func DeepElseIf(a, b bool) {
	f := func() { // want `closure has nesting depth of 3 \(max 2\)`
		if a {
		} else if b {
			if a {
				if b {
				}
			}
		}
	}
	f()
}

// Bad: nesting under an else block
func DeepElse(a bool) {
	f := func() { // want `closure has nesting depth of 3 \(max 2\)`
		if a {
		} else {
			if a {
				if a {
					if a {
					}
				}
			}
		}
	}
	f()
}

// Good: two levels of nesting is allowed, with an else that stays shallow
func Shallow(a bool) {
	f := func() {
		if a {
			if a {
			}
		} else {
		}
	}
	f()
}

// Bad: captures six locals of the enclosing function
func Captures(p1, p2 int) {
	var v1 int
	v2 := 2
	v3, v4 := 3, 4
	f := func() int { // want `closure captures 6 variables from outer scope \(max 5\); consider passing them as parameters`
		return p1 + p2 + v1 + v2 + v3 + v4
	}
	_ = f()
}

// Good: parameters, builtins and closure-local names don't count as captures
func FewCaptures(xs []int) {
	n, m, k, j := 1, 2, 3, 4
	var s struct{ x int }
	s.x = 1
	f := func(n, m int) int {
		inner := func() int { return k }
		return len(xs) + n + m + j + inner()
	}
	_ = f(n, m)
	_ = s
}

// Good: deferred closures are exempt
func Deferred(a bool) {
	defer func() {
		if a {
			if a {
				if a {
				}
			}
		}
	}()
	defer cleanup()
}

// Good: goroutine closures are exempt
func Goroutine(a bool) {
	go func() {
		if a {
			if a {
				if a {
				}
			}
		}
	}()
	go cleanup()
}

// Good: returned closures (handler factories) are exempt
func Factory(a bool) (func(), int) {
	return func() {
		if a {
			if a {
				if a {
				}
			}
		}
	}, 0
}

// Good: visitor callbacks are exempt, other arguments are left alone
func Visitors(a bool, xs []int) {
	ForEach(xs, func(int) {
		if a {
			if a {
				if a {
				}
			}
		}
	})
	Walk(cleanup)
	fmt.Println(xs)
}

// Good: Cobra command and HTTP handler fields are exempt
func Fields(a bool) (*Command, *Server) {
	cmd := &Command{
		Use: "x",
		RunE: func() error {
			if a {
				if a {
					if a {
					}
				}
			}
			return nil
		},
		Run: cleanup,
	}
	srv := &Server{
		Handler: func() {
			if a {
				if a {
					if a {
					}
				}
			}
		},
		// Bad: any other field is checked
		Other: func() { // want `closure has nesting depth of 3`
			if a {
				if a {
					if a {
					}
				}
			}
		},
	}
	return cmd, srv
}

// Bad: map literal values with non-identifier keys are checked
func MapOfFuncs(a bool) map[string]func() {
	m := map[string]func(){
		"x": func() { // want `closure has nesting depth of 3`
			if a {
				if a {
					if a {
					}
				}
			}
		},
	}
	return m
}

// Suppressed via nolint
func Suppressed(a bool) {
	//nolint:closurecomplexity
	f := func() {
		if a {
			if a {
				if a {
				}
			}
		}
	}
	f()
}
