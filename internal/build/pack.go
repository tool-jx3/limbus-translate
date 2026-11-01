package build

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"golang.org/x/image/font/sfnt"
)

const (
	// dataDir is the top directory of the pack, in the worktree and in the zips.
	dataDir = "LimbusCompany_Data"
	// configPath is the file that makes the game use the language directory.
	configPath = dataDir + "/Lang/config.json"
	// fontFile is the name the font has in the zip, whatever it is called on
	// disk; oflFile is its licence, which lies next to the font on disk (see
	// FontLicensePath).
	fontFile = "SarasaGothicTC-Bold.ttf"
	oflFile  = "OFL.txt"
)

// langPath is the path, relative to the worktree, of rel inside the language
// directory.
func langPath(opt Options, rel string) string {
	return dataDir + "/Lang/" + opt.LangDir + "/" + rel
}

// configJSON is the content of Lang/config.json.
func configJSON(langDir string) []byte {
	name, _ := json.Marshal(langDir) // a string cannot fail to encode
	return []byte(`{"lang": ` + string(name) + "}\n")
}

// ZipPaths returns where Run puts the zip with the text only and the zip that
// also holds the font.
func ZipPaths(opt Options) (text, complete string) {
	name := func(kind string) string {
		return filepath.Join(opt.OutDir, opt.LangDir+"_"+kind+"_"+opt.RawTag+".zip")
	}
	return name("text"), name("complete")
}

// FontLicensePath is where the licence of the font at fontPath must lie: next
// to it, called OFL.txt.
func FontLicensePath(fontPath string) string {
	return filepath.Join(filepath.Dir(fontPath), oflFile)
}

// inputs are the files besides the records that a build reads. They are all
// read and checked before anything is written.
type inputs struct {
	font, ofl, license []byte
	face               *sfnt.Font
}

func loadInputs(opt Options) (*inputs, error) {
	var in inputs
	var err error
	if in.font, err = os.ReadFile(opt.FontPath); err != nil {
		return nil, fmt.Errorf("build: read font: %w", err)
	}
	sum := sha256.Sum256(in.font)
	if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, opt.FontSHA256) {
		return nil, fmt.Errorf("build: %w: %s has sha256 %s, want %q", ErrFontChecksum, opt.FontPath, got, opt.FontSHA256)
	}
	if in.face, err = sfnt.Parse(in.font); err != nil {
		return nil, fmt.Errorf("build: parse font %s: %w", opt.FontPath, err)
	}
	if in.ofl, err = os.ReadFile(FontLicensePath(opt.FontPath)); err != nil {
		return nil, fmt.Errorf("build: read font licence: %w", err)
	}
	if in.license, err = os.ReadFile(opt.LicensePath); err != nil {
		return nil, fmt.Errorf("build: read licence: %w", err)
	}
	return &in, nil
}

// missingGlyphs returns, sorted, the characters of set that the font has no
// glyph for. Control characters are not glyphs.
func (in *inputs) missingGlyphs(set *runeSet) ([]rune, error) {
	var buf sfnt.Buffer
	missing := []rune{}
	for r := range set.all() {
		if r < 0x20 {
			continue
		}
		idx, err := in.face.GlyphIndex(&buf, r)
		if err != nil {
			return nil, fmt.Errorf("build: look up %U in the font: %w", r, err)
		}
		if idx == 0 {
			missing = append(missing, r)
		}
	}
	return missing, nil
}

// tree writes the files of the pack into a worktree and remembers which ones.
type tree struct {
	root    string
	written map[string]bool // slash paths relative to root
}

func newTree(root string) *tree {
	return &tree{root: root, written: map[string]bool{}}
}

func (t *tree) path(rel string) string {
	return filepath.Join(t.root, filepath.FromSlash(rel))
}

// write puts data at rel. A file that already holds data is not touched, so
// that its time stays and git has nothing to look at.
func (t *tree) write(rel string, data []byte) error {
	t.written[rel] = true
	full := t.path(rel)
	if old, err := os.ReadFile(full); err == nil && bytes.Equal(old, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return nil
}

// prune removes what a build did not write from LimbusCompany_Data (output of
// files KR no longer has, the font of an older layout, another language's
// directory), then the directories that became empty. Nothing outside
// LimbusCompany_Data is looked at, so .meta stays.
func (t *tree) prune() error {
	top := t.path(dataDir)
	var dirs []string
	err := filepath.WalkDir(top, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != top {
				dirs = append(dirs, p)
			}
			return nil
		}
		rel, err := filepath.Rel(t.root, p)
		if err != nil {
			return err
		}
		if !t.written[filepath.ToSlash(rel)] {
			return os.Remove(p)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("build: remove stale output: %w", err)
	}
	for i := len(dirs) - 1; i >= 0; i-- { // children before parents
		entries, err := os.ReadDir(dirs[i])
		if err != nil {
			return fmt.Errorf("build: remove stale output: %w", err)
		}
		if len(entries) == 0 {
			if err := os.Remove(dirs[i]); err != nil {
				return fmt.Errorf("build: remove stale output: %w", err)
			}
		}
	}
	return nil
}

// zipEntry is one file of a zip: data, or when data is nil the file at path.
type zipEntry struct {
	name string
	data []byte
	path string
}

// writeZips writes the two zips. The text zip holds the pack as it is in the
// worktree; the complete zip adds the font, twice (the game reads Context and
// Title separately), and its licence.
func writeZips(opt Options, t *tree, in *inputs) error {
	var text []zipEntry
	for rel := range t.written {
		text = append(text, zipEntry{name: rel, path: t.path(rel)})
	}
	complete := append(slices.Clone(text),
		zipEntry{name: langPath(opt, "Font/Context/"+fontFile), data: in.font},
		zipEntry{name: langPath(opt, "Font/Title/"+fontFile), data: in.font},
		zipEntry{name: langPath(opt, "Font/"+oflFile), data: in.ofl},
	)
	textZip, completeZip := ZipPaths(opt)
	modified := opt.BuiltAt.UTC().Truncate(time.Second)
	if err := writeZip(textZip, text, modified); err != nil {
		return err
	}
	return writeZip(completeZip, complete, modified)
}

// writeZip writes a zip of entries sorted by name, each deflated and stamped
// with modified, and no entry for a directory: the same entries give the same
// bytes. It goes to a temporary file first, so an interrupted run never leaves
// a half-written zip under the final name.
func writeZip(dst string, entries []zipEntry, modified time.Time) (err error) {
	slices.SortFunc(entries, func(a, b zipEntry) int { return strings.Compare(a.name, b.name) })
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".tmp*")
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(tmp)
		}
	}()

	zw := zip.NewWriter(f)
	for _, e := range entries {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: e.name, Method: zip.Deflate, Modified: modified})
		if err != nil {
			return fmt.Errorf("build: zip %s: %w", e.name, err)
		}
		if e.data != nil {
			_, err = w.Write(e.data)
		} else {
			err = copyFile(w, e.path)
		}
		if err != nil {
			return fmt.Errorf("build: zip %s: %w", e.name, err)
		}
	}
	if err = zw.Close(); err != nil {
		return fmt.Errorf("build: zip %s: %w", dst, err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("build: zip %s: %w", dst, err)
	}
	if err = os.Chmod(tmp, 0o644); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	if err = os.Rename(tmp, dst); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return nil
}

func copyFile(w io.Writer, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(w, src)
	return err
}
