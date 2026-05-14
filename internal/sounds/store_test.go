package sounds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertFromClientRenamesCollision(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "horn.mp3"), []byte("server"), 0o640); err != nil {
		t.Fatalf("seed file failed: %v", err)
	}
	if err := store.upsertCatalog("horn.mp3", metadata{Origin: "server"}); err != nil {
		t.Fatalf("seed catalog failed: %v", err)
	}

	info, err := store.UpsertFromClient("horn.mp3", []byte("client"), 1<<20, "", true)
	if err != nil {
		t.Fatalf("UpsertFromClient failed: %v", err)
	}
	if info.FileName == "horn.mp3" {
		t.Fatalf("expected collision rename, got %q", info.FileName)
	}
	if info.Origin != "client" {
		t.Fatalf("expected origin client, got %q", info.Origin)
	}
	if !info.IsDefault {
		t.Fatalf("expected default marker true")
	}
}

func TestSaveUploadedSetsServerOrigin(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if _, err := store.SaveUploaded("upload.mp3", strings.NewReader("abc"), 1<<20); err != nil {
		t.Fatalf("SaveUploaded failed: %v", err)
	}
	items, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if items[0].Origin != "server" {
		t.Fatalf("expected origin server, got %q", items[0].Origin)
	}
}
