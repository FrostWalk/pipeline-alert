package sounds

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const selectedFileName = ".selected.json"
const catalogFileName = ".catalog.json"

// SelectedState persists chosen sound on disk.
type SelectedState struct {
	FileName string `json:"fileName"`
}

// Store manages sound files under a directory.
type Store struct {
	dir string
}

type metadata struct {
	Origin    string `json:"origin,omitempty"`
	IsDefault bool   `json:"isDefault,omitempty"`
}

func NewStore(dir string) (*Store, error) {
	d := filepath.Clean(dir)
	if err := os.MkdirAll(d, 0o750); err != nil {
		return nil, fmt.Errorf("create sounds dir: %w", err)
	}
	return &Store{dir: d}, nil
}

func (s *Store) Dir() string { return s.dir }

// List returns non-hidden files in the store.
func (s *Store) List() ([]Info, error) {
	catalog, _ := s.readCatalog()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var out []Info
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		sum, err := s.hashFile(e.Name())
		if err != nil {
			continue
		}
		ct := mime.TypeByExtension(filepath.Ext(e.Name()))
		if ct == "" {
			ct = "application/octet-stream"
		}
		meta := catalog[e.Name()]
		out = append(out, Info{
			FileName:    e.Name(),
			SizeBytes:   info.Size(),
			ContentType: ct,
			UpdatedAt:   info.ModTime().UTC(),
			SHA256:      sum,
			Origin:      meta.Origin,
			IsDefault:   meta.IsDefault,
		})
	}
	return out, nil
}

type Info struct {
	FileName    string
	SizeBytes   int64
	ContentType string
	UpdatedAt   time.Time
	SHA256      string
	Origin      string
	IsDefault   bool
}

func (s *Store) Selected() (string, error) {
	b, err := os.ReadFile(filepath.Join(s.dir, selectedFileName))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var st SelectedState
	if err := json.Unmarshal(b, &st); err != nil {
		return "", err
	}
	return strings.TrimSpace(st.FileName), nil
}

func (s *Store) SetSelected(name string) error {
	st := SelectedState{FileName: strings.TrimSpace(name)}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, ".selected.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, selectedFileName))
}

var ErrExists = errors.New("sound file already exists")
var ErrNotFound = errors.New("sound file not found")

// SaveUploaded writes a new sound file; rejects if file exists.
func (s *Store) SaveUploaded(fileName string, src io.Reader, maxBytes int64) (int64, error) {
	base := filepath.Base(fileName)
	if base != fileName || base == "." || base == ".." || strings.HasPrefix(base, ".") {
		return 0, fmt.Errorf("invalid file name")
	}
	dst := filepath.Join(s.dir, base)
	if _, err := os.Stat(dst); err == nil {
		return 0, ErrExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}

	tmp := dst + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	n, err := io.Copy(f, io.LimitReader(src, maxBytes+1))
	if err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	if n > maxBytes {
		_ = os.Remove(tmp)
		return 0, fmt.Errorf("file exceeds max size")
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	_ = s.upsertCatalog(base, metadata{Origin: "server", IsDefault: false})
	return n, nil
}

// UpsertFromClient stores sound bytes sent by Pi client.
// If a filename collision has different content, it stores a deterministic renamed copy.
func (s *Store) UpsertFromClient(fileName string, data []byte, maxBytes int64, expectedSHA256 string, isDefault bool) (Info, error) {
	base := filepath.Base(fileName)
	if base != fileName || base == "." || base == ".." || strings.HasPrefix(base, ".") {
		return Info{}, fmt.Errorf("invalid file name")
	}
	if int64(len(data)) > maxBytes {
		return Info{}, fmt.Errorf("file exceeds max size")
	}
	actual := hashBytes(data)
	if strings.TrimSpace(expectedSHA256) != "" && !strings.EqualFold(actual, strings.TrimSpace(expectedSHA256)) {
		return Info{}, fmt.Errorf("sha256 mismatch")
	}

	targetName := base
	existing, err := s.ReadFileBytes(base, maxBytes)
	if err == nil {
		existingHash := hashBytes(existing)
		if !strings.EqualFold(existingHash, actual) {
			targetName = collisionName(base, actual)
		}
	}

	if has, _ := s.Has(targetName); !has {
		if err := os.WriteFile(filepath.Join(s.dir, targetName), data, 0o640); err != nil {
			return Info{}, err
		}
	}
	_ = s.upsertCatalog(targetName, metadata{Origin: "client", IsDefault: isDefault})
	return s.buildInfo(targetName)
}

func (s *Store) Has(fileName string) (bool, error) {
	base := filepath.Base(fileName)
	dst := filepath.Join(s.dir, base)
	fi, err := os.Stat(dst)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return fi.Mode().IsRegular(), nil
}

func (s *Store) Open(fileName string) (*os.File, error) {
	base := filepath.Base(fileName)
	dst := filepath.Join(s.dir, base)
	f, err := os.Open(dst)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

func (s *Store) ReadFileBytes(fileName string, max int64) ([]byte, error) {
	f, err := s.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, fmt.Errorf("file exceeds max read size")
	}
	return b, nil
}

func (s *Store) buildInfo(fileName string) (Info, error) {
	fi, err := os.Stat(filepath.Join(s.dir, fileName))
	if err != nil {
		return Info{}, err
	}
	sum, err := s.hashFile(fileName)
	if err != nil {
		return Info{}, err
	}
	catalog, _ := s.readCatalog()
	meta := catalog[fileName]
	ct := mime.TypeByExtension(filepath.Ext(fileName))
	if ct == "" {
		ct = "application/octet-stream"
	}
	return Info{
		FileName:    fileName,
		SizeBytes:   fi.Size(),
		ContentType: ct,
		UpdatedAt:   fi.ModTime().UTC(),
		SHA256:      sum,
		Origin:      meta.Origin,
		IsDefault:   meta.IsDefault,
	}, nil
}

func (s *Store) hashFile(fileName string) (string, error) {
	f, err := s.Open(fileName)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hexHash(h), nil
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:])
}

func hexHash(h hash.Hash) string {
	return fmt.Sprintf("%x", h.Sum(nil))
}

func collisionName(fileName, sha string) string {
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	short := strings.ToLower(sha)
	if len(short) > 12 {
		short = short[:12]
	}
	return base + ".client-" + short + ext
}

func (s *Store) readCatalog() (map[string]metadata, error) {
	b, err := os.ReadFile(filepath.Join(s.dir, catalogFileName))
	if errors.Is(err, os.ErrNotExist) {
		return map[string]metadata{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out map[string]metadata
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]metadata{}, nil
	}
	return out, nil
}

func (s *Store) upsertCatalog(fileName string, meta metadata) error {
	catalog, err := s.readCatalog()
	if err != nil {
		return err
	}
	prev := catalog[fileName]
	if prev.Origin == "server" && meta.Origin == "client" {
		meta.Origin = "server"
	}
	meta.IsDefault = prev.IsDefault || meta.IsDefault
	catalog[fileName] = meta
	b, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, catalogFileName+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, catalogFileName))
}
