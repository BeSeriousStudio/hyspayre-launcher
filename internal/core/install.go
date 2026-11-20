package core

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ProgressFunc func(done, total int64)

func DownloadVerified(link, expectedHash string, expectedSize int64, target string, progress ProgressFunc) error {
	if !ValidHTTPS(link) {
		return errors.New("Download-Adresse ist nicht vertrauenswürdig")
	}
	if !sha256Pattern.MatchString(expectedHash) {
		return errors.New("Ungültige SHA-256-Prüfsumme")
	}
	if expectedSize < 1 || expectedSize > MaxPackageSize {
		return errors.New("Unzulässige Dateigröße")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	partial := target + ".part"
	// Resume from an existing partial only when the response honors Range.
	var current int64
	if fi, err := os.Stat(partial); err == nil {
		current = fi.Size()
	}
	if current >= expectedSize || current < 0 {
		os.Remove(partial)
		current = 0
	}
	client := &http.Client{Timeout: 0, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !ValidHTTPS(req.URL.String()) {
			return errors.New("Nicht erlaubte Download-Umleitung")
		}
		return nil
	}, Transport: &http.Transport{TLSHandshakeTimeout: 12 * time.Second, ResponseHeaderTimeout: 25 * time.Second, IdleConnTimeout: 20 * time.Second}}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return err
	}
	if current > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", current))
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return fmt.Errorf("Download HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode == 206 {
		if current == 0 || !strings.HasPrefix(resp.Header.Get("Content-Range"), fmt.Sprintf("bytes %d-", current)) {
			return errors.New("Ungültige Range-Antwort")
		}
	} else {
		current = 0
	}
	flags := os.O_WRONLY | os.O_CREATE
	if current > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	out, err := os.OpenFile(partial, flags, 0600)
	if err != nil {
		return err
	}
	done := current
	progress(done, expectedSize)
	buf := make([]byte, 256*1024)
	for {
		n, e := resp.Body.Read(buf)
		if n > 0 {
			done += int64(n)
			if done > expectedSize {
				out.Close()
				os.Remove(partial)
				return errors.New("Download größer als erwartet")
			}
			if _, we := out.Write(buf[:n]); we != nil {
				out.Close()
				return we
			}
			progress(done, expectedSize)
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			out.Close()
			return e
		}
	}
	if err = out.Close(); err != nil {
		return err
	}
	if done != expectedSize {
		return fmt.Errorf("Unvollständiger Download (%d/%d Bytes)", done, expectedSize)
	}
	file, err := os.Open(partial)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(h, file)
	file.Close()
	if err != nil {
		return err
	}
	if hex.EncodeToString(h.Sum(nil)) != expectedHash {
		os.Remove(partial)
		return errors.New("SHA-256-Prüfsumme stimmt nicht – Download verworfen")
	}
	os.Remove(target)
	return os.Rename(partial, target)
}

func windowsReserved(part string) bool {
	root := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
	if root == "CON" || root == "PRN" || root == "AUX" || root == "NUL" {
		return true
	}
	if len(root) == 4 && (strings.HasPrefix(root, "COM") || strings.HasPrefix(root, "LPT")) && root[3] >= '1' && root[3] <= '9' {
		return true
	}
	return false
}

// ExtractArchive refuses absolute paths, traversal, symlinks, NTFS ADS, zip bombs and special names.
func ExtractArchive(zipFile, destination string) error {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) > 15000 {
		return errors.New("Zu viele Dateien im Spielpaket")
	}
	if err = os.MkdirAll(destination, 0700); err != nil {
		return err
	}
	var total uint64
	for _, f := range reader.File {
		name := f.Name
		if strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || strings.IndexByte(name, 0) >= 0 {
			return fmt.Errorf("Unsicherer Pfad: %q", name)
		}
		parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
		for _, p := range parts {
			if p == "" || p == "." || p == ".." || strings.ContainsAny(p, ":<>\"|?*") || strings.HasSuffix(p, ".") || strings.HasSuffix(p, " ") || windowsReserved(p) {
				return fmt.Errorf("Unzulässiger Dateiname %q", name)
			}
		}
		if f.Mode()&os.ModeSymlink != 0 || f.Mode()&os.ModeType != 0 && !f.FileInfo().IsDir() {
			return errors.New("Verknüpfungen und Sonderdateien sind nicht erlaubt")
		}
		total += f.UncompressedSize64
		if total > uint64(MaxExtractedSize) {
			return errors.New("Archiv ist entpackt zu groß")
		}
		target := filepath.Join(append([]string{destination}, parts...)...)
		absroot, _ := filepath.Abs(destination)
		abstarget, _ := filepath.Abs(target)
		rel, err := filepath.Rel(absroot, abstarget)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return errors.New("Pfad verlässt Installationsordner")
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		if err != nil {
			src.Close()
			return err
		}
		copied, copyerr := io.CopyN(dst, src, int64(f.UncompressedSize64)+1)
		closeErr := dst.Close()
		src.Close()
		if copyerr != io.EOF {
			if copyerr == nil {
				return errors.New("Archivgröße stimmt nicht")
			}
			return copyerr
		}
		if copied != int64(f.UncompressedSize64) {
			return errors.New("Archivdatei hat falsche Länge")
		}
		if closeErr != nil {
			return closeErr
		}
		if runtime.GOOS != "windows" {
			os.Chmod(target, 0644)
		}
	}
	return nil
}

func SafeInstallDir(base, id string) (string, error) {
	if !gameIDPattern.MatchString(id) {
		return "", errors.New("Ungültige Spiele-ID")
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("Installationspfad muss absolut sein")
	}
	return filepath.Join(base, id), nil
}

func InstallArchive(base, id, archive, exe string) (string, error) {
	if !ValidExePath(exe) {
		return "", errors.New("Ungültiger Startpfad")
	}
	target, err := SafeInstallDir(base, id)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}
	stage := target + ".installing"
	backup := target + ".backup"
	os.RemoveAll(stage)
	if err := ExtractArchive(archive, stage); err != nil {
		os.RemoveAll(stage)
		return "", err
	}
	binary := filepath.Join(stage, filepath.FromSlash(exe))
	info, err := os.Stat(binary)
	if err != nil || info.IsDir() {
		os.RemoveAll(stage)
		return "", fmt.Errorf("Startdatei %s fehlt im Archiv", exe)
	}
	if _, err := os.Stat(backup); err == nil {
		os.RemoveAll(stage)
		return "", errors.New("Altes Backup vorhanden – manuell prüfen")
	}
	hadOld := false
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			os.RemoveAll(stage)
			return "", errors.New("Spiel ist möglicherweise noch geöffnet; bitte schließen und erneut versuchen")
		}
		hadOld = true
	}
	if err := os.Rename(stage, target); err != nil {
		if hadOld {
			os.Rename(backup, target)
		}
		os.RemoveAll(stage)
		return "", err
	}
	if hadOld {
		os.RemoveAll(backup)
	}
	return filepath.Join(target, filepath.FromSlash(exe)), nil
}
