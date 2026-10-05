// Package callbacks consists of framework callbacks, where fmt.Errorf is
// allowed. It is analyzed alongside package plain to check that one
// package's callback exemption never leaks into another package.
package callbacks

import "fmt"

func requestHandler() error { return fmt.Errorf("handler: %d", 1) }

func authMiddleware() error { return fmt.Errorf("middleware: %d", 2) }

func unaryInterceptor() error { return fmt.Errorf("interceptor: %d", 3) }

func doneCallback() error { return fmt.Errorf("callback: %d", 4) }

func startHook() error { return fmt.Errorf("hook: %d", 5) }

func streamHandler() error { return fmt.Errorf("handler: %d", 6) }
