package a

import (
	"errors"
	"log"
	"regexp"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
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

// Good: Fatal methods of a type outside the known logging packages may do
// anything, whatever the variable is called.
func localLogger(zapLog Logger, logger Logger, other Logger) {
	zapLog.Fatal("x")
	logger.Fatalw("x")
	logger.Info("fine")
	other.Fatal("x")
}

// Server keeps its loggers in fields.
type Server struct {
	logger *log.Logger
	zlog   *zap.Logger
	sugar  *zap.SugaredLogger
	lr     *logrus.Logger
}

// Bad: logger methods are matched by type, wherever the logger comes from:
// a struct field, a call result or a parameter.
func (s *Server) stop(err error, std *log.Logger) {
	s.logger.Fatal("x")                 // want `Fatal log in library code terminates the program; return an error instead`
	s.logger.Panicf("x")                // want `Panic log in library code terminates the program; return an error instead`
	std.Fatalln("x")                    // want `Fatal log in library code terminates the program`
	s.zlog.Fatal("x")                   // want `Fatal log in library code terminates the program`
	s.zlog.Panic("x")                   // want `Panic log in library code terminates the program`
	s.sugar.Fatalw("x")                 // want `Fatal log in library code terminates the program`
	s.zlog.Sugar().Panicw("x")          // want `Panic log in library code terminates the program`
	zap.L().Fatal("x")                  // want `Fatal log in library code terminates the program`
	zap.S().Fatalw("x")                 // want `Fatal log in library code terminates the program`
	s.lr.Fatal("x")                     // want `Fatal log in library code terminates the program`
	s.lr.WithError(err).Fatalf("x")     // want `Fatal log in library code terminates the program`
	logrus.WithError(err).Fatal("x")    // want `Fatal log in library code terminates the program`
	logrus.New().Panicf("x")            // want `Panic log in library code terminates the program`
	logrus.WithError(err).Panicln("x")  // want `Panic log in library code terminates the program`
	s.logger.Println("fine")            // Good: not a fatal call
	s.zlog.Info("fine")                 // Good: not a fatal call
	s.zlog.DPanic("fine")               // Good: DPanic only panics in development
	s.sugar.Infow("fine")               // Good: not a fatal call
	logrus.WithError(err).Error("fine") // Good: not a fatal call
}

// Good: a local function called panic is not the builtin.
func shadowed() {
	panic := func(string) {}
	panic("not the builtin")
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

// Bad: package-level code after init is not inside init.
var afterInit = func() int {
	panic("x") // want `panic\(\) in library code`
}()

// Bad: a method called init is not the package initializer.
func (Logger) init() {
	panic("x") // want `panic\(\) in library code`
}

// Good: function literals inside init belong to init.
func init() {
	func() { panic("bad init") }()
}

// Good: panics in TestMain are allowed, even outside a _test.go file.
func TestMain() {
	panic("allowed")
}

// Good: suppressed by nolint.
func suppressed() {
	panic("unreachable") //nolint:nopanic
}

// wrapped is a logger that wraps zap, as otelzap does.
type wrapped struct{ l *zap.Logger }

// Good: a wrapper's Fatal and Panic methods terminate by design.
func (w *wrapped) Fatal(msg string) { w.l.Fatal(msg) }

func (w *wrapped) Panicf(msg string) { w.l.Panic(msg) }

func (w *wrapped) DPanicw(msg string) { w.l.Panic(msg) }

func (w *wrapped) FatalContext(msg string) { w.l.Fatal(msg) }

// Bad: a method that only starts like one isn't a log wrapper.
func (w *wrapped) PanicHandler(msg string) {
	w.l.Panic(msg) // want `Panic log in library code terminates the program`
}

// Bad: neither is a wrapper's other methods.
func (w *wrapped) Info(msg string) {
	w.l.Fatal(msg) // want `Fatal log in library code terminates the program`
}
