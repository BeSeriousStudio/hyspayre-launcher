package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Installed struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Executable  string `json:"executable"`
	InstalledAt string `json:"installed_at"`
}

type Task struct {
	ID      string `json:"id"`
	Stage   string `json:"stage"`
	Message string `json:"message"`
	Done    int64  `json:"done"`
	Total   int64  `json:"total"`
}

type Settings struct {
	InstallRoot string `json:"install_root"`
}

type Snapshot struct {
	MSIXManaged  bool                 `json:"msix_managed"`
	Version      string               `json:"version"`
	Catalog      Catalog              `json:"catalog"`
	CatalogError string               `json:"catalog_error"`
	Installed    map[string]Installed `json:"installed"`
	Tasks        map[string]Task      `json:"tasks"`
	Settings     Settings             `json:"settings"`
}

type Manager struct {
	mu           sync.RWMutex
	Catalog      Catalog
	CatalogError string
	Installed    map[string]Installed
	Tasks        map[string]Task
	Settings     Settings
	Dir          string
	Active       bool
	Session      string
	LastSeen     time.Time
	Exit         func()
}

func DataDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		var e error
		base, e = os.UserConfigDir()
		if e != nil {
			return "", e
		}
	}
	return filepath.Join(base, "HyspayreLauncher"), nil
}

func NewManager(fallback []byte) (*Manager, error) {
	dir, err := DataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return nil, err
	}
	c, err := VerifyCatalog(fallback)
	if err != nil {
		return nil, fmt.Errorf("Fallback-Katalog ungültig: %w", err)
	}
	m := &Manager{Dir: dir, Session: hex.EncodeToString(b), Catalog: c, Installed: make(map[string]Installed), Tasks: make(map[string]Task), LastSeen: time.Now()}
	m.Settings.InstallRoot = filepath.Join(dir, "Games")
	m.readState()
	if err := os.MkdirAll(m.Settings.InstallRoot, 0700); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) readState() {
	raw, err := os.ReadFile(filepath.Join(m.Dir, "state.json"))
	if err != nil {
		return
	}
	var state struct {
		Installed map[string]Installed `json:"installed"`
		Settings  Settings             `json:"settings"`
	}
	if json.Unmarshal(raw, &state) != nil {
		return
	}
	if state.Installed != nil {
		m.Installed = state.Installed
	}
	if state.Settings.InstallRoot != "" && filepath.IsAbs(state.Settings.InstallRoot) {
		m.Settings = state.Settings
	}
}

func (m *Manager) saveState() error {
	state := struct {
		Installed map[string]Installed `json:"installed"`
		Settings  Settings             `json:"settings"`
	}{m.Installed, m.Settings}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	temp := filepath.Join(m.Dir, "state.tmp")
	if err := os.WriteFile(temp, raw, 0600); err != nil {
		return err
	}
	return os.Rename(temp, filepath.Join(m.Dir, "state.json"))
}

func (m *Manager) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := Snapshot{MSIXManaged: IsMSIX(), Version: ClientVersion, Catalog: m.Catalog, CatalogError: m.CatalogError, Installed: make(map[string]Installed), Tasks: make(map[string]Task), Settings: m.Settings}
	s.Catalog.Signature = "" // not needed by the visual interface
	for k, v := range m.Installed {
		s.Installed[k] = v
	}
	for k, v := range m.Tasks {
		s.Tasks[k] = v
	}
	return s
}

