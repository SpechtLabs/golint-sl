package a

import (
	"errors"
	"fmt"
	"log"
	"log/slog"

	"example.com/catalog"
)

// Bad: sensitive values passed as variadic arguments
func Variadic(password string) {
	fmt.Println(password)              // want `sensitive parameter "password" may be logged`
	log.Printf("%s", password)         // want `sensitive parameter "password" may be logged`
	slog.Info("login", "pw", password) // want `sensitive parameter "password" may be logged`
}

// Bad: a slice literal holding a sensitive value, spread into a print call
func SliceLiteral(token string) {
	args := []any{"token", token}
	fmt.Println(args...) // want `sensitive parameter "token" may be logged`
}

func loginUser(user, pass string) bool { return user == pass }

// Good: a function whose name contains "log" is not a logger
func Login(password string) bool {
	return loginUser("u", password)
}

// Good: neither is a package whose name contains "log"
func AddSecret(secret string) {
	catalog.Add(secret)
}

// Good: fmt functions that only build values
func Describe(password string) (string, error) {
	return fmt.Sprintln(password), fmt.Errorf("bad password %q", password)
}

// Good: a variadic call that doesn't log
func JoinErrors(token string) error {
	return errors.Join(errors.New(token), errors.New("x"))
}
