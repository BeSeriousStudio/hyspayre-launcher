// Package social connects the desktop client to the SAME authenticated Supabase
// RPCs used by the Hyspayre website. No privileged service-role key is present.
package social

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	BaseURL = "https://amgpdukqacxogafqpmho.supabase.co"
	// This is the PUBLIC Supabase publishable key also shipped in the website.
	PublishableKey = "sb_publishable_oHFuEqSBxUkj82I_MZLzkw_2nJjyjyF"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type tokens struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
	Expires int64  `json:"expires_at"`
}

type Client struct {
	mu        sync.Mutex
	base      string
	publicKey string
	http      *http.Client
	tokenFile string
	session   tokens
}

func New(dataDir string) *Client {
	return NewCustom(BaseURL, PublishableKey, dataDir, nil)
}

// NewCustom supports offline tests with a fake Supabase API server.
func NewCustom(base, key, dataDir string, client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("Redirect von Supabase nicht zugelassen")
		}}
	}
	c := &Client{base: strings.TrimRight(base, "/"), publicKey: key, http: client, tokenFile: filepath.Join(dataDir, "session.dpapi")}
	if raw, e := os.ReadFile(c.tokenFile); e == nil && len(raw) > 0 {
		if decrypted, e := unprotect(raw); e == nil {
			var t tokens
			if json.Unmarshal(decrypted, &t) == nil && t.Refresh != "" {
				c.session = t
			}
		}
	}
	return c
}

type authResult struct {
	Access    string `json:"access_token"`
	Refresh   string `json:"refresh_token"`
	ExpiresIn int64  `json:"expires_in"`
}

func (c *Client) send(endpoint string, body any, access string) (json.RawMessage, error) {
	var rdr io.Reader
	if body != nil {
		raw, e := json.Marshal(body)
		if e != nil {
			return nil, e
		}
		rdr = bytes.NewReader(raw)
	}
	req, e := http.NewRequest(http.MethodPost, c.base+endpoint, rdr)
	if e != nil {
		return nil, e
	}
	req.Header.Set("apikey", c.publicKey)
	req.Header.Set("Content-Type", "application/json")
	if access != "" {
		req.Header.Set("Authorization", "Bearer "+access)
	}
	res, e := c.http.Do(req)
	if e != nil {
		return nil, fmt.Errorf("Supabase nicht erreichbar: %w", e)
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, 512*1024+1))
	if e != nil {
		return nil, e
	}
	if len(raw) > 512*1024 {
		return nil, errors.New("Supabase-Antwort zu groß")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var m struct {
			Message          string `json:"message"`
			Code             string `json:"code"`
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			Msg              string `json:"msg"`
		}
		_ = json.Unmarshal(raw, &m)
		message := m.ErrorDescription
		if message == "" {
			message = m.Message
		}
		if message == "" {
			message = m.Msg
		}
		if message == "" {
			message = m.Error
		}
		if message == "" {
			message = fmt.Sprintf("Supabase HTTP %d", res.StatusCode)
		}
		return nil, &APIError{Code: res.StatusCode, Message: message, PostgrestCode: m.Code}
	}
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if !json.Valid(raw) {
		return nil, errors.New("Ungültige Supabase-Antwort")
	}
	return raw, nil
}

type APIError struct {
	Code          int
	Message       string
	PostgrestCode string
}

func (a *APIError) Error() string { return a.Message }

func (c *Client) assignSessionLocked(s authResult, persist bool) error {
	if s.Access == "" || s.Refresh == "" {
		return errors.New("Ungültige Anmeldung")
	}
	if s.ExpiresIn < 1 {
		s.ExpiresIn = 3600
	}
	c.session = tokens{Access: s.Access, Refresh: s.Refresh, Expires: time.Now().Unix() + s.ExpiresIn}
	if !persist {
		_ = os.Remove(c.tokenFile)
		return nil
	}
	raw, _ := json.Marshal(c.session)
	protected, e := protect(raw)
	if e != nil {
		return fmt.Errorf("Sitzung kann nicht sicher gespeichert werden: %w", e)
	}
	if e = os.MkdirAll(filepath.Dir(c.tokenFile), 0700); e != nil {
		return e
	}
	temp := c.tokenFile + ".tmp"
	if e = os.WriteFile(temp, protected, 0600); e != nil {
		return e
	}
	if e = os.Rename(temp, c.tokenFile); e != nil {
		_ = os.Remove(temp)
		return e
	}
	return nil
}

