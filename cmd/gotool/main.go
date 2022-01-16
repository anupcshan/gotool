package main

import (
	"fmt"
	"log"

	"github.com/anupcshan/gotool"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	goroot, err := gotool.InstallGo()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("GOROOT:", goroot)
}
