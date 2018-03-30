package main

import (
	"flag"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/lc"
)

func main() {
	flag.BoolVar(&lc.CompileFlags.OnlyImport, "only-import", false, "only parse import package")
	flag.StringVar(&lc.CompileFlags.PackageName, "package-name", "", "package name")
	flag.Parse()
	lc.Main(flag.Args())
}
