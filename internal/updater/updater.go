package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner       = "zhenchaochen"
	repoName        = "kronos"
	releaseURL      = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
	downloadBase    = "https://github.com/" + repoOwner + "/" + repoName + "/releases/download"
	binaryName      = "kronos"
	httpTimeout     = 30 * time.Second
	downloadTimeout = 5 * time.Minute
	maxDownloadSize = 100 * 1024 * 1024 // 100 MB
)

type release struct {
	TagName string `json:"tag_name"`
}

// Update checks GitHub for a newer release and replaces the current binary if found.
func Update(currentVersion string) error {
	rel, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("checking latest release: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if current == "dev" || current == "" {
		return fmt.Errorf("cannot update a development build (version=%q)", currentVersion)
	}

	if !isNewer(current, latest) {
		fmt.Printf("Already up to date (v%s).\n", current)
		return nil
	}

	fmt.Printf("Updating v%s → v%s ...\n", current, latest)

	assetName := buildAssetName(latest)
	assetURL := fmt.Sprintf("%s/%s/%s", downloadBase, rel.TagName, assetName)

	archivePath, err := downloadFile(assetURL)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer os.Remove(archivePath)

	binPath, err := extractBinary(archivePath, assetName)
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}
	defer os.Remove(binPath)

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current binary: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	if err := replaceBinary(execPath, binPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	fmt.Printf("Updated to v%s successfully.\n", latest)
	return nil
}

func fetchLatestRelease() (*release, error) {
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(releaseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// isNewer returns true if latest > current using simple semver comparison.
func isNewer(current, latest string) bool {
	cp := parseSemver(current)
	lp := parseSemver(latest)
	for i := 0; i < 3; i++ {
		if lp[i] > cp[i] {
			return true
		}
		if lp[i] < cp[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	var parts [3]int
	fmt.Sscanf(v, "%d.%d.%d", &parts[0], &parts[1], &parts[2])
	return parts
}

func buildAssetName(version string) string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}

	// GoReleaser default naming: {project}_{Version}_{Os}_{Arch}{ext}
	return fmt.Sprintf("kronos_%s_%s_%s%s", version, capitalize(runtime.GOOS), runtime.GOARCH, ext)
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func downloadFile(url string) (string, error) {
	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "kronos-update-*")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmp, io.LimitReader(resp.Body, maxDownloadSize)); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}

func extractBinary(archivePath, assetName string) (string, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(archivePath)
	}
	return extractFromTarGz(archivePath)
}

func extractFromTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}

		if filepath.Base(hdr.Name) == binaryName && hdr.Typeflag == tar.TypeReg {
			return writeTemp(tr)
		}
	}
	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}

func extractFromZip(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	target := binaryName + ".exe"
	for _, f := range r.File {
		if filepath.Base(f.Name) != target {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		return writeTemp(rc)
	}
	return "", fmt.Errorf("binary %q not found in archive", target)
}

func writeTemp(r io.Reader) (string, error) {
	tmp, err := os.CreateTemp("", "kronos-bin-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(tmp, io.LimitReader(r, maxDownloadSize)); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}

func replaceBinary(current, newBin string) error {
	old := current + ".old"

	if err := os.Rename(current, old); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	if err := copyFile(newBin, current); err != nil {
		// Attempt rollback
		os.Rename(old, current)
		return fmt.Errorf("installing new binary: %w", err)
	}

	os.Remove(old)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}

	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
