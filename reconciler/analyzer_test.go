package reconciler_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/reconciler"
)

func TestReconcilerAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, reconciler.Analyzer, "a")
}

func TestAnalyzeReconciler(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want reconciler.ReconcilerInfo
	}{
		{
			name: "proper signature with RequeueAfter and IsNotFound",
			src: `func (r *R) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
				if apierrors.IsNotFound(err) {
					return ctrl.Result{}, nil
				}
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			}`,
			want: reconciler.ReconcilerInfo{Name: "Reconcile", HasProperSig: true, UsesRequeue: true, HasNotFoundCheck: true},
		},
		{
			name: "Requeue key counts as requeue",
			src:  `func (r *R) Reconcile() (ctrl.Result, error) { return ctrl.Result{Requeue: true}, nil }`,
			want: reconciler.ReconcilerInfo{Name: "Reconcile", HasProperSig: true, UsesRequeue: true},
		},
		{
			name: "other keys, positional and map elements are not requeue",
			src: `func (r *R) Reconcile() (ctrl.Result, error) {
				_ = pkg.Result{"k": 1}
				_ = pkg.Result{1, 2}
				_ = ctrl.Other{Requeue: true}
				_ = Result{Requeue: true}
				return ctrl.Result{Priority: 1}, nil
			}`,
			want: reconciler.ReconcilerInfo{Name: "Reconcile", HasProperSig: true},
		},
		{
			name: "non-selector calls and other selector calls",
			src: `func Reconcile() (a, b int) {
				f()
				x.IsFound(err)
				return 0, 0
			}`,
			want: reconciler.ReconcilerInfo{Name: "Reconcile"},
		},
		{
			name: "no results",
			src:  `func Reconcile() { _ = apierrors.IsNotFound(nil) }`,
			want: reconciler.ReconcilerInfo{Name: "Reconcile", HasNotFoundCheck: true},
		},
		{
			name: "no body",
			src:  `func Reconcile() (ctrl.Result, error)`,
			want: reconciler.ReconcilerInfo{Name: "Reconcile", HasProperSig: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "x.go", "package x\n"+tt.src, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fn, ok := file.Decls[0].(*ast.FuncDecl)
			if !ok {
				t.Fatalf("first declaration is %T, want *ast.FuncDecl", file.Decls[0])
			}

			got := reconciler.AnalyzeReconciler(fn)
			if got.Name != tt.want.Name || got.HasProperSig != tt.want.HasProperSig ||
				got.UsesRequeue != tt.want.UsesRequeue || got.HasNotFoundCheck != tt.want.HasNotFoundCheck ||
				len(got.ForbiddenCalls) != 0 {
				t.Errorf("AnalyzeReconciler() = %+v, want %+v", *got, tt.want)
			}
		})
	}
}