func (m *Manager) Refresh() error {
	client := http.Client{Timeout: 15 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 || !ValidHTTPS(req.URL.String()) {
			return errors.New("Unsichere Katalog-Weiterleitung")
		}
		return nil
	}}
	req, err := http.NewRequest("GET", CatalogURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := client.Do(req)
	if err == nil {
		if resp.StatusCode != 200 {
			err = fmt.Errorf("Katalog HTTP %d", resp.StatusCode)
		} else {
			var data []byte
			data, err = io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
			if err == nil {
				var c Catalog
				c, err = VerifyCatalog(data)
				if err == nil {
					m.mu.Lock()
					m.Catalog = c
					m.CatalogError = ""
					m.mu.Unlock()
					os.WriteFile(filepath.Join(m.Dir, "catalog.cache.json"), data, 0600)
				}
			}
		}
		resp.Body.Close()
	}
	if err != nil {
		// Use a previously cryptographically verified catalog offline, if available.
		m.mu.Lock()
		m.CatalogError = "Offline: " + err.Error()
		m.mu.Unlock()
		raw, re := os.ReadFile(filepath.Join(m.Dir, "catalog.cache.json"))
		if re == nil {
			c, ve := VerifyCatalog(raw)
			if ve == nil {
				m.mu.Lock()
				m.Catalog = c
				m.mu.Unlock()
			}
		}
	}
	return err
}

func (m *Manager) findGame(id string) (Game, bool) {
	for _, g := range m.Catalog.Games {
		if g.ID == id {
			return g, true
		}
	}
	return Game{}, false
}

func (m *Manager) setTask(id, stage, msg string, done, total int64) {
	m.mu.Lock()
	m.Tasks[id] = Task{ID: id, Stage: stage, Message: msg, Done: done, Total: total}
	m.mu.Unlock()
}

