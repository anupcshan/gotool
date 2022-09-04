package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

func main() {
	var goVersion string
	flag.StringVar(&goVersion, "go-version", "", "Go version to rebuild release for (empty to automatically select latest release)")
	flag.Parse()

	if goVersion == "" {
		log.Fatal("Go version not specified")
	}

	log.SetFlags(log.Lmicroseconds)

	toolchainURL := fmt.Sprintf("https://go.dev/dl/go%s.src.tar.gz", goVersion)
	log.Printf("Building Go version %s", goVersion)

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

	g, ctx := errgroup.WithContext(context.Background())
	for _, arch := range flag.Args() {
		arch := arch
		g.Go(func() error {
			archSrc := "/src-" + arch
			log.Printf("Building for arch %s", arch)
			if err := os.RemoveAll(archSrc); err != nil {
				return err
			}
			if err := os.MkdirAll(archSrc, 0755); err != nil {
				return err
			}
			if err := exec.CommandContext(ctx, "tar", "-C", archSrc, "-xzf", f.Name()).Run(); err != nil {
				return err
			}

			log.Println("Done extracting")

			build := exec.CommandContext(ctx, "./make.bash")
			build.Dir = filepath.Join(archSrc, "go/src")
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
				return err
			}

			if arch != "amd64" {
				if err := os.RemoveAll(filepath.Join(archSrc, "go/pkg/linux_amd64")); err != nil {
					return err
				}
				if err := os.RemoveAll(filepath.Join(archSrc, "go/pkg/tool/linux_amd64")); err != nil {
					return err
				}

				for _, bin := range []string{"go", "gofmt"} {
					if err := os.Rename(
						filepath.Join(archSrc, "go/bin", "linux_"+arch, bin),
						filepath.Join(archSrc, "go/bin", bin),
					); err != nil {
						return err
					}
				}
			}

			mksquashfs := exec.CommandContext(
				ctx,
				"mksquashfs",
				filepath.Join(archSrc, "go"),
				fmt.Sprintf("/tmp/buildresult/gotool.%s.sqfs", arch),
				"-noappend",
			)
			mksquashfs.Stdout = os.Stdout
			mksquashfs.Stderr = os.Stderr
			mksquashfs.Env = append(os.Environ(), "SOURCE_DATE_EPOCH=1600000000")
			if err := mksquashfs.Run(); err != nil {
				return err
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
