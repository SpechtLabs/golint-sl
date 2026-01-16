// Package varscope provides an analyzer that detects variables with scope that is too broad.
//
// Variables should be declared as close to their usage as possible.
// Broad scope makes code harder to follow and increases risk of unintended modifications.
package varscope

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `detect variables declared too far from their first use

Variables should be declared as close to their usage as possible.
Broad scope makes code harder to read and increases risk of bugs.

Good pattern:
    func process() error {
        // x declared right before use
        x := computeX()
        return doSomething(x)
    }

Bad pattern:
    func process() error {
        x := computeX()  // declared here
        
        // ... 20 lines of unrelated code ...
        
        return doSomething(x)  // used here
    }

This analyzer flags:
1. Variables declared more than 10 lines before first use
2. Variables modified multiple times (prefer immutability)
3. Variables declared at function start but only used in one branch`

var Analyzer = &analysis.Analyzer{
	Name:     "varscope",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// MaxLinesBetweenDeclAndUse is the maximum allowed lines between declaration and first use
const MaxLinesBetweenDeclAndUse = 10

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		checkFunction(reporter, pass, fn)
	})

	return nil, nil
}

type varDecl struct {
	name     string
	declLine int
	declPos  token.Pos
	uses     []int // line numbers of uses
	assigns  int   // number of assignments (including initial)
}

func checkFunction(reporter *nolint.Reporter, pass *analysis.Pass, fn *ast.FuncDecl) {
	vars := make(map[string]*varDecl)

	// First pass: collect declarations
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE { // :=
				for _, lhs := range node.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						line := pass.Fset.Position(ident.Pos()).Line
						vars[ident.Name] = &varDecl{
							name:     ident.Name,
							declLine: line,
							declPos:  ident.Pos(),
							assigns:  1,
						}
					}
				}
			} else if node.Tok == token.ASSIGN { // =
				for _, lhs := range node.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						if v, exists := vars[ident.Name]; exists {
							v.assigns++
						}
					}
				}
			}

		case *ast.ValueSpec:
			for _, name := range node.Names {
				line := pass.Fset.Position(name.Pos()).Line
				vars[name.Name] = &varDecl{
					name:     name.Name,
					declLine: line,
					declPos:  name.Pos(),
					assigns:  1,
				}
			}
		}

		return true
	})

	// Second pass: collect usages
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		if v, exists := vars[ident.Name]; exists {
			line := pass.Fset.Position(ident.Pos()).Line
			// Don't count the declaration line as a use
			if line != v.declLine {
				v.uses = append(v.uses, line)
			}
		}

		return true
	})

	// Check for issues
	for _, v := range vars {
		// Skip common short-lived variables
		if isCommonLoopVar(v.name) {
			continue
		}

		// Skip if no uses (dead code - other linters catch this)
		if len(v.uses) == 0 {
			continue
		}

		// Check distance from declaration to first use
		firstUse := v.uses[0]
		for _, use := range v.uses {
			if use < firstUse {
				firstUse = use
			}
		}

		distance := firstUse - v.declLine
		if distance > MaxLinesBetweenDeclAndUse {
			reporter.Reportf(v.declPos,
				"variable %q declared %d lines before first use; declare variables closer to their usage",
				v.name, distance)
		}

		// Check for excessive mutations
		if v.assigns > 3 {
			reporter.Reportf(v.declPos,
				"variable %q is assigned %d times; consider using immutable values or breaking into smaller functions",
				v.name, v.assigns)
		}
	}

	// Check for variables only used in one branch
	checkBranchOnlyVars(pass, fn)
}

func isCommonLoopVar(name string) bool {
	common := map[string]bool{
		"i": true, "j": true, "k": true,
		"v": true, "ok": true, "err": true,
		"_": true, "ctx": true,
	}
	return common[name]
}

func checkBranchOnlyVars(pass *analysis.Pass, fn *ast.FuncDecl) {
	// Find variables declared at function level but only used in one if branch
	for _, stmt := range fn.Body.List {
		// Check for pattern: var x = ...; if cond { use(x) }
		if assignStmt, ok := stmt.(*ast.AssignStmt); ok {
			if assignStmt.Tok != token.DEFINE {
				continue
			}

			// Check if next statement is an if that uses this variable
			idx := stmtIndex(fn.Body.List, stmt)
			if idx < 0 || idx+1 >= len(fn.Body.List) {
				continue
			}

			nextStmt := fn.Body.List[idx+1]
			ifStmt, ok := nextStmt.(*ast.IfStmt)
			if !ok {
				continue
			}

			// Check if variable is only used in this if statement
			for _, lhs := range assignStmt.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok {
					continue
				}

				usedInIf := usesVar(ifStmt, ident.Name)
				usedElsewhere := false

				// Check rest of function
				for i := idx + 2; i < len(fn.Body.List); i++ {
					if usesVar(fn.Body.List[i], ident.Name) {
						usedElsewhere = true
						break
					}
				}

				if usedInIf && !usedElsewhere && ifStmt.Else == nil {
					reporter.Reportf(ident.Pos(),
						"variable %q is only used inside the following if block; consider declaring it inside the if",
						ident.Name)
				}
			}
		}
	}
}

func stmtIndex(stmts []ast.Stmt, target ast.Stmt) int {
	for i, s := range stmts {
		if s == target {
			return i
		}
	}
	return -1
}

func usesVar(node ast.Node, name string) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && ident.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}
