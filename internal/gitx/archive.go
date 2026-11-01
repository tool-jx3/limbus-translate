package gitx

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Archive writes the files of paths, as they are in rev, below dest: the
// output of "git archive --format=tar rev -- paths" is unpacked with
// archive/tar, so no tar command is needed. rev is a tag, a branch or a
// commit; paths are relative to the root of repo, and a directory brings
// everything under it. The bytes are those stored in git; the attributes of
// the revision's own tree decide whether git converts them, which the raw
// branch turns off ("* -text").
//
// dest is created if needed and should be new or empty: a file that already
// exists there is an error, and a failure part-way leaves the files extracted
// so far. The caller removes dest.
//
// Entries that could escape dest, or write through a link, are refused: an
// absolute path, a ".." in the path, a link of any kind, and anything but
// regular files and directories.
func Archive(repo, rev string, paths []string, dest string) error {
	if strings.HasPrefix(rev, "-") {
		return fmt.Errorf("gitx: archive: revision %q looks like an option", rev)
	}
	out, err := git(repo, append([]string{"archive", "--format=tar", rev, "--"}, paths...)...)
	if err != nil {
		return err
	}
	return extractTar(strings.NewReader(out), dest)
}

// extractTar unpacks the tar stream r below dest. See Archive for what it
// refuses.
func extractTar(r io.Reader, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("gitx: read the archive: %w", err)
		}
		switch hdr.Typeflag {
		case tar.TypeXGlobalHeader:
			// git archive starts with one that records the commit id.
		case tar.TypeDir:
			target, err := entryTarget(dest, hdr.Name)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			target, err := entryTarget(dest, hdr.Name)
			if err != nil {
				return err
			}
			if err := writeEntry(target, tr); err != nil {
				return fmt.Errorf("gitx: extract %s: %w", hdr.Name, err)
			}
		default:
			return fmt.Errorf("gitx: archive entry %q is not a regular file or a directory (type %q)", hdr.Name, hdr.Typeflag)
		}
	}
}

// entryTarget maps the name of an archive entry to its place below dest, or
// refuses a name that is absolute or holds "..".
func entryTarget(dest, name string) (string, error) {
	rel := strings.TrimSuffix(name, "/")
	if rel == "" || slices.Contains(strings.Split(rel, "/"), "..") || !filepath.IsLocal(filepath.FromSlash(rel)) {
		return "", fmt.Errorf("gitx: archive entry %q is outside the destination", name)
	}
	return filepath.Join(dest, filepath.FromSlash(rel)), nil
}

// writeEntry creates the file at target with the bytes of r. O_EXCL makes a
// link or a file that is already there an error instead of being written to.
func writeEntry(target string, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
