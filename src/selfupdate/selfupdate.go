package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	DefaultRepo = "getkonfi/konfi"
	BinaryName  = "konfi"
)

type Options struct {
	Repo           string
	Version        string
	CurrentVersion string
	CheckOnly      bool
	Out            io.Writer
	Client         *http.Client
}

type ManagedInstall struct {
	Manager string
	Command string
	Path    string
}

func (m *ManagedInstall) Error() string {
	if m.Command == "" {
		return fmt.Sprintf("%s is managed by %s; use that package manager to update", m.Path, m.Manager)
	}
	return fmt.Sprintf("%s is managed by %s; update with `%s`", m.Path, m.Manager, m.Command)
}

type releaseInfo struct {
	TagName string `json:"tag_name"`
}

// Run updates the current konfi executable from GitHub release artifacts.
func Run(ctx context.Context, opts Options) error {
	if opts.Repo == "" {
		opts.Repo = DefaultRepo
	}
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 60 * time.Second}
	}
	s := newStyler(opts.Out)

	exe, paths, err := executablePaths()
	if err != nil {
		return err
	}
	if managed, ok := DetectManagedInstall(paths); ok {
		return managed
	}

	target, err := resolveTarget(ctx, opts)
	if err != nil {
		return err
	}
	targetVersion := strings.TrimPrefix(target.TagName, "v")

	if !opts.versionRequested() {
		if cmp, ok := compareVersions(opts.CurrentVersion, targetVersion); ok {
			switch {
			case cmp == 0:
				fmt.Fprintln(opts.Out, s.success(fmt.Sprintf("✓ konfi is already up to date (%s)", displayVersion(opts.CurrentVersion))))
				return nil
			case cmp > 0:
				fmt.Fprintln(opts.Out, s.dim(fmt.Sprintf("konfi %s is newer than latest release %s", displayVersion(opts.CurrentVersion), targetVersion)))
				return nil
			}
		}
	}

	if opts.CheckOnly {
		if opts.versionRequested() {
			fmt.Fprintln(opts.Out, s.info(fmt.Sprintf("konfi can install %s", targetVersion)))
			return nil
		}
		fmt.Fprintln(opts.Out, s.info(fmt.Sprintf("konfi %s is available (current %s)", targetVersion, displayVersion(opts.CurrentVersion))))
		return nil
	}

	artifact, err := artifactName(BinaryName, target.TagName, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	tmpdir, err := os.MkdirTemp("", "konfi-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", opts.Repo, target.TagName)
	archivePath := filepath.Join(tmpdir, artifact)
	checksumPath := filepath.Join(tmpdir, "checksums.txt")

	if err := download(ctx, opts.Client, baseURL+"/"+artifact, archivePath, s, artifact); err != nil {
		return err
	}
	if err := download(ctx, opts.Client, baseURL+"/checksums.txt", checksumPath, s, ""); err != nil {
		return fmt.Errorf("download checksums.txt: %w", err)
	}
	if err := verifyArchiveChecksum(archivePath, checksumPath, artifact); err != nil {
		return err
	}

	newBinary := filepath.Join(tmpdir, BinaryName)
	if err := extractBinary(archivePath, BinaryName, newBinary); err != nil {
		return err
	}
	if err := replaceExecutable(newBinary, exe); err != nil {
		return err
	}

	fmt.Fprintln(opts.Out, s.success(fmt.Sprintf("✓ updated konfi to %s", targetVersion))+s.dim(" → "+exe))
	return nil
}

func (o Options) versionRequested() bool {
	return strings.TrimSpace(o.Version) != ""
}

func resolveTarget(ctx context.Context, opts Options) (*releaseInfo, error) {
	if opts.versionRequested() {
		return &releaseInfo{TagName: normalizeTag(opts.Version)}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", opts.Repo), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "konfi-self-update")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := opts.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("fetch latest release: no published release found for %s", opts.Repo)
		}
		return nil, fmt.Errorf("fetch latest release: github returned %s", resp.Status)
	}

	var rel releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode latest release: %w", err)
	}
	if rel.TagName == "" {
		return nil, errors.New("latest release has no tag")
	}
	return &rel, nil
}

// download streams url to dst. when label is non-empty it reports progress:
// a live bar on interactive terminals, otherwise a single line.
func download(ctx context.Context, client *http.Client, url, dst string, s styler, label string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "konfi-self-update")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", path.Base(url), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: got %s", path.Base(url), resp.Status)
	}

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer f.Close()

	dest := io.Writer(f)
	if label != "" {
		if s.interactive && resp.ContentLength > 0 {
			pw := newProgressWriter(s, label, resp.ContentLength)
			dest = io.MultiWriter(f, pw)
			defer pw.finish()
		} else {
			fmt.Fprintln(s.out, s.info("downloading "+label))
		}
	}

	if _, err := io.Copy(dest, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func verifyArchiveChecksum(archivePath, checksumPath, artifact string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("read checksums.txt: %w", err)
	}

	expected := ""
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if filepath.Base(name) == artifact {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksums.txt does not include %s", artifact)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", archivePath, err)
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return fmt.Errorf("hash %s: %w", archivePath, err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("checksum mismatch for %s", artifact)
	}
	return nil
}

