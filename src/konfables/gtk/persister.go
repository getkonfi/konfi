package gtk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eminert/konfi/pkg"
)

// settingsFiles lists every gtk settings.ini location konfi knows about,
// in mirror-preference order (gtk-4.0 preferred as primary).
var settingsDirs = []string{"gtk-4.0", "gtk-3.0"}

// MirrorPersister writes identical content to every existing gtk settings.ini.
// it embeds *pkg.FilePersister for the primary file (Load, Path, Watch) and
// overrides Save to fan the same bytes out to the mirror targets.
type MirrorPersister struct {
	*pkg.FilePersister
	mirrors []string // additional settings.ini paths that exist at construction
}

// NewMirrorPersister builds a persister whose primary is `primary` and whose
// mirror targets are the given sibling paths. only existing siblings should be
// passed so konfi never spawns a gtk-3 config on a gtk-4-only box.
func NewMirrorPersister(primary string, mirrors ...string) *MirrorPersister {
	return &MirrorPersister{
		FilePersister: pkg.NewFilePersister(primary),
		mirrors:       mirrors,
	}
}

// Save writes the primary via the embedded persister, then mirrors the same
// bytes to every sibling. each mirror gets its own konfi backup of its prior content.
func (mp *MirrorPersister) Save(ctx context.Context, original, data []byte) error {
	if err := mp.FilePersister.Save(ctx, original, data); err != nil {
		return err
	}
	for _, path := range mp.mirrors {
		if err := mp.writeMirror(path, data); err != nil {
			return fmt.Errorf("mirror %s: %w", path, err)
		}
	}
	return nil
}

// writeMirror backs up the mirror's current content and atomically writes data,
// preserving the file's existing permissions.
func (mp *MirrorPersister) writeMirror(path string, data []byte) error {
	perm := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
		if prior, rerr := os.ReadFile(path); rerr == nil {
			if werr := pkg.WriteBackup(path, prior, perm, mp.BackupLimit()); werr != nil {
				return fmt.Errorf("backup: %w", werr)
			}
		}
	}
	if err := pkg.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return pkg.AtomicWrite(path, data, perm)
}

// ResolvePaths picks the primary settings.ini and lists existing mirrors.
// primary preference: gtk-4.0 if it exists, else gtk-3.0, else gtk-3.0 default.
// mirrors: every other settings.ini that currently exists (never created).
func ResolvePaths() (primary string, mirrors []string) {
	existing := make(map[string]bool, len(settingsDirs))
	for _, dir := range settingsDirs {
		p := pkg.XDGConfigPath(dir, "settings.ini")
		if pkg.FileExists(p) {
			existing[dir] = true
		}
	}

	switch {
	case existing["gtk-4.0"]:
		primary = pkg.XDGConfigPath("gtk-4.0", "settings.ini")
	case existing["gtk-3.0"]:
		primary = pkg.XDGConfigPath("gtk-3.0", "settings.ini")
	default:
		primary = pkg.XDGConfigPath("gtk-3.0", "settings.ini")
	}

	for _, dir := range settingsDirs {
		p := pkg.XDGConfigPath(dir, "settings.ini")
		if existing[dir] && p != primary {
			mirrors = append(mirrors, p)
		}
	}
	return primary, mirrors
}
