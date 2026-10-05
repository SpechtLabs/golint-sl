package a

import (
	"errors"
	"log"
	"regexp"

	"github.com/sirupsen/logrus"
)

type Logger struct{}

func (Logger) Fatal(msg string)  {}
func (Logger) Fatalw(msg string) {}
func (Logger) Info(msg string)   {}

// Good: returning an error.
func ParseConfig(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty config")
	}
	return data, nil
}

// Bad: panic in library code.
func MustParseConfig(data []byte) []byte {
	if len(data) == 0 {
		panic("empty config") // want `panic\(\) in library code; return an error instead to let callers handle failures gracefully`
	}
	return data
}

// Bad: the standard library log fatal and panic variants.
func stdlog() {
	log.Fatal("x")      // want `log.Fatal\(\) in library code terminates the program; return an error instead`
	log.Fatalf("x")     // want `log.Fatalf\(\) in library code terminates the program`
	log.Fatalln("x")    // want `log.Fatalln\(\) in library code terminates the program`
	log.Panic("x")      // want `log.Panic\(\) in library code terminates the program`
	log.Panicf("x")     // want `log.Panicf\(\) in library code terminates the program`
	log.Panicln("x")    // want `log.Panicln\(\) in library code terminates the program`
	log.Println("fine") // Good: not a fatal call
}

// Bad: the logrus fatal and panic variants.
func logrusCalls() {
	logrus.Fatal("x")   // want `logrus.Fatal\(\) in library code terminates the program`
	logrus.Fatalf("x")  // want `logrus.Fatalf\(\) in library code terminates the program`
	logrus.Fatalln("x") // want `logrus.Fatalln\(\) in library code terminates the program`
	logrus.Panic("x")   // want `logrus.Panic\(\) in library code terminates the program`
	logrus.Panicf("x")  // want `logrus.Panicf\(\) in library code terminates the program`
	logrus.Panicln("x") // want `logrus.Panicln\(\) in library code terminates the program`
	logrus.Info("fine") // Good: not a fatal call
}

// Bad: Fatal on a receiver named like a zap logger or "logger".
func zapStyle(zapLog Logger, logger Logger, other Logger) {
	zapLog.Fatal("x")   // want `Fatal log in library code terminates the program; return an error instead`
	logger.Fatalw("x")  // want `Fatal log in library code terminates the program`
	logger.Info("fine") // Good: not Fatal
	other.Fatal("x")    // Good: receiver name does not look like a logger
}

// Good: Must* helpers from other packages are not reported.
var re = regexp.MustCompile(`a+`)

// Good: calls through a non-identifier receiver, function literals and
// parenthesised functions are inspected without a report.
func otherCalls() {
	_ = re.FindString("aa")
	_ = regexp.MustCompile(`b+`).String()
	func() {}()
	(func() {})()
}

// Good: panics in init are allowed.
func init() {
	if re == nil {
		panic("bad regexp")
	}
}

// Good: panics in TestMain are allowed, even outside a _test.go file.
func TestMain() {
	panic("allowed")
}

// Good: suppressed by nolint.
func suppressed() {
	panic("unreachable") //nolint:nopanic
}