func (m *Manager) StartInstall(id string, force bool) error {
	m.mu.Lock()
	game, found := m.findGame(id)
	if !found || game.Status != "released" {
		m.mu.Unlock()
		return errors.New("Spiel ist noch nicht veröffentlicht")
	}
	if t, ok := m.Tasks[id]; ok && (t.Stage == "download" || t.Stage == "install") {
		m.mu.Unlock()
		return errors.New("Installation läuft bereits")
	}
	if installed, ok := m.Installed[id]; ok && installed.Version == game.Version && !force {
		m.mu.Unlock()
		return errors.New("Diese Version ist bereits installiert")
	}
	installDir := m.Settings.InstallRoot
	m.Tasks[id] = Task{ID: id, Stage: "download", Message: "Download startet ...", Total: game.Size}
	m.mu.Unlock()
	go func() {
		archive := filepath.Join(m.Dir, "Cache", id+"-"+Sha256Hex([]byte(game.Version))[:12]+".zip")
		// Even a cached archive is verified again. No unauthenticated executable runs.
		err := func() error {
			if fi, e := os.Stat(archive); e == nil && fi.Size() == game.Size {
				if hash, e := hashFile(archive); e == nil && hash == game.SHA256 {
					return nil
				}
			}
			return DownloadVerified(game.PackageURL, game.SHA256, game.Size, archive, func(done, total int64) {
				m.setTask(id, "download", "Lädt Spieldateien ...", done, total)
			})
		}()
		if err != nil {
			m.setTask(id, "error", err.Error(), 0, 0)
			return
		}

		// When a cached package exists, its hash was checked above.
		if hash, e := hashFile(archive); e != nil || hash != game.SHA256 {
			m.setTask(id, "error", "Download fehlgeschlagen oder Prüfsumme ungültig", 0, 0)
			return
		}
		m.setTask(id, "install", "Installiert und überprüft Dateien ...", game.Size, game.Size)
		_, err = InstallArchive(installDir, id, archive, game.Executable)
		if err != nil {
			m.setTask(id, "error", err.Error(), 0, 0)
			return
		}
		m.mu.Lock()
		m.Installed[id] = Installed{ID: id, Version: game.Version, Executable: game.Executable, InstalledAt: time.Now().Format(time.RFC3339)}
		e := m.saveState()
		m.mu.Unlock()
		if e != nil {
			m.setTask(id, "error", "Installiert, aber Status konnte nicht gespeichert werden: "+e.Error(), 0, 0)
			return
		}
		m.setTask(id, "done", "Bereit zum Spielen", game.Size, game.Size)
	}()
	return nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (m *Manager) Launch(id string) error {
	m.mu.RLock()
	inst, ok := m.Installed[id]
	base := m.Settings.InstallRoot
	m.mu.RUnlock()
	if !ok {
		return errors.New("Spiel ist nicht installiert")
	}
	if !ValidExePath(inst.Executable) {
		return errors.New("Ungültiger Spielpfad")
	}
	folder, err := SafeInstallDir(base, id)
	if err != nil {
		return err
	}
	exe := filepath.Join(folder, filepath.FromSlash(inst.Executable))
	if fi, err := os.Stat(exe); err != nil || fi.IsDir() {
		return errors.New("Spielstartdatei fehlt: Bitte reparieren")
	}
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func (m *Manager) Uninstall(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.Installed[id]; !ok {
		return errors.New("Spiel ist nicht installiert")
	}
	if t, ok := m.Tasks[id]; ok && (t.Stage == "download" || t.Stage == "install") {
		return errors.New("Bitte zuerst Download beenden")
	}
	target, err := SafeInstallDir(m.Settings.InstallRoot, id)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	delete(m.Installed, id)
	delete(m.Tasks, id)
	return m.saveState()
}

func (m *Manager) ChangeInstallRoot(root string) error {
	root = strings.TrimSpace(root)
	if !filepath.IsAbs(root) {
		return errors.New("Bitte einen vollständigen absoluten Pfad angeben")
	}
	if root == filepath.VolumeName(root)+string(os.PathSeparator) {
		return errors.New("Laufwerkswurzel ist kein geeigneter Installationsordner")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Installed) > 0 {
		return errors.New("Installationsordner erst ändern, wenn keine Spiele installiert sind")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	probe := filepath.Join(root, ".hyspayre-write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0600); err != nil {
		return err
	}
	os.Remove(probe)
	m.Settings.InstallRoot = root
	return m.saveState()
}

func (m *Manager) LauncherUpdate() error {
    // Never modify an MSIX package installed in WindowsApps: updates belong to
    // Microsoft Store or Windows App Installer, and package contents are protected.
    if IsMSIX() {
        return errors.New("Dieser Launcher wurde als MSIX installiert. Updates erfolgen über den Microsoft Store oder Windows App Installer.")
    }
	m.mu.Lock()
	r := m.Catalog.Launcher
	if r.URL == "" || r.Version == "" || CompareVersion(r.Version, ClientVersion) <= 0 {
		m.mu.Unlock()
		return errors.New("Client ist aktuell")
	}
	if t, ok := m.Tasks["launcher"]; ok && (t.Stage == "download" || t.Stage == "install") {
		m.mu.Unlock()
		return errors.New("Update läuft bereits")
	}
	m.Tasks["launcher"] = Task{ID: "launcher", Stage: "download", Message: "Client-Update wird geladen ...", Total: r.Size}
	m.mu.Unlock()
	go func() {
		target := filepath.Join(m.Dir, "Cache", "Hyspayre-Launcher-"+r.Version+".exe")
		err := DownloadVerified(r.URL, r.SHA256, r.Size, target, func(done, total int64) {
			m.setTask("launcher", "download", "Client-Update wird geladen ...", done, total)
		})
		if err != nil {
			m.setTask("launcher", "error", err.Error(), 0, 0)
			return
		}
		current, e := os.Executable()
		if e != nil {
			m.setTask("launcher", "error", e.Error(), 0, 0)
			return
		}
		cmd := exec.Command(target, "--apply-update", current)
		if e = cmd.Start(); e != nil {
			m.setTask("launcher", "error", e.Error(), 0, 0)
			return
		}
		cmd.Process.Release()
		m.setTask("launcher", "done", "Client startet neu ...", r.Size, r.Size)
		if m.Exit != nil {
			go func() { time.Sleep(1300 * time.Millisecond); m.Exit() }()
		}
	}()
	return nil
}

func CompareVersion(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var x, y int
		if len(pa) > i {
			fmt.Sscanf(pa[i], "%d", &x)
		}
		if len(pb) > i {
			fmt.Sscanf(pb[i], "%d", &y)
		}
		if x > y {
			return 1
		}
		if x < y {
			return -1
		}
	}
	return 0
}

func (m *Manager) MuTouch(update bool) {
	if !update {
		return
	}
	m.mu.Lock()
	m.LastSeen = time.Now()
	m.mu.Unlock()
}
func (m *Manager) LastActivity() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.LastSeen
}
