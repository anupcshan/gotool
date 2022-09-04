package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anupcshan/gotool"
	"github.com/google/go-github/v47/github"
)

const dockerfileContents = `
FROM golang:latest

RUN apt-get update && apt-get install -y squashfs-tools

COPY build-toolchain /usr/bin/build-toolchain
`

const (
	githubUser     = "gotool-bot"
	githubRepoUser = "anupcshan"
)

var (
	archs     = []string{"amd64", "arm64", "arm"}
	goVersion string
)

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

	args := []string{
		"run",
		"--rm",
		"--volume", fmt.Sprintf("%s:/tmp/buildresult:Z", builddir),
		"gotool-rebuild-toolchain",
		"/usr/bin/build-toolchain",
		"--go-version",
		goVersion,
	}

	args = append(args, archs...)

	dockerRun := exec.Command("docker", args...)
	dockerRun.Dir = builddir
	dockerRun.Stderr = os.Stderr
	dockerRun.Stdout = os.Stdout
	if err := dockerRun.Run(); err != nil {
		return fmt.Errorf("error running build-toolchain: %w", err)
	}

	return nil
}

func upload(ctx context.Context, builddir string, client *github.Client) error {
	var buf = new(bytes.Buffer)
	buf.WriteString(`package gotool

var checksums = map[string]string{
`)
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
		fmt.Fprintf(buf, `"%s": "%s",`+"\n", arch, hex.EncodeToString(h.Sum(nil)))
	}

	buf.WriteString(`}`)

	fmted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	log.Printf("%s", fmted)

	log.Println("Creating release")
	release, err := createRelease(ctx, client)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Release: %+v", release)

	if err := uploadAssets(ctx, client, release, builddir); err != nil {
		log.Fatal(err)
	}

	return closeRelease(ctx, client, release)

	// Create a commit with checksums.go changed with a new tag (release version).
	// Merge commit with master.
}

func closeRelease(ctx context.Context, client *github.Client, release *github.RepositoryRelease) error {
	release, _, err := client.Repositories.EditRelease(ctx, githubRepoUser, "gotool", *release.ID, &github.RepositoryRelease{
		Draft:   github.Bool(false),
		TagName: github.String(goVersion),
		Name:    github.String(fmt.Sprintf("Go %s static toolchain", goVersion)),
	})

	log.Printf("%+v", release)

	return err
}

func uploadAssets(ctx context.Context, client *github.Client, release *github.RepositoryRelease, builddir string) error {
	for _, arch := range archs {
		sqfsPath := filepath.Join(builddir, fmt.Sprintf("gotool.%s.sqfs", arch))
		f, err := os.Open(sqfsPath)
		if err != nil {
			return err
		}

		releaseAsset, _, err := client.Repositories.UploadReleaseAsset(ctx, githubRepoUser, "gotool", *release.ID, &github.UploadOptions{
			Name: fmt.Sprintf("gotool.%s.sqfs", arch),
		}, f)
		log.Printf("Asset: %+v", releaseAsset)
		if err != nil {
			return err
		}
	}

	return nil
}

func createRelease(ctx context.Context, client *github.Client) (*github.RepositoryRelease, error) {
	release, _, err := client.Repositories.CreateRelease(ctx, githubRepoUser, "gotool", &github.RepositoryRelease{
		Draft:   github.Bool(true),
		TagName: github.String(goVersion),
		Name:    github.String(fmt.Sprintf("Go %s static toolchain", goVersion)),
	})

	return release, err
}

// https://pkg.go.dev/golang.org/x/website/internal/dl#Release
// Releases sorted from newest to oldest.
func getLatestGoRelease() (string, error) {
	resp, err := http.Get("https://go.dev/dl/?mode=json")
	if err != nil {
		return "", err
	}

	var releases []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}

	log.Printf("%+v", releases)

	for _, release := range releases {
		if release.Stable {
			return strings.TrimPrefix(release.Version, "go"), nil
		}
	}

	return "", fmt.Errorf("No stable releases found")
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)

	flag.StringVar(&goVersion, "go-version", "", "Go version to rebuild release for (empty to automatically select latest release)")
	flag.Parse()

	if goVersion == "" {
		var err error
		goVersion, err = getLatestGoRelease()
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Detected latest Go version %s", goVersion)
	}

	if goVersion == gotool.GoVersion {
		log.Printf("No change in Go version. Not building a new release ...")
		return
	}

	tmp, err := ioutil.TempDir("/tmp", "rebuild-toolchain")
	if err != nil {
		log.Fatal(err)
	}

	authToken := os.Getenv("GH_AUTH_TOKEN")
	if authToken == "" {
		log.Fatal("GH_AUTH_USER unset")
	}

	client := github.NewClient(&http.Client{
		Transport: &github.BasicAuthTransport{
			Username: githubUser,
			Password: authToken,
		},
	})

	ctx := context.Background()

	log.Printf("Building toolchain in %s", tmp)

	if err := rebuild(tmp); err != nil {
		_ = os.RemoveAll(tmp)
		log.Fatal(err)
	}

	if err := upload(ctx, tmp, client); err != nil {
		_ = os.RemoveAll(tmp)
		log.Fatal(err)
	}
}
