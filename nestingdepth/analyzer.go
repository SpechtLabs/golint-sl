// Package nestingdepth provides an analyzer that enforces shallow nesting and early returns.
//
// Deeply nested code (indentation hell) makes code hard to read and maintain.
// This analyzer enforces early returns and suggests function extraction.
package nestingdepth

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
const Doc = `enforce shallow nesting depth and early returns

This analyzer detects:
1. Functions with nesting depth > 3
2. If-else chains of more than two branches whose first branch returns
3. Nested if statements that could be combined with &&
4. Functions with more than five error checks, which should be split into
   smaller helpers

Deep nesting (indentation hell) causes:
- Reader fatigue from parsing complex logic
- Difficulty testing all code paths
- Hard to track logical states
- Exponential complexity as logic grows

Good pattern (early return):
    func ActiveItems(ids []string) ([]Item, error) {
        var items []Item
        for _, id := range ids {
            item, ok := cache.Get(id)
            if !ok || !item.Active {
                continue
            }

            if item.Expired() {
                return nil, ErrExpired
            }

            items = append(items, item)
        }
        return items, nil
    }

Bad pattern (deep nesting, depth 4):
    func ActiveItems(ids []string) ([]Item, error) {
        var items []Item
        for _, id := range ids {
            if item, ok := cache.Get(id); ok {
                if item.Active {
                    if item.Expired() {
                        return nil, ErrExpired
                    }
                    items = append(items, item)
                }
            }
        }
        return items, nil
    }`

// Analyzer reports deeply nested code, long if-else chains and combinable nested ifs.
var Analyzer = &analysis.Analyzer{
	Name:     "nestingdepth",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// MaxNestingDepth is the maximum allowed nesting depth
const MaxNestingDepth = 3

// MaxIfElseChain is the maximum allowed if-else chain length
const MaxIfElseChain = 2

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		checkFunction(reporter, fn)
	})

	return nil, nil
}

func checkFunction(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	// Check overall nesting depth
	maxDepth := calculateMaxDepth(fn.Body, 0)
	if maxDepth > MaxNestingDepth {
		reporter.Reportf(fn.Pos(),
			"function %q has nesting depth of %d (max %d); use early returns to flatten the code",
			fn.Name.Name, maxDepth, MaxNestingDepth)
	}

	// Check for if-else chains that should be early returns
	checkIfElseChains(reporter, fn.Body)

	// Check for nested ifs that could be combined
	checkNestedIfs(reporter, fn.Body)

	// Check for functions that are too long and should be split
	checkFunctionLength(reporter, fn)
}

// calculateMaxDepth calculates the maximum nesting depth in a block
func calculateMaxDepth(node ast.Node, currentDepth int) int {
	maxDepth := currentDepth

	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
			*ast.TypeSwitchStmt, *ast.SelectStmt:
			// These increase nesting
			depth := calculateMaxDepthInner(n, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
			return false // We handle children ourselves

		case *ast.FuncLit:
			// Don't count nested function literals as part of parent depth
			return false
		}

		return true
	})

	return maxDepth
}

func calculateMaxDepthInner(node ast.Node, currentDepth int) int {
	maxDepth := currentDepth

	switch n := node.(type) {
	case *ast.IfStmt:
		// Check body
		bodyDepth := calculateMaxDepth(n.Body, currentDepth)
		if bodyDepth > maxDepth {
			maxDepth = bodyDepth
		}

		// Check else
		if n.Else != nil {
			elseDepth := currentDepth
			if elseBlock, ok := n.Else.(*ast.BlockStmt); ok {
				elseDepth = calculateMaxDepth(elseBlock, currentDepth)
			} else if elseIf, ok := n.Else.(*ast.IfStmt); ok {
				elseDepth = calculateMaxDepthInner(elseIf, currentDepth)
			}
			if elseDepth > maxDepth {
				maxDepth = elseDepth
			}
		}

	case *ast.ForStmt:
		bodyDepth := calculateMaxDepth(n.Body, currentDepth)
		if bodyDepth > maxDepth {
			maxDepth = bodyDepth
		}

	case *ast.RangeStmt:
		bodyDepth := calculateMaxDepth(n.Body, currentDepth)
		if bodyDepth > maxDepth {
			maxDepth = bodyDepth
		}

	case *ast.SwitchStmt:
		bodyDepth := calculateMaxDepth(n.Body, currentDepth)
		if bodyDepth > maxDepth {
			maxDepth = bodyDepth
		}

	case *ast.TypeSwitchStmt:
		bodyDepth := calculateMaxDepth(n.Body, currentDepth)
		if bodyDepth > maxDepth {
			maxDepth = bodyDepth
		}

	case *ast.SelectStmt:
		bodyDepth := calculateMaxDepth(n.Body, currentDepth)
		if bodyDepth > maxDepth {
			maxDepth = bodyDepth
		}
	}

	return maxDepth
}

