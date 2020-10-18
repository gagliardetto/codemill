package main

import "github.com/gagliardetto/codemill/cmd/link/internal/ld/testdata/issue25459/a"

var Glob int

func main() {
	a.Another()
	Glob += a.ConstIf() + a.CallConstIf()
}
