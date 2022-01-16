package gotool

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"gopkg.in/freddierice/go-losetup.v1"
)

var (
	sqfsRootTemplate = flag.String("gotool.sqfsroot_template", "http://localhost:8000/gotool.amd64.sqfs", "SQFS root path")
)

func ensureSqfsOnDisk() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sqfsPath := filepath.Join(homeDir, fmt.Sprintf("%s.sqfs", GoVersion))
	if _, err := os.Stat(sqfsPath); err == nil {
		return sqfsPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	log.Println("Writing sqfs image to", sqfsPath)

	f, err := os.CreateTemp(homeDir, "sqfs")
	if err != nil {
		return "", err
	}

	resp, err := http.Get(*sqfsRootTemplate)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	hasher := sha256.New()

	if _, err := io.Copy(io.MultiWriter(f, hasher), resp.Body); err != nil {
		return "", err
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != checksums[runtime.GOARCH] {
		return "", fmt.Errorf("Checksum mismatch, actual %q, expected %q", actualChecksum, checksums[runtime.GOARCH])
	}

	if err := os.Rename(f.Name(), sqfsPath); err != nil {
		return "", err
	}

	return sqfsPath, nil
}

func ensureSqfsMounted(sqfsPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sqMountDir := filepath.Join(homeDir, "sqmount")
	if err := os.MkdirAll(sqMountDir, 0755); err != nil {
		return "", err
	}

	b, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}
		mountpoint := parts[4]
		if mountpoint == sqMountDir {
			// Already mounted.
			// TODO: Check go version?
			return sqMountDir, nil
		}
	}

	dev, err := losetup.Attach(sqfsPath, 0, true)
	if err != nil {
		return "", err
	}

	log.Println("Mounting sqfs image at", sqMountDir)

	return sqMountDir, syscall.Mount(dev.Path(), sqMountDir, "squashfs", syscall.MS_RDONLY, "")
}

func InstallGo() (string, error) {
	// Make sure sqfs is on disk
	sqfsPath, err := ensureSqfsOnDisk()
	if err != nil {
		return "", err
	}

	// Ensure sqfs is mounted
	return ensureSqfsMounted(sqfsPath)
}
