package main

import (
	"flag"
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func pass(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return true
			}

			callExpr, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := callExpr.Fun.(*ast.SelectorExpr)

			if !ok {
				return true
			}

			pkg := pass.TypesInfo.ObjectOf(sel.Sel).Pkg()

			// pkg is nil for method calls on local variables
			if pkg == nil || pkg.Path() != "errors" || sel.Sel.String() != "As" {
				return true
			}

			// we only want expressions like errors.As(err, &target) or errors.As
			switch x := callExpr.Args[1].(type) {
			// errors.As(err, target)
			case *ast.Ident:
				// allow error inits like `err := &CustomError{}`
				if errorDeclaration, ok := x.Obj.Decl.(*ast.AssignStmt); ok {
					if !ok {
						return true
					}
					switch declType := errorDeclaration.Rhs[0].(type) {
					case *ast.UnaryExpr:
						if declType.Op.String() == "&" {
							return true
						}
					}
				}

				pass.Report(analysis.Diagnostic{
					Pos:     callExpr.Args[1].Pos(),
					Message: fmt.Sprintf("this call to errors.As will panic. Consider prefixing %s with &", x.Name),
				})
			}

			return true
		})
	}

	return nil, nil
}

func main() {
	analyzer := analysis.Analyzer{
		Name:             "faulty_errorsas",
		Doc:              "faulty_errorsas\n\nThis linter detects potential panics when using errors.As.",
		Flags:            flag.FlagSet{},
		Run:              pass,
		RunDespiteErrors: false,
	}

	singlechecker.Main(&analyzer)
}
