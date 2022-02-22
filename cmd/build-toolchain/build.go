package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/anupcshan/gotool"
)

var archs = []string{"amd64", "arm64", "arm"}

func main() {
	log.SetFlags(log.Lmicroseconds)

	toolchainURL := fmt.Sprintf("https://go.dev/dl/go%s.src.tar.gz", gotool.GoVersion)
	log.Printf("Building Go version %s", gotool.GoVersion)

	f, err := ioutil.TempFile("", "toolchain")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Get(toolchainURL)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	log.Println("Done fetching")

	for _, arch := range archs {
		log.Printf("Building for arch %s", arch)
		if err := os.RemoveAll("/src"); err != nil {
			log.Fatal(err)
		}
		if err := os.MkdirAll("/src", 0755); err != nil {
			log.Fatal(err)
		}
		if err := exec.Command("tar", "-C", "/src", "-xzf", f.Name()).Run(); err != nil {
			log.Fatal(err)
		}

		log.Println("Done extracting")

		build := exec.Command("./make.bash")
		build.Dir = "/src/go/src"
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		build.Env = append(
			os.Environ(),
			"CGO_ENABLED=0",
			"GOARCH="+arch,
			"SOURCE_DATE_EPOCH=1600000000",
		)
		if arch == "arm" {
			build.Env = append(build.Env, "GOARM=7")
		}
		if err := build.Run(); err != nil {
			log.Fatal(err)
		}

		if arch != "amd64" {
			if err := os.RemoveAll("/src/go/pkg/linux_amd64"); err != nil {
				log.Fatal(err)
			}
			if err := os.RemoveAll("/src/go/pkg/tool/linux_amd64"); err != nil {
				log.Fatal(err)
			}

			for _, bin := range []string{"go", "gofmt"} {
				if err := os.Rename(
					filepath.Join("/src/go/bin", "linux_"+arch, bin),
					filepath.Join("/src/go/bin/", bin),
				); err != nil {
					log.Fatal(err)
				}
			}
		}

		mksquashfs := exec.Command(
			"mksquashfs",
			"/src/go",
			fmt.Sprintf("/tmp/buildresult/gotool.%s.sqfs", arch),
			"-noappend",
		)
		mksquashfs.Stdout = os.Stdout
		mksquashfs.Stderr = os.Stderr
		mksquashfs.Env = append(os.Environ(), "SOURCE_DATE_EPOCH=1600000000")
		if err := mksquashfs.Run(); err != nil {
			log.Fatal(err)
		}
	}
}
