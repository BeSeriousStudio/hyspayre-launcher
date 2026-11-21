package main

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"hyspayre/launcher/internal/core"
	"hyspayre/launcher/internal/social"
)

//go:embed ui/* ui/assets/*
var UI embed.FS

type server struct {
	m        *core.Manager
	social   *social.Client
	host     string
	origin   string
	h        *http.Server
	exitOnce sync.Once
}

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "--apply-update" {
		applyUpdate(os.Args[2])
		return
	}
	dir, err := core.DataDir()
	if err != nil {
		return
	}
	os.MkdirAll(dir, 0700)
	lf, e := os.OpenFile(filepath.Join(dir, "launcher.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e == nil {
		defer lf.Close()
		log.SetOutput(lf)
	}
	fallback, err := UI.ReadFile("ui/catalog_fallback.json")
	if err != nil {
		log.Printf("Fallback fehlt: %v", err)
		return
	}
	m, err := core.NewManager(fallback)
	if err != nil {
		log.Printf("Manager start failed: %v", err)
		return
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("Loopback bind failed: %v", err)
		return
	}
	s := &server{m: m, social: social.New(m.Dir), host: l.Addr().String(), origin: "http://" + l.Addr().String()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	mux.HandleFunc("/style.css", s.static)
	mux.HandleFunc("/favicon.ico", s.static)
	mux.HandleFunc("/app.js", s.static)
	mux.HandleFunc("/assets/", s.static)
	mux.HandleFunc("/api/bootstrap", s.bootstrap)
	mux.HandleFunc("/api/action", s.action)
	mux.HandleFunc("/api/settings", s.settings)
	mux.HandleFunc("/api/refresh", s.refresh)
	mux.HandleFunc("/api/open", s.open)
	mux.HandleFunc("/api/exit", s.exit)
	mux.HandleFunc("/api/auth/status", s.authStatus)
	mux.HandleFunc("/api/auth/login", s.authLogin)
	mux.HandleFunc("/api/auth/logout", s.authLogout)
	mux.HandleFunc("/api/social/friends", s.socialFriends)
	mux.HandleFunc("/api/social/search", s.socialSearch)
	mux.HandleFunc("/api/social/profile", s.socialProfile)
	mux.HandleFunc("/api/social/action", s.socialAction)
	s.h = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.host {
			http.Error(w, "Ungültiger Host", 403)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' https://hyspayre.com https://amgpdukqacxogafqpmho.supabase.co data:; connect-src 'self'; font-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		mux.ServeHTTP(w, r)
	}), ReadHeaderTimeout: 7 * time.Second}
	m.Exit = func() {
		s.exitOnce.Do(func() {
			appClosing.Store(true)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				s.h.Shutdown(ctx)
			}()
		})
	}
	go func() {
		if err := m.Refresh(); err != nil {
			log.Printf("Catalog refresh: %v", err)
		}
	}()
	go func() {
		if err := s.h.Serve(l); err != nil && err != http.ErrServerClosed {
			log.Printf("Server: %v", err)
		}
	}()
	// Check last interface request; close background process if its dedicated app window closed.
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			m.MuTouch(false)
			if time.Since(m.LastActivity()) > 2*time.Minute {
				m.Exit()
				return
			}
		}
	}()
	url := s.origin + "/?launch=" + m.Session
	if len(os.Args) < 2 || os.Args[1] != "--headless" {
		if err := openAppWindow(url, m.Dir); err != nil {
			log.Printf("Window launch: %v", err)
		}
	} else {
		fmt.Println("TEST_URL=" + url)
		fmt.Println("TEST_TOKEN=" + m.Session)
	}
	// Serve waits until exit; the native game stays open independently.
	<-waitForDone(s.h)
}

func waitForDone(h *http.Server) <-chan struct{} {
	done := make(chan struct{})
	// This helper is not the listener. Avoid waiting for Shutdown alone by watching a private context.
	go func() {
		for {
			time.Sleep(300 * time.Millisecond)
			if appClosing.Load() {
				close(done)
				return
			}
		}
	}()
	return done
}

// A plain boolean guard is intentionally implemented via the enclosing server shutdown below.
var appClosing = &atomicBool{}

type atomicBool struct {
	mu sync.Mutex
	v  bool
}

