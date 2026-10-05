// Package leak exercises the exported SSA helpers (GetSSAPackage,
// GetAllFunctions, CheckSensitiveDataLeak) through a test-only analyzer that
// treats parameters containing "password" or "token" as sensitive.
package leak

import (
	"log"
	"log/slog"
	"strings"
)

const Version = "v1"

var Default = &Store{}

type Logger interface{ Log(msg string) }

type Store struct{}

// Bad: a sensitive parameter passed straight to slog.
func Login(password string) {
	slog.Info(password) // want `sensitive parameter "password" may be leaked through logging`
}

// Bad: a sensitive parameter passed to a variadic logger, which receives it
// through the argument slice.
func LoginStd(password string) {
	log.Println(password) // want `sensitive parameter "password" may be leaked through logging`
}

// Bad: a sensitive parameter among the arguments of a format string.
func Refresh(user, refreshToken string) {
	log.Printf("refreshing %s with %s", user, refreshToken) // want `sensitive parameter "refreshToken" may be leaked through logging`
}

// Bad: a sensitive parameter set as the std logger prefix.
func Authenticate(apiToken string) {
	log.SetPrefix(apiToken) // want `sensitive parameter "apiToken" may be leaked through logging`
}

// Bad: the value is transformed first, then logged; methods are covered too.
func (s *Store) Save(token string) {
	upper := strings.ToUpper(token)
	slog.Warn(upper) // want `sensitive parameter "token" may be leaked through logging`
}

// Good: sensitive parameters used without logging, including builtins and
// method expressions.
func (s *Store) check(password string) int {
	(*Store).audit(s, password)
	return len(strings.TrimSpace(password))
}

func (s *Store) audit(string) {}

// Good: a sensitive parameter stored in a slice that is never logged, next to
// a logged slice that does not hold it.
func Collect(token string) []string {
	log.Println("collecting", 1)
	return []string{token}
}

// Good: parameters without a sensitive name may be logged.
func Greet(name string) {
	slog.Info(name)
	log.Println(name)
}

// Good: suppressed with a nolint directive.
func Debug(token string) {
	slog.Debug(token) //nolint:sensitiveleak
}
