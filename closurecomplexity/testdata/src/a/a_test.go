package a

// Good: closures in test files are skipped, however complex
func helperForTests(a bool) {
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
