package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const dockerfileContents = `
FROM golang:latest

RUN apt-get update && apt-get install -y squashfs-tools

COPY build-toolchain /usr/bin/build-toolchain

ENTRYPOINT /usr/bin/build-toolchain
`

var archs = []string{"amd64", "arm64", "arm"}

func rebuild(builddir string) error {
	cmd := exec.Command(
		"go",
		"install",
		"github.com/anupcshan/gotool/cmd/build-toolchain",
	)
	cmd.Env = append(os.Environ(), "GOOS=linux", "CGO_ENABLED=0", fmt.Sprintf("GOBIN=%s", builddir))
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build build-toolchain: %w", err)
	}

	dockerBuild := exec.Command(
		"docker",
		"build",
		"--rm",
		"--pull",
		"--tag=gotool-rebuild-toolchain",
		"--file=-",
		".",
	)
	dockerBuild.Dir = builddir
	dockerBuild.Stderr = os.Stderr
	dockerBuild.Stdout = os.Stdout
	dockerBuild.Stdin = bytes.NewReader([]byte(dockerfileContents))
	if err := dockerBuild.Run(); err != nil {
		return fmt.Errorf("error building docker container: %w", err)
	}

	dockerRun := exec.Command(
		"docker",
		"run",
		"--rm",
		"--volume", fmt.Sprintf("%s:/tmp/buildresult:Z", builddir),
		"gotool-rebuild-toolchain",
	)
	dockerRun.Dir = builddir
	dockerRun.Stderr = os.Stderr
	dockerRun.Stdout = os.Stdout
	if err := dockerRun.Run(); err != nil {
		return fmt.Errorf("error running build-toolchain: %w", err)
	}

	return nil
}

func upload(builddir string) error {
	for _, arch := range archs {
		sqfsPath := filepath.Join(builddir, fmt.Sprintf("gotool.%s.sqfs", arch))
		f, err := os.Open(sqfsPath)
		if err != nil {
			return err
		}
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return err
		}
		log.Printf("%s: %s", arch, hex.EncodeToString(h.Sum(nil)))
	}

	return nil
}

func main() {
	tmp, err := ioutil.TempDir("/tmp", "rebuild-toolchain")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Building toolchain in %s", tmp)

	if err := rebuild(tmp); err != nil {
		_ = os.RemoveAll(tmp)
		log.Fatal(err)
	}

	if err := upload(tmp); err != nil {
		_ = os.RemoveAll(tmp)
		log.Fatal(err)
	}
}
