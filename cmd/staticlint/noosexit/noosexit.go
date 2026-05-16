// Package noosexit defines an analyzer that reports direct calls to os.Exit
// from the main function of package main.
//
// The analyzer is intended to keep shutdown logic testable and to prevent
// abrupt termination from the program entrypoint.
package noosexit

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "reports os.Exit calls",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {

	if !strings.HasPrefix(pass.Pkg.Path(), "github.com/Nakohartum/practicum-metrics") {
		return nil, nil
	}
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if ok && fn.Name.Name == "main" {
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "Exit" {
						return true
					}

					ident, ok := sel.X.(*ast.Ident)

					if !ok {
						return true
					}

					obj := pass.TypesInfo.Uses[ident]

					pkgName, ok := obj.(*types.PkgName)
					if !ok || pkgName.Imported().Path() != "os" {
						return true
					}

					pass.Reportf(call.Pos(), "direct call to os.Exit is not allowed in main.main")

					return true
				})
				return false
			}
			return true
		})
	}
	return nil, nil
}