func (c *Client) Login(email, password string, remember bool) error {
	email = strings.TrimSpace(email)
	if len(email) < 5 || len(email) > 254 || !strings.Contains(email, "@") || len(password) < 1 || len(password) > 512 {
		return errors.New("Bitte gültige E-Mail-Adresse und Passwort eingeben")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	raw, e := c.send("/auth/v1/token?grant_type=password", map[string]string{"email": email, "password": password}, "")
	if e != nil {
		return e
	}
	var r authResult
	if json.Unmarshal(raw, &r) != nil {
		return errors.New("Ungültige Login-Antwort")
	}
	return c.assignSessionLocked(r, remember)
}

func (c *Client) Logout() {
	c.mu.Lock()
	defer c.mu.Unlock()
	// The remote Supabase session is short-lived. Revoke refresh credentials as well.
	if c.session.Access != "" {
		_, _ = c.send("/auth/v1/logout", nil, c.session.Access)
	}
	c.session = tokens{}
	_ = os.Remove(c.tokenFile)
}

func (c *Client) refreshLocked() error {
	if c.session.Refresh == "" {
		return errors.New("Bitte melde dich an")
	}
	previous := c.session
	raw, e := c.send("/auth/v1/token?grant_type=refresh_token", map[string]string{"refresh_token": previous.Refresh}, "")
	if e != nil {
		c.session = tokens{}
		_ = os.Remove(c.tokenFile)
		return errors.New("Anmeldung abgelaufen. Bitte erneut anmelden")
	}
	var next authResult
	if json.Unmarshal(raw, &next) != nil {
		return errors.New("Sitzung kann nicht erneuert werden")
	}
	// If a DPAPI file exists, the session was remembered when login was performed.
	_, statErr := os.Stat(c.tokenFile)
	return c.assignSessionLocked(next, statErr == nil)
}

func (c *Client) rpc(name string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session.Access == "" {
		return nil, errors.New("Bitte melde dich an")
	}
	if c.session.Expires <= time.Now().Unix()+60 {
		if e := c.refreshLocked(); e != nil {
			return nil, e
		}
	}
	result, e := c.send("/rest/v1/rpc/"+name, params, c.session.Access)
	var apiErr *APIError
	if errors.As(e, &apiErr) && apiErr.Code == 401 {
		if e = c.refreshLocked(); e != nil {
			return nil, e
		}
		return c.send("/rest/v1/rpc/"+name, params, c.session.Access)
	}
	return result, e
}

func (c *Client) MyProfile() (json.RawMessage, error) { return c.rpc("bss_me", map[string]any{}) }
func (c *Client) Friends() (json.RawMessage, error) {
	return c.rpc("bss_social_friends", map[string]any{})
}
func (c *Client) Search(term string, offset, limit int) (json.RawMessage, error) {
	term = strings.TrimSpace(term)
	if utf8.RuneCountInString(term) < 1 || utf8.RuneCountInString(term) > 30 {
		return nil, errors.New("Suchbegriff: 1 bis 30 Zeichen")
	}
	if offset < 0 || offset > 10000 {
		return nil, errors.New("Ungültige Suchposition")
	}
	if limit < 1 || limit > 100 {
		return nil, errors.New("Ungültige Ergebnisanzahl")
	}
	// SQL 08 supports one-character searches, partial matches and pagination.
	live, err := c.rpc("bss_social_search_live", map[string]any{"p_query": term, "p_offset": offset, "p_limit": limit})
	if err == nil {
		return live, nil
	}
	var apiErr *APIError
	// Fallback ONLY for a missing RPC signature in the REST API (PGRST202).
	// Other errors, including auth/RLS failures, must never be silently masked.
	if !errors.As(err, &apiErr) || apiErr.PostgrestCode != "PGRST202" {
		return nil, err
	}
	if utf8.RuneCountInString(term) < 2 {
		return json.Marshal(map[string]any{
			"players": []any{}, "total": 0, "has_more": false, "legacy": true,
			"notice": "Für die Suche ab einem Zeichen bitte Supabase SQL 08 ausführen. Bis dahin mindestens zwei Zeichen eingeben.",
		})
	}
	if offset >= 20 {
		return json.Marshal(map[string]any{
			"players": []any{}, "total": 0, "has_more": false, "legacy": true,
			"notice": "Die ältere Suche zeigt maximal 20 Treffer. Für weitere Spieler bitte Supabase SQL 08 ausführen.",
		})
	}
	// SQL 07 supports substring lookup, but returns at most 20 matches without pagination.
	old, legacyErr := c.rpc("bss_social_search", map[string]any{"p_query": term})
	if legacyErr != nil {
		var oldAPI *APIError
		if errors.As(legacyErr, &oldAPI) && oldAPI.PostgrestCode == "PGRST202" {
			return nil, errors.New("Spielersuche in Supabase fehlt. Bitte zuerst SQL 07 und danach SQL 08 im Supabase SQL Editor ausführen")
		}
		return nil, legacyErr
	}
	var oldResult struct {
		Players []json.RawMessage `json:"players"`
	}
	if err := json.Unmarshal(old, &oldResult); err != nil {
		return nil, errors.New("Unerwartete Antwort der älteren Spielersuche")
	}
	if oldResult.Players == nil {
		oldResult.Players = []json.RawMessage{}
	}
	players := oldResult.Players
	// Legacy RPC has a 20-result cap, not a real total count.
	if offset < len(players) {
		players = players[offset:]
	} else {
		players = []json.RawMessage{}
	}
	if len(players) > limit {
		players = players[:limit]
	}
	return json.Marshal(map[string]any{
		"players": players, "total": len(players), "has_more": false, "legacy": true,
		"notice": "Kompatibilitätsmodus: Die alte Supabase-Suche zeigt höchstens 20 Spieler. SQL 08 aktiviert vollständige Live-Suche.",
	})
}
func (c *Client) Profile(id string) (json.RawMessage, error) {
	if !uuidPattern.MatchString(id) {
		return nil, errors.New("Ungültige Spieler-ID")
	}
	return c.rpc("bss_social_profile", map[string]string{"p_user": id})
}
func (c *Client) Action(id, action string) (json.RawMessage, error) {
	if !uuidPattern.MatchString(id) {
		return nil, errors.New("Ungültige Spieler-ID")
	}
	switch action {
	case "request", "accept", "decline", "cancel", "remove":
	default:
		return nil, errors.New("Ungültige Freundschaftsaktion")
	}
	return c.rpc("bss_social_action", map[string]string{"p_user": id, "p_action": action})
}
