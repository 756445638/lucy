package lc

import (
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/ast"
)

var (
	CompileFlags         Flags
	compiler             LucyCompile
	loader               RealNameLoader
	ParseFunctionHandler func(bs []byte, pos *ast.Pos) (*ast.Function, []error)
)

type Flags struct {
	OnlyImport  bool
	PackageName string
}

func init() {
	ast.NameLoader = &loader
	ParseFunctionHandler = ast.ParseFunctionHandler
}

const (
	mainClassName = "main.class"
)
