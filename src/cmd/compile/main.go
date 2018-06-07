package main

import (
	"flag"

	"gitee.com/yuyang-fine/lucy/src/cmd/compile/common"
	"gitee.com/yuyang-fine/lucy/src/cmd/compile/lc"
)

func main() {
	flag.BoolVar(&common.CompileFlags.OnlyImport, "only-import", false, "only parse import package")
	flag.BoolVar(&common.CompileFlags.DisableCheckUnUsedVariable, "disable-check-unused-variable", false, "disable check unused variable")
	flag.StringVar(&common.CompileFlags.PackageName, "package-name", "", "package name")
	flag.IntVar(&common.CompileFlags.JvmVersion, "jvm-version", 54, "jvm major verion")
	flag.Parse()
	lc.Main(flag.Args())
}
