package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/anupcshan/gotool"
)

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	if flag.NArg() == 0 {
		// Prevent Gokrazy from restarting this process
		os.Exit(125)
	}

	goroot, err := gotool.InstallGo()
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(filepath.Join(goroot, "bin/go"), flag.Args()...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
