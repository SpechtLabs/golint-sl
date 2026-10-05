// Package statusupdate provides an analyzer that ensures Kubernetes reconcilers
// properly update the Status subresource after making changes.
package statusupdate

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
const Doc = `ensure reconcilers update Status after changes

This analyzer detects reconcilers that:
1. Modify spec or status fields but don't call Status().Update()
2. Create/Update resources but don't reflect state in Status
3. Handle errors without updating Status.Conditions

Kubernetes best practice is to always update Status to reflect current state,
including error conditions. This allows users and other controllers to observe
the actual state of resources.`

// Analyzer reports reconcilers that change resources without updating their Status.
var Analyzer = &analysis.Analyzer{
	Name:     "statusupdate",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// statusField is the name of the Status subresource field and accessor.
const statusField = "Status"

// statusUsage records which mutation and status operations a reconciler performs
type statusUsage struct {
	resourceMutation bool
	statusUpdate     bool
	conditionUpdate  bool
}

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

		if !isReconcileFunction(fn) {
			return
		}

		checkReconcilerStatus(reporter, fn)
	})

	return nil, nil
}

func isReconcileFunction(fn *ast.FuncDecl) bool {
	if fn.Name == nil || fn.Name.Name != "Reconcile" {
		return false
	}

	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}

	recv := fn.Recv.List[0]
	recvType := types.ExprString(recv.Type)

	patterns := []string{"Reconciler", "Controller", "Operator"}
	for _, pattern := range patterns {
		if strings.Contains(recvType, pattern) {
			return true
		}
	}

	return false
}

func checkReconcilerStatus(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	var usage statusUsage

	// Track what operations are performed
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			usage.recordCall(call)
		}
		return true
	})

	// Also check for direct Status field assignments
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok {
			usage.recordAssignment(assign)
		}
		return true
	})

	// Report issues
	if usage.resourceMutation && !usage.statusUpdate {
		reporter.Reportf(fn.Pos(),
			"reconciler mutates resources but doesn't update Status; use Status().Update() to reflect current state")
	}

	// Only warn about missing conditions if there's complex logic
	if usage.resourceMutation && !usage.conditionUpdate && hasComplexLogic(fn) {
		reporter.Reportf(fn.Pos(),
			"reconciler performs mutations but doesn't update Status.Conditions; consider using conditions to report state")
	}
}

// recordCall records the mutation, Status and condition operations performed by a call
func (u *statusUsage) recordCall(call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	methodName := sel.Sel.Name

	// Check for resource mutations
	mutationMethods := []string{"Create", "Update", "Patch", "Delete"}
	for _, method := range mutationMethods {
		if methodName == method {
			u.resourceMutation = true
		}
	}

	// Check for Status() calls
	if methodName == statusField {
		u.statusUpdate = true
	}

	// Check for condition updates (various patterns)
	conditionPatterns := []string{
		"SetCondition",
		"SetConditions",
		"UpdateCondition",
		"SetStatusCondition",
		"SetTypedCondition",
		"SetReadyCondition",
	}
	for _, pattern := range conditionPatterns {
		if strings.Contains(methodName, pattern) || methodName == pattern {
			u.conditionUpdate = true
		}
	}

	// Check for meta.SetStatusCondition
	if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "meta" && strings.Contains(methodName, "Condition") {
		u.conditionUpdate = true
	}
}

// recordAssignment records direct assignments to Status and Conditions fields
func (u *statusUsage) recordAssignment(assign *ast.AssignStmt) {
	for _, lhs := range assign.Lhs {
		sel, ok := lhs.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		// Check for .Status. assignments
		if isStatusFieldAccess(sel) {
			u.statusUpdate = true
		}

		// Check for .Conditions assignments
		if sel.Sel.Name == "Conditions" {
			u.conditionUpdate = true
		}
	}
}

func isStatusFieldAccess(sel *ast.SelectorExpr) bool {
	// Check for patterns like obj.Status.Field
	if innerSel, ok := sel.X.(*ast.SelectorExpr); ok && innerSel.Sel.Name == statusField {
		return true
	}

	// Direct .Status assignment
	if sel.Sel.Name == statusField {
		return true
	}

	return false
}

func hasComplexLogic(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}

	// Count complexity indicators
	complexity := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.SelectStmt:
			complexity++
		}
		return complexity < 5
	})

	return complexity >= 3
}
