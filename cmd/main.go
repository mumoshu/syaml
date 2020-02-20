package main

import (
	"github.com/mumoshu/syaml"
	"io/ioutil"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 3 {
		op := syaml.Set(syaml.Keys(args[1]), args[2])

		if err := syaml.FileApply(args[0], op); err != nil {
			syaml.Errorf("err: %v", err)
			os.Exit(1)
		}
	} else if len(args) == 2 {
		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}

		op := syaml.Set(syaml.Keys(args[0]), args[1])

		if err := syaml.BytesApply(bytes, op); err != nil {
			syaml.Errorf("err: %v", err)
			os.Exit(1)
		}
	} else {
		syaml.Errorf("invalid number of args: you should pass 2 or 3 args, but got: %v\n", args)
		os.Exit(1)
	}
}

