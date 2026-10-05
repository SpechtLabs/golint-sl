// Package pkgnaming enforces Go package naming conventions.
//
// Good package names are short, clear, and lower case, with no underscores
// or mixedCaps. Package names should not stutter with their exported types.
package pkgnaming

import (
	"go/ast"
	"strings"
	"unicode"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
const Doc = `enforce Go package naming conventions

Package naming rules:
1. Names should be short, concise, and lowercase
2. No underscores or mixedCaps (use single words)
3. Avoid stutter: don't repeat package name in exported names
   - Bad: user.UserService, http.HTTPClient
   - Good: user.Service, http.Client
4. Use singular form: "user" not "users"
5. Avoid generic names: util, common, misc, helper

Reference: https://go.dev/blog/package-names`

// Analyzer reports package names that break Go naming conventions.
var Analyzer = &analysis.Analyzer{
	Name:     "pkgnaming",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Generic package names that should be avoided
var genericNames = map[string]bool{
	"util":    true,
	"utils":   true,
	"common":  true,
	"misc":    true,
	"helper":  true,
	"helpers": true,
	"base":    true,
	"core":    true,
	"shared":  true,
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	pkgName := pass.Pkg.Name()

	// Check package name issues
	checkPackageName(reporter, pass, pkgName)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.TypeSpec:
			// Callers only ever spell exported package-level types with the
			// package qualifier, so unexported and function-local types can't
			// stutter.
			if isPackageLevelExported(pass, node.Name) {
				checkStutter(reporter, pkgName, node.Name.Name, node, "type")
			}

		case *ast.FuncDecl:
			// Only check exported functions without receivers
			if node.Recv == nil && ast.IsExported(node.Name.Name) {
				checkStutter(reporter, pkgName, node.Name.Name, node, "function")
			}
		}
	})

	return nil, nil
}

func checkPackageName(reporter *nolint.Reporter, pass *analysis.Pass, name string) {
	// Every finding is reported on the first file
	if len(pass.Files) == 0 {
		return
	}

	// Check for generic names
	if genericNames[name] {
		reporter.Reportf(pass.Files[0].Package,
			"package name %q is too generic; use a more descriptive name that indicates what the package does",
			name)
	}

	// Check for underscores (but allow *_test packages - Go's external test convention)
	if strings.Contains(name, "_") && !strings.HasSuffix(name, "_test") {
		reporter.Reportf(pass.Files[0].Package,
			"package name %q contains underscore; use a single lowercase word",
			name)
	}

	// Check for mixed case
	hasUpper := false
	for _, r := range name {
		if unicode.IsUpper(r) {
			hasUpper = true
			break
		}
	}
	if hasUpper {
		reporter.Reportf(pass.Files[0].Package,
			"package name %q contains uppercase letters; package names should be lowercase",
			name)
	}

	// Check for plural (common mistake)
	pluralSuffixes := []string{"ers", "ors", "ies", "es", "s"}
	for _, suffix := range pluralSuffixes {
		if !strings.HasSuffix(name, suffix) || len(name) <= len(suffix)+2 {
			continue
		}

		// Avoid false positives for words that naturally end in s
		exceptions := map[string]bool{
			"status": true, "class": true, "address": true,
			"process": true, "access": true, "express": true,
			"progress": true, "analysis": true, "basis": true,
		}
		if !exceptions[name] {
			reporter.Reportf(pass.Files[0].Package,
				"package name %q appears to be plural; use singular form",
				name)
		}
		break
	}
}

func checkStutter(reporter *nolint.Reporter, pkgName, exportedName string, node ast.Node, kind string) {
	pkgLower := strings.ToLower(pkgName)
	nameLower := strings.ToLower(exportedName)

	// Check if the exported name starts with the package name
	if strings.HasPrefix(nameLower, pkgLower) {
		// Extract what comes after the package name prefix
		suffix := exportedName[len(pkgName):]
		if suffix != "" && unicode.IsUpper(rune(suffix[0])) {
			// Exception: Mock* types in mock packages are a common convention
			// The Mock prefix clearly indicates the type's purpose for testing
			if pkgLower == "mock" && strings.HasPrefix(exportedName, "Mock") {
				return
			}

			// This is stutter: http.HTTPClient, user.UserService
			reporter.Reportf(node.Pos(),
				"%s %s.%s stutters; consider renaming to %s.%s",
				kind, pkgName, exportedName, pkgName, suffix)
		}
	}
}

// isPackageLevelExported reports whether name declares an exported object in
// the package scope, the only kind callers refer to as pkg.Name.
func isPackageLevelExported(pass *analysis.Pass, name *ast.Ident) bool {
	if !name.IsExported() {
		return false
	}
	obj := pass.TypesInfo.Defs[name]
	return obj != nil && obj.Parent() == pass.Pkg.Scope()
}
