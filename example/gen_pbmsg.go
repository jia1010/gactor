package main

import (
	"github.com/mafei198/goslib/pbmsg"
)

func main() {
	protoDir := "./protos"
	pkg := "protos"
	outfile := "./protos/register.go"
	pbmsg.Generate(protoDir, pkg, outfile)
}
