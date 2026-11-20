//go:build !windows

package social

import "errors"

// On non-Windows platforms, persistent login is disabled until an OS keychain
// backend is implemented. Never store OAuth tokens unencrypted as a fallback.
func protect([]byte) ([]byte, error) {
	return nil, errors.New("Sichere Sitzungsablage ist derzeit nur unter Windows verfügbar. Ohne 'Angemeldet bleiben' anmelden")
}
func unprotect([]byte) ([]byte, error) { return nil, errors.New("Keine sichere Sitzungsablage") }
