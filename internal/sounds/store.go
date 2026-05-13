package sounds

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const selectedFileName = ".selected.json"

// SelectedState persists chosen sound on disk.
type SelectedState struct {
	FileName string `json:"fileName"`
}

// Store manages sound files under a directory.
type Store struct {
	dir string
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
		ct := mime.TypeByExtension(filepath.Ext(e.Name()))
		if ct == "" {
			ct = "application/octet-stream"
		}
		out = append(out, Info{
			FileName:    e.Name(),
			SizeBytes:   info.Size(),
			ContentType: ct,
			UpdatedAt:   info.ModTime().UTC(),
		})
	}
	return out, nil
}

type Info struct {
	FileName    string
	SizeBytes   int64
	ContentType string
	UpdatedAt   time.Time
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
	return n, nil
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
