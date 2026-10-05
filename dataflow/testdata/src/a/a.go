package a

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"

	"go.uber.org/zap"
)

type Creds struct {
	User string
	Pass string
}

// Good: a parameter whose name is not sensitive may be logged
func LogUser(username string) zap.Field {
	return zap.String("user", username)
}

// Bad: sensitive parameters reaching non-variadic logging helpers
func LogToken(token string) zap.Field {
	return zap.String("token", token) // want `sensitive parameter "token" may be logged; sanitize or redact before logging`
}

func LogSecret(secret any) zap.Field {
	return zap.Any("secret", secret) // want `sensitive parameter "secret" may be logged`
}

func LogPassword(password string) slog.Attr {
	return slog.String("password", password) // want `sensitive parameter "password" may be logged`
}

// Bad: sensitive values used as the format string of print/log functions
func PrintfPassword(password string) {
	fmt.Printf(password) // want `sensitive parameter "password" may be logged`
}

func LogfAPIKey(apiKey string) {
	log.Printf(apiKey) // want `sensitive parameter "apiKey" may be logged`
}

func FprintfToken(w io.Writer, token string) {
	fmt.Fprintf(w, token) // want `sensitive parameter "token" may be logged`
}

// Bad: a slice of sensitive values spread into a print call
func PrintCredentials(credentials []any) {
	fmt.Println(credentials...) // want `sensitive parameter "credentials" may be logged`
}

// Good: Sprintf and Sprint only build strings
func FormatSecret(secret string) string {
	return fmt.Sprintf(secret)
}

func JoinSecrets(secrets []any) string {
	return fmt.Sprint(secrets...)
}

// Bad: the value flows through another call first
func LogDerived(password string) zap.Field {
	upper := strings.ToUpper(password)
	return zap.String("pw", upper) // want `sensitive parameter "password" may be logged`
}

// Bad: the value flows through a phi node
func LogMaybeRedacted(token string, redact bool) zap.Field {
	if redact {
		token = "***"
	}
	return zap.String("token", token) // want `sensitive parameter "token" may be logged`
}

// Bad: the value flows through a struct field access
func LogCredUser(cred *Creds) zap.Field {
	return zap.String("user", cred.User) // want `sensitive parameter "cred" may be logged`
}

// Bad: the value flows through a type assertion
func LogAsserted(secret any) zap.Field {
	s := secret.(string)
	return zap.String("s", s) // want `sensitive parameter "secret" may be logged`
}

// Good: sensitive values passed to functions that don't log
func store(password string) {}

func Save(password string) {
	store(password)
}

// Good: generic callees (no package on the instance) are not treated as logging
func remember[T any](v T) {}

func Remember(token string) {
	remember(token)
}

// Good: an unused sensitive parameter
func Ignore(secret string) {}

// Suppressed via nolint
func Suppressed(token string) zap.Field {
	return zap.String("token", token) //nolint:dataflow
}

// Context propagation: callees that expect a context get one.
type Doer interface{ Do() }

func needsCtx(ctx context.Context, n int) {}

func noArgs() {}

func Propagates(ctx context.Context, d Doer, f func()) {
	needsCtx(ctx, 1)
	noArgs()
	d.Do()
	f()
}

func NoContext() {
	needsCtx(context.Background(), 1)
}

// Bad: an API key, named in camelCase.
func PrintAPIKey(apiKey string) {
	fmt.Println(apiKey) // want `sensitive parameter "apiKey" may be logged`
}

// Good: a bare key is a map or config key, not a credential.
func PrintConfigKey(key string) {
	fmt.Printf("%s\n", key)
}

// Good: names that only contain a credential word don't name a credential.
func PrintAuthor(author, tokenizer, monkey string) {
	fmt.Println(author, tokenizer, monkey)
}

// Good: neither does a name where the credential word qualifies another.
func PrintSecretName(secretName string) {
	log.Printf("using secret %s", secretName)
}
