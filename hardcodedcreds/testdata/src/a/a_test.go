package a

// Good: test files are skipped; mock credentials are standard in tests
var testPassword = "hunter22"

func fixture() Config {
	apiToken := "AKIAIOSFODNN7EXAMPLE"
	return Config{Password: "hunter22", Token: apiToken}
}