func (a *atomicBool) Load() bool   { a.mu.Lock(); defer a.mu.Unlock(); return a.v }
func (a *atomicBool) Store(v bool) { a.mu.Lock(); a.v = v; a.mu.Unlock() }

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("launch")), []byte(s.m.Session)) != 1 {
		http.Error(w, "Nicht autorisiert", 403)
		return
	}
	raw, e := UI.ReadFile("ui/index.html")
	if e != nil {
		http.Error(w, "UI fehlt", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	raw = []byte(strings.ReplaceAll(string(raw), "__HYSPAYRE_SESSION__", s.m.Session))
	w.Write(raw)
}
func (s *server) static(w http.ResponseWriter, r *http.Request) {
	path := "ui" + r.URL.Path
	if strings.Contains(path, "..") || strings.HasSuffix(path, "/") {
		http.NotFound(w, r)
		return
	}
	bytes, e := UI.ReadFile(path)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	switch {
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(path, ".webp"):
		w.Header().Set("Content-Type", "image/webp")
	case strings.HasSuffix(path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(path, ".ico"):
		w.Header().Set("Content-Type", "image/x-icon")
	case strings.HasSuffix(path, ".png"):
		w.Header().Set("Content-Type", "image/png")
	}
	w.Write(bytes)
}
func (s *server) authorized(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, "Methode nicht erlaubt", 405)
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != s.origin {
		http.Error(w, "Origin nicht erlaubt", 403)
		return false
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Hyspayre-Token")), []byte(s.m.Session)) != 1 {
		http.Error(w, "Nicht autorisiert", 403)
		return false
	}
	s.m.MuTouch(true)
	return true
}
func sendJSON(w http.ResponseWriter, result any, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(result)
}
func fail(w http.ResponseWriter, err error) {
	sendJSON(w, map[string]string{"error": err.Error()}, 400)
}
func (s *server) bootstrap(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "GET") {
		return
	}
	sendJSON(w, s.m.Snapshot(), 200)
}
func decode(r *http.Request, dst any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("JSON erforderlich")
	}
	return json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(dst)
}
func (s *server) action(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	var msg struct {
		Action string `json:"action"`
		GameID string `json:"game_id"`
	}
	if err := decode(r, &msg); err != nil {
		fail(w, err)
		return
	}
	var err error
	switch msg.Action {
	case "install":
		err = s.m.StartInstall(msg.GameID, false)
	case "repair":
		err = s.m.StartInstall(msg.GameID, true)
	case "launch":
		err = s.m.Launch(msg.GameID)
	case "uninstall":
		err = s.m.Uninstall(msg.GameID)
	case "launcher_update":
		err = s.m.LauncherUpdate()
	default:
		err = fmt.Errorf("Unbekannte Aktion")
	}
	if err != nil {
		fail(w, err)
		return
	}
	sendJSON(w, map[string]bool{"ok": true}, 200)
}
func (s *server) settings(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	var x struct {
		InstallRoot string `json:"install_root"`
	}
	if err := decode(r, &x); err != nil {
		fail(w, err)
		return
	}
	if err := s.m.ChangeInstallRoot(x.InstallRoot); err != nil {
		fail(w, err)
		return
	}
	sendJSON(w, map[string]bool{"ok": true}, 200)
}
func (s *server) refresh(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	if err := s.m.Refresh(); err != nil {
		fail(w, err)
		return
	}
	sendJSON(w, map[string]bool{"ok": true}, 200)
}
func (s *server) open(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	var x struct {
		URL string `json:"url"`
	}
	if err := decode(r, &x); err != nil {
		fail(w, err)
		return
	}
	if !strings.HasPrefix(x.URL, core.PublicSite+"/") {
		fail(w, fmt.Errorf("Nur offizielle Hyspayre-Links erlaubt"))
		return
	}
	if err := openExternal(x.URL); err != nil {
		fail(w, err)
		return
	}
	sendJSON(w, map[string]bool{"ok": true}, 200)
}
func (s *server) exit(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r, "POST") {
		return
	}
	sendJSON(w, map[string]bool{"ok": true}, 200)
	appClosing.Store(true)
	s.m.Exit()
}
func applyUpdate(original string) {
	if runtime.GOOS != "windows" || !strings.EqualFold(filepath.Base(original), "Hyspayre-Launcher.exe") {
		return
	}
	source, err := os.Executable()
	if err != nil {
		return
	}
	// Wait for the original process to release its executable.
	backup := original + ".previous"
	for i := 0; i < 120; i++ {
		if _, e := os.Stat(backup); e == nil {
			os.Remove(backup)
		}
		if e := os.Rename(original, backup); e == nil {
			in, e := os.Open(source)
			if e != nil {
				os.Rename(backup, original)
				return
			}
			out, e := os.OpenFile(original, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
			if e != nil {
				in.Close()
				os.Rename(backup, original)
				return
			}
			_, e = io.Copy(out, in)
			in.Close()
			out.Close()
			if e != nil {
				os.Remove(original)
				os.Rename(backup, original)
				return
			}
			cmd := exec.Command(original)
			cmd.Dir = filepath.Dir(original)
			if cmd.Start() == nil {
				cmd.Process.Release()
			}
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// Assert our embedded files exist at build time.
var _ fs.FS = UI
