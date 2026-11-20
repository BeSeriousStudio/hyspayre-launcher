package core

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogSignature(t *testing.T) {
	raw, err := os.ReadFile("../../cmd/launcher/ui/catalog_fallback.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := VerifyCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Games) != 1 || c.Games[0].ID != "hyspayre" {
		t.Fatalf("games: %+v", c.Games)
	}
	c.Games[0].Status = "released"
	hacked, _ := json.Marshal(c)
	if _, err := VerifyCatalog(hacked); err == nil {
		t.Fatal("Unsigned modification was accepted")
	}
}
func TestPathRules(t *testing.T) {
	for _, x := range []string{"../hello.exe", "C:/evil.exe", "../../x.exe", "folder/../x.exe", "/x.exe", "evil.bat", "x\\y.exe"} {
		if ValidExePath(x) {
			t.Errorf("Unexpected allowed executable: %s", x)
		}
	}
	if !ValidExePath("Hyspayre.exe") || !ValidExePath("bin/Hyspayre.exe") {
		t.Fatal("Valid exe path rejected")
	}
	if ValidHTTPS("http://hyspayre.com/x") || ValidHTTPS("https://other.example/x") || !ValidHTTPS("https://hyspayre.com/launcher/games/test/1.zip") {
		t.Fatal("Download URL security failed")
	}
}
func createZip(t *testing.T, file, name string, contents []byte) {
	t.Helper()
	target, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(target)
	f, e := z.Create(name)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.Write(contents); e != nil {
		t.Fatal(e)
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	if e := target.Close(); e != nil {
		t.Fatal(e)
	}
}
func TestArchiveInstall(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "export.zip")
	createZip(t, zipPath, "Hyspayre.exe", []byte("binary-data"))
	bin, err := InstallArchive(filepath.Join(root, "games"), "hyspayre", zipPath, "Hyspayre.exe")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(bin)
	if err != nil || !bytes.Equal(raw, []byte("binary-data")) {
		t.Fatalf("install: %v", err)
	}
	createZip(t, zipPath, "Hyspayre.exe", []byte("update"))
	if _, err := InstallArchive(filepath.Join(root, "games"), "hyspayre", zipPath, "Hyspayre.exe"); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(bin)
	if string(raw) != "update" {
		t.Fatal("update not applied")
	}
	if _, err := os.Stat(filepath.Join(root, "games", "hyspayre.backup")); !os.IsNotExist(err) {
		t.Fatal("Backup not cleaned up")
	}
}
func TestZipTraversalRejected(t *testing.T) {
	for _, name := range []string{"../outside.exe", "/abs.exe", "a\\b.exe", "CON.exe", "file:name.exe", "folder/../bad.exe"} {
		t.Run(strings.ReplaceAll(name, "/", "_"), func(t *testing.T) {
			root := t.TempDir()
			z := filepath.Join(root, "unsafe.zip")
			createZip(t, z, name, []byte("wrong"))
			if err := ExtractArchive(z, filepath.Join(root, "dest")); err == nil {
				t.Errorf("Unsafe path accepted %q", name)
			}
		})
	}
}