// checkIfElseChains detects if-else chains that should use early returns.
// Every else-if is an IfStmt of its own, so each chain is measured and
// reported once, at its head, and its else-ifs are skipped when the walk
// reaches them.
func checkIfElseChains(reporter *nolint.Reporter, body *ast.BlockStmt) {
	inChain := make(map[*ast.IfStmt]bool)

	ast.Inspect(body, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok || inChain[ifStmt] {
			return true
		}

		// Count else-if chain length; a final else block is a branch too
		chainLength := 1
		for current := ifStmt; current.Else != nil; {
			chainLength++
			elseIf, ok := current.Else.(*ast.IfStmt)
			if !ok {
				break
			}
			inChain[elseIf] = true
			current = elseIf
		}

		// Check if this could be converted to early returns
		if chainLength > MaxIfElseChain && couldUseEarlyReturn(ifStmt) {
			reporter.Reportf(ifStmt.Pos(),
				"if-else chain with %d branches; consider using early returns to flatten",
				chainLength)
		}

		return true
	})
}

// couldUseEarlyReturn checks if an if statement could be refactored to early return
func couldUseEarlyReturn(ifStmt *ast.IfStmt) bool {
	// If the if body ends with a return, this could be an early return pattern
	if len(ifStmt.Body.List) > 0 {
		lastStmt := ifStmt.Body.List[len(ifStmt.Body.List)-1]
		if _, ok := lastStmt.(*ast.ReturnStmt); ok {
			return true
		}
	}
	return false
}

// checkNestedIfs detects nested if statements that could be combined
func checkNestedIfs(reporter *nolint.Reporter, body *ast.BlockStmt) {
	ast.Inspect(body, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		// Check if the only statement in the if body is another if
		if len(ifStmt.Body.List) == 1 {
			if innerIf, ok := ifStmt.Body.List[0].(*ast.IfStmt); ok {
				// Skip if either if has an Init clause (short variable declaration)
				// These cannot be easily combined with && because the variable
				// needs to be assigned before checking the condition
				// Example: if x := getValue(); x != nil { if err := x.Do(); err != nil { ... } }
				if innerIf.Init != nil {
					return true
				}

				// Nested if that could potentially be combined with &&
				if ifStmt.Else == nil && innerIf.Else == nil {
					reporter.Reportf(innerIf.Pos(),
						"nested if statements could be combined with && operator")
				}
			}
		}

		return true
	})
}

// checkFunctionLength checks if a function is too long and should be split
func checkFunctionLength(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	// Count if err != nil patterns
	errCheckCount := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok && isErrCheck(ifStmt) {
			errCheckCount++
		}
		return true
	})

	// If function has many error checks, suggest splitting
	if errCheckCount > 5 {
		reporter.Reportf(fn.Pos(),
			"function %q has %d error checks; consider extracting helper functions",
			fn.Name.Name, errCheckCount)
	}
}

// isErrCheck checks if an if statement is checking for err != nil
func isErrCheck(ifStmt *ast.IfStmt) bool {
	binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return false
	}

	// Check for err != nil
	if ident, ok := binExpr.X.(*ast.Ident); ok && ident.Name == "err" {
		return true
	}

	if ident, ok := binExpr.Y.(*ast.Ident); ok && ident.Name == "err" {
		return true
	}

	return false
}
