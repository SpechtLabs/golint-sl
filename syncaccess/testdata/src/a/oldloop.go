//go:build go1.21

package a

// Bad: before Go 1.22, every iteration shares the variable the for clause
// declares.
func sharedDeclaredLoopVars(xs []int) {
	for i := 0; i < 3; i++ {
		go func() {
			use(i) // want `loop variable "i" captured by goroutine; this may cause unexpected behavior - pass as parameter instead`
		}()
	}
	for _, v := range xs {
		go func() {
			use(v) // want `loop variable "v" captured by goroutine`
		}()
	}
}
