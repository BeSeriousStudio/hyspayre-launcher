package core

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	ClientVersion          = "0.2.3"
	CatalogURL             = "https://hyspayre.com/launcher/catalog.json"
	PublicSite             = "https://hyspayre.com"
	MaxPackageSize   int64 = 10 << 30
	MaxExtractedSize int64 = 15 << 30
)

// Replaced at build with the public half of the publisher's Ed25519 key.
var PublicSigningKeyBase64 = "xpOR5HrV2HOEZcQoCI/lFg4lv/yPYYtzk5v8g8iN6gM="

var gameIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)
var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type LauncherRelease struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

type Game struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
	Genre       string `json:"genre"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	PackageURL  string `json:"package_url"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	Executable  string `json:"executable"`
	ImageURL    string `json:"image_url"`
	PatchNotes  string `json:"patch_notes"`
}

type CatalogPayload struct {
	Schema      int             `json:"schema"`
	PublishedAt string          `json:"published_at"`
	Launcher    LauncherRelease `json:"launcher"`
	Games       []Game          `json:"games"`
}

type Catalog struct {
	Schema      int             `json:"schema"`
	PublishedAt string          `json:"published_at"`
	Launcher    LauncherRelease `json:"launcher"`
	Games       []Game          `json:"games"`
	Signature   string          `json:"signature"`
}

func (c Catalog) Payload() CatalogPayload {
	return CatalogPayload{Schema: c.Schema, PublishedAt: c.PublishedAt, Launcher: c.Launcher, Games: c.Games}
}

func SignCatalog(c *Catalog, priv ed25519.PrivateKey) error {
	if err := c.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(c.Payload())
	if err != nil {
		return err
	}
	c.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, raw))
	return nil
}

func VerifyCatalog(raw []byte) (Catalog, error) {
	var c Catalog
	if len(raw) > 2*1024*1024 {
		return c, errors.New("Katalog ist zu groß")
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, err
	}
	key, err := base64.StdEncoding.DecodeString(PublicSigningKeyBase64)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return c, errors.New("Signatur-Schlüssel im Client ungültig")
	}
	sig, err := base64.StdEncoding.DecodeString(c.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return c, errors.New("Katalog-Signatur fehlt")
	}
	payload, err := json.Marshal(c.Payload())
	if err != nil {
		return c, err
	}
	if !ed25519.Verify(ed25519.PublicKey(key), payload, sig) {
		return c, errors.New("Katalog-Signatur nicht gültig – nichts wird installiert")
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

func ValidHTTPS(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	if u.Scheme != "https" || u.User != nil || u.Hostname() == "" || u.Fragment != "" {
		return false
	}
	// Restrict all installable code to the project-controlled host. No arbitrary URLs from a manifest.
	return strings.EqualFold(u.Hostname(), "hyspayre.com") && u.Port() == ""
}

func ValidExePath(exe string) bool {
	if !strings.HasSuffix(strings.ToLower(exe), ".exe") || exe == "" || strings.Contains(exe, "\\") || strings.HasPrefix(exe, "/") {
		return false
	}
	for _, part := range strings.Split(exe, "/") {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, ":<>\"|?*\\") {
			return false
		}
	}
	return true
}

func (c Catalog) Validate() error {
	if c.Schema != 1 {
		return errors.New("Katalogschema nicht unterstützt")
	}
	if len(c.Games) > 100 {
		return errors.New("Zu viele Spiele")
	}
	seen := map[string]bool{}
	for _, game := range c.Games {
		if !gameIDPattern.MatchString(game.ID) {
			return fmt.Errorf("Ungültige Spiele-ID %q", game.ID)
		}
		if seen[game.ID] {
			return fmt.Errorf("Doppelte Spiele-ID %q", game.ID)
		}
		seen[game.ID] = true
		if len(game.Title) > 110 || game.Title == "" || len(game.Description) > 1200 || len(game.PatchNotes) > 3000 {
			return fmt.Errorf("Ungültige Metadaten für %s", game.ID)
		}
		if game.Status != "upcoming" && game.Status != "released" {
			return fmt.Errorf("Ungültiger Status %s", game.Status)
		}
		if game.Status == "released" {
			if game.Version == "" || !ValidExePath(game.Executable) || !ValidHTTPS(game.PackageURL) || !sha256Pattern.MatchString(game.SHA256) || game.Size <= 0 || game.Size > MaxPackageSize {
				return fmt.Errorf("Unvollständiger Release für %s", game.ID)
			}
		}
		if game.ImageURL != "" && !ValidHTTPS(game.ImageURL) {
			return fmt.Errorf("Ungültige Bildadresse %s", game.ID)
		}
	}
	if c.Launcher.URL != "" && (!ValidHTTPS(c.Launcher.URL) || !sha256Pattern.MatchString(c.Launcher.SHA256) || c.Launcher.Size <= 0 || c.Launcher.Size > 300*1024*1024) {
		return errors.New("Launcher-Update unvollständig")
	}
	return nil
}

func Sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
