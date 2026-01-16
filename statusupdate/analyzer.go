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
)

const Doc = `ensure reconcilers update Status after changes

This analyzer detects reconcilers that:
1. Modify spec or status fields but don't call Status().Update()
2. Create/Update resources but don't reflect state in Status
3. Handle errors without updating Status.Conditions

Kubernetes best practice is to always update Status to reflect current state,
including error conditions. This allows users and other controllers to observe
the actual state of resources.`

var Analyzer = &analysis.Analyzer{
	Name:     "statusupdate",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		if !isReconcileFunction(fn) {
			return
		}

		checkReconcilerStatus(pass, fn)
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

func checkReconcilerStatus(pass *analysis.Pass, fn *ast.FuncDecl) {
	hasResourceMutation := false
	hasStatusUpdate := false
	hasConditionUpdate := false

	// Track what operations are performed
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		methodName := sel.Sel.Name

		// Check for resource mutations
		mutationMethods := []string{"Create", "Update", "Patch", "Delete"}
		for _, method := range mutationMethods {
			if methodName == method {
				hasResourceMutation = true
			}
		}

		// Check for Status() calls
		if methodName == "Status" {
			hasStatusUpdate = true
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
				hasConditionUpdate = true
			}
		}

		// Check for meta.SetStatusCondition
		if ident, ok := sel.X.(*ast.Ident); ok {
			if ident.Name == "meta" && strings.Contains(methodName, "Condition") {
				hasConditionUpdate = true
			}
		}

		return true
	})

	// Also check for direct Status field assignments
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for _, lhs := range assign.Lhs {
			sel, ok := lhs.(*ast.SelectorExpr)
			if !ok {
				continue
			}

			// Check for .Status. assignments
			if isStatusFieldAccess(sel) {
				hasStatusUpdate = true
			}

			// Check for .Conditions assignments
			if sel.Sel.Name == "Conditions" {
				hasConditionUpdate = true
			}
		}

		return true
	})

	// Report issues
	if hasResourceMutation && !hasStatusUpdate {
		pass.Reportf(fn.Pos(),
			"reconciler mutates resources but doesn't update Status; use Status().Update() to reflect current state")
	}

	// Only warn about missing conditions if there's complex logic
	if hasResourceMutation && !hasConditionUpdate && hasComplexLogic(fn) {
		pass.Reportf(fn.Pos(),
			"reconciler performs mutations but doesn't update Status.Conditions; consider using conditions to report state")
	}
}

func isStatusFieldAccess(sel *ast.SelectorExpr) bool {
	// Check for patterns like obj.Status.Field
	if innerSel, ok := sel.X.(*ast.SelectorExpr); ok {
		if innerSel.Sel.Name == "Status" {
			return true
		}
	}

	// Direct .Status assignment
	if sel.Sel.Name == "Status" {
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
