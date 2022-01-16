package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/anupcshan/gotool"
)

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	goroot, err := gotool.InstallGo()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("GOROOT:", goroot)
}