func extractBinary(archivePath, binary, dst string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", archivePath, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg || path.Base(hdr.Name) != binary {
			continue
		}

		out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return fmt.Errorf("create extracted binary: %w", err)
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return fmt.Errorf("extract binary: %w", copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close extracted binary: %w", closeErr)
		}
		if err := os.Chmod(dst, 0o755); err != nil {
			return fmt.Errorf("chmod extracted binary: %w", err)
		}
		return nil
	}
	return fmt.Errorf("%s not found in %s", binary, filepath.Base(archivePath))
}

func replaceExecutable(src, dst string) error {
	info, err := os.Stat(dst)
	if err != nil {
		return fmt.Errorf("stat executable %s: %w", dst, err)
	}
	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o755
	}

	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".konfi-update-*")
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("cannot replace %s: permission denied; rerun `konfi update` with sufficient privileges", dst)
		}
		return fmt.Errorf("create replacement in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	in, err := os.Open(src)
	if err != nil {
		tmp.Close()
		return fmt.Errorf("open new binary: %w", err)
	}
	defer in.Close()

	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return fmt.Errorf("write replacement: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod replacement: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close replacement: %w", err)
	}

	if err := os.Rename(tmpName, dst); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("cannot replace %s: permission denied; rerun `konfi update` with sufficient privileges", dst)
		}
		return fmt.Errorf("replace %s: %w", dst, err)
	}
	return nil
}

func executablePaths() (string, []string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", nil, fmt.Errorf("resolve current executable: %w", err)
	}

	paths := []string{exe}
	if real, err := filepath.EvalSymlinks(exe); err == nil && real != exe {
		paths = append(paths, real)
		exe = real
	}
	return exe, paths, nil
}

func artifactName(binary, tag, goos, goarch string) (string, error) {
	switch goos {
	case "linux", "darwin":
	default:
		return "", fmt.Errorf("unsupported operating system: %s", goos)
	}

	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("unsupported architecture: %s", goarch)
	}

	version := strings.TrimPrefix(normalizeTag(tag), "v")
	return fmt.Sprintf("%s_%s_%s_%s.tar.gz", binary, version, goos, goarch), nil
}

func normalizeTag(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

func compareVersions(current, target string) (int, bool) {
	current = normalizeSemver(current)
	target = normalizeSemver(target)
	if current == "" || target == "" {
		return 0, false
	}
	return semver.Compare(current, target), true
}

func normalizeSemver(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	v = "v" + v
	if !semver.IsValid(v) {
		return ""
	}
	return v
}

func displayVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	return strings.TrimPrefix(v, "v")
}

// DetectManagedInstall returns a package manager match for paths that should
// not be modified by konfi itself.
func DetectManagedInstall(paths []string) (*ManagedInstall, bool) {
	for _, p := range paths {
		if managed, ok := managedInstallByPath(p); ok {
			return managed, true
		}
	}
	for _, p := range dedupe(paths) {
		if managed, ok := managedInstallByPackageDB(p); ok {
			return managed, true
		}
	}
	return nil, false
}

func managedInstallByPath(p string) (*ManagedInstall, bool) {
	clean := filepath.ToSlash(filepath.Clean(p))
	switch {
	case strings.HasPrefix(clean, "/nix/store/"):
		return &ManagedInstall{Manager: "nix", Command: "update your nix profile, flake, or shell input", Path: p}, true
	case strings.Contains(clean, "/Cellar/konfi/"),
		strings.Contains(clean, "/Caskroom/konfi/"),
		strings.HasPrefix(clean, "/opt/homebrew/bin/"),
		strings.HasPrefix(clean, "/usr/local/Homebrew/"),
		strings.HasPrefix(clean, "/home/linuxbrew/.linuxbrew/"):
		return &ManagedInstall{Manager: "homebrew", Command: "brew upgrade konfi", Path: p}, true
	default:
		return nil, false
	}
}

func managedInstallByPackageDB(p string) (*ManagedInstall, bool) {
	checks := []struct {
		manager string
		command string
		args    []string
		update  string
	}{
		{"dpkg", "dpkg-query", []string{"-S", p}, "sudo apt install --only-upgrade konfi"},
		{"rpm", "rpm", []string{"-qf", p}, "sudo dnf upgrade konfi"},
		{"pacman", "pacman", []string{"-Qo", p}, "sudo pacman -Syu konfi"},
		{"apk", "apk", []string{"info", "-W", p}, "sudo apk upgrade konfi"},
	}

	for _, check := range checks {
		if _, err := exec.LookPath(check.command); err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		cmd := exec.CommandContext(ctx, check.command, check.args...)
		err := cmd.Run()
		cancel()
		if err == nil {
			return &ManagedInstall{Manager: check.manager, Command: check.update, Path: p}, true
		}
	}
	return nil, false
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
