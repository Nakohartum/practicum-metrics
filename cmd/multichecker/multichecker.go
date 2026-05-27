// Package main provides a custom multichecker.
//
// Run:
//
//	go run ./cmd/multichecker ./...
//
// Included analyzers:
//   - printf: checks Printf-style calls
//   - shadow: reports shadowed variables
//   - structtag: validates struct tags
//   - stdmethods: checks signatures of standard methods
//   - SA*: all Staticcheck analyzers
//   - ST*: style-related analyzer from Staticcheck
//   - noosexit: forbids direct os.Exit calls in main.main
package main

import (
	"github.com/Nakohartum/practicum-metrics/cmd/staticlint/noosexit"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	var analyzers = []*analysis.Analyzer{
		printf.Analyzer,
		shadow.Analyzer,
		stdmethods.Analyzer,
		structtag.Analyzer,
		noosexit.Analyzer,
		nilness.Analyzer,
		unusedresult.Analyzer,
	}

	for _, a := range staticcheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	for _, a := range stylecheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	multichecker.Main(
		analyzers...,
	)
}
