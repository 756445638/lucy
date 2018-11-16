package main

import (
	"fmt"
	"gitee.com/yuyang-fine/lucy/src/cmd/common"
	"gitee.com/yuyang-fine/lucy/src/cmd/lucy/install_lucy_array"
	"gitee.com/yuyang-fine/lucy/src/cmd/lucy/run"
	"os"
	"runtime"
)

func printUsage() {
	msg := `lucy is a new programing language build on jvm
	version                print version
	build                  build package and don't run
	install                install directory and it's sub directories 
	run                    run a lucy package
	clean                  clean compiled files
	test                   test a package`
	fmt.Println(msg)
}

func main() {
	if len(os.Args) == 1 {
		printUsage()
		os.Exit(0)
	}
	switch os.Args[1] {
	case "version":
		fmt.Printf("lucy-%v@(%s/%s)\n", common.VERSION, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	case "build":
		(&run.RunLucyPackage{}).RunCommand("run", append([]string{"-build"}, os.Args[2:]...))
	case "run":
		(&run.RunLucyPackage{}).RunCommand(os.Args[1], os.Args[2:])
	case "install":
		args := []string{"lucy/cmd/langtools/install"}
		args = append(args, os.Args[2:]...)
		(&run.RunLucyPackage{}).RunCommand("run", args)
	case "clean":
		args := []string{"lucy/cmd/langtools/clean"}
		args = append(args, os.Args[2:]...)
		(&run.RunLucyPackage{}).RunCommand("run", args)
	case "test":
		args := []string{"lucy/cmd/langtools/test"}
		args = append(args, os.Args[2:]...)
		(&run.RunLucyPackage{}).RunCommand("run", args)
	case "install_lucy_array":
		(&install_lucy_array.InstallLucyArray{}).RunCommand("install_lucy_array", nil)
	case "tool":
		args := []string{"lucy/cmd/langtools/tool"}
		args = append(args, os.Args[2:]...)
		(&run.RunLucyPackage{}).RunCommand("run", args)
	default:
		printUsage()
		os.Exit(1)
	}
}
