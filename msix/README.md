# Hyspayre Launcher – Microsoft Store & Windows App Installer

Stand: 21. November 2026 · Launcher `0.2.5` · MSIX `1.2.5.0` (x64)

## Was bereits umgesetzt ist

- Das bestehende `Hyspayre-Setup.exe` bleibt als direkter Website-Download verfügbar.
- `.github/workflows/build-msix.yml` kompiliert den Windows-Launcher und erstellt mit dem Windows-SDK-Tool `MakeAppx.exe` ein **unsigniertes** MSIX zur Microsoft-Store-Einreichung.
- `scripts/build_msix.py` erzeugt Manifest und Hyspayre-Diamantlogo-Bilder (`44`, `150`, `310`, Store-Logo) und integriert die Launcher-EXE in ein Windows-Desktop-App-Paket mit `runFullTrust`.
- `scripts/create_appinstaller.py` kann **nach** der Signierung der außerhalb des Stores vertriebenen MSIX ein `.appinstaller`-Update-Dokument erzeugen.
- Der Launcher erkennt Windows-App-Paketidentität automatisch. Im MSIX-Fall ist das alte EXE-Selbstupdate deaktiviert, da der MSIX-Installationsordner schreibgeschützt ist. Supabase und Spiel-Dateien bleiben unter LOCALAPPDATA.

## 1: Erstes MSIX für Microsoft Store bauen

1. In [Microsoft Partner Center](https://partner.microsoft.com/dashboard) ein Entwicklerkonto haben/erstellen, `Hyspayre Launcher` als App anlegen und den Namen reservieren.
2. In **Product management > App identity** die **Package/Identity Name** und den **Publisher** (vollständiger DN, meist `CN=...`) herauskopieren. Die Werte **nicht erraten**.
3. Im GitHub-Repository unter `Settings > Secrets and variables > Actions > Variables` zwei **Repository variables** anlegen:
   - `MSIX_IDENTITY_NAME` = Package/Identity Name (z. B. aus dem Partner Center, kein Demo-Wert)
   - `MSIX_PUBLISHER` = der vollständige Publisher-DN aus dem Partner Center
4. Den gelieferten Quellcode in das GitHub-Repository übernehmen und unter `Actions > Build Microsoft Store MSIX (unsigned) > Run workflow` den Build starten.
5. Unter dem fertigen Workflow `Artifacts` die Datei `hyspayre-025-unsigned-store-msix-and-classic-installer` herunterladen. Darin liegt `HyspayreLauncher_1.2.5.0_x64.msix`. Ohne Variablen ist es **nur ein Entwickler-Paket** und keine Store-Einreichung.
6. MSIX auf einem Windows-Testsystem prüfen und nach Microsofts Zertifizierungsanforderungen testen (Windows App Certification Kit). Im Partner Center unter `Packages` das MSIX hochladen, Metadaten/Datenschutz/Screenshots vervollständigen und die App zur Zertifizierung einreichen. Microsoft signiert **erst nach erfolgreicher Prüfung** für den Store.

**Warum MSIX `1.2.5.0`, wenn der Launcher `0.2.5` heißt?** Die Store-Manifest-Version muss vier Zahlen enthalten, und die erste Zahl darf nicht `0` sein; der Launcher verwendet weiterhin semantische Versionsnummern.

**Wichtig:** `MakeAppx.exe` läuft hier im Windows-GitHub-Runner. Die MSIX-Datei ist ein **Build-Ergebnis des Workflows**, kein bereits signiertes/hochladbares Website-Paket. Den tatsächlichen Windows-Installationsablauf und Store-Zertifizierung konnten wir hier nicht durchführen.

## 2: Direkter Website-Download ab sofort (EXE)

- `Hyspayre-Setup.exe` aus dem separaten IONOS-ZIP nach `/public/downloads/Hyspayre-Setup.exe` kopieren.
- `download.html`, `js/site.js`, `js/config.js` und `downloads/.htaccess` aus dem IONOS-ZIP mit Erhalt der Verzeichnisstruktur nach `/public/` kopieren.
- Auf `https://hyspayre.com/download.html` kann der Besucher sofort den klassischen Setup-Installer herunterladen.
- Der EXE-Installer ist derzeit **nicht Authenticode-signiert**. SmartScreen kann weiter warnen. Windows und Website-Download funktionieren unabhängig vom Microsoft Store.

## 3: Direktes MSIX über Windows App Installer anbieten – ERST NACH SIGNATUR

Ein über eine Website heruntergeladenes MSIX muss mit einem **auf Zielgeräten vertrauenswürdigen Code-Signing-Zertifikat** signiert sein. Die kostenlose Microsoft-Store-Signierung gilt **nur für die Store-Verteilung** und signiert nicht automatisch deinen eigenen Website-Download. Nicht signierte MSIX-Downloads dürfen nicht als normale Benutzerinstallation beworben werden.

1. Eine MSIX mit der **Identität des eigenen vertrauenswürdigen Signierzertifikats** bauen. `Identity Publisher` im Manifest und Subject des Zertifikats müssen übereinstimmen. Für eine normale Veröffentlichung kann dieselbe Identität wie im Store **nur** verwendet werden, wenn das Zertifikat dazu passt. Sonst sind Store- und Direktinstallationen unterschiedliche Paketidentitäten.
2. Die `.msix` mit `signtool.exe` und einem vertrauenswürdigen Zertifikat signieren. Nicht nur die einzelne `.exe` signieren.
3. **Nach der Signierung** `HyspayreLauncher_1.2.5.0_x64.msix` nach `/public/downloads/` hochladen und prüfen, dass der HTTPS-Download die korrekte Binärdatei liefert.
4. Eine `.appinstaller` erzeugen (Publisher/Identity müssen exakt zu der signierten MSIX passen):

   ```powershell
   python scripts/create_appinstaller.py --identity "ECHTE_PAKETIDENTITAET" --publisher "CN=ECHTER_ZERTIFIKATINHABER" --base-url "https://hyspayre.com/downloads/"
   ```

5. `dist/Hyspayre.appinstaller` als `/public/downloads/Hyspayre.appinstaller` hochladen und die HTTPS-Adresse prüfen. Die stabile URL wird auch für künftige Windows-App-Installer-Updates verwendet.
6. `js/config.js` bearbeiten und erst **dann** setzen:

   ```js
   "msixAvailable": true,
   "msixAppInstallerUrl": "downloads/Hyspayre.appinstaller",
   ```

7. `download.html` bietet damit zusätzlich eine aktive App-Installer-Downloadschaltfläche. Doppelklick auf die heruntergeladene `.appinstaller` startet Windows App Installer. Der `ms-appinstaller:`-URL-Protocol-Handler ist auf vielen Windows-Installationen seit 2023 deaktiviert, daher wird er hier **nicht** verwendet.

**Vorsicht:** EXE-Installer und MSIX installieren zwei unterschiedliche Varianten. Nicht beide gleichzeitig installieren; unter Umständen entstehen doppelte Startmenüeinträge. Eine Store-Veröffentlichung ist keine automatische Veröffentlichung des signierten MSIX auf deiner Website.

## 4: Microsoft-Store-Button freischalten

Sobald Microsoft die App veröffentlicht hat, den echten Store-Link (von der veröffentlichten Microsoft-Store-Produktseite) in `js/config.js` bei `microsoftStoreUrl` hinterlegen. Bis dahin bleibt der Button deaktiviert; es wird **kein fiktiver Store-Eintrag** angezeigt.

## 5: Publisher-Katalog / Updates

- Der EXE-Launcher hat weiterhin seinen eigenen Ed25519-signierten Katalog (`/public/launcher/catalog.json`), und dieser bleibt unabhängig von Microsoft Store/MSIX.
- Für Release `0.2.5`: **nur** den neuen, final ggf. Authenticode-signierten `Hyspayre-Launcher.exe` hash-en; Katalog mit dem *bisherigen privaten Hyspayre-Ed25519-Schlüssel* signieren. Den privaten Schlüssel niemals auf die Website oder ins öffentliche GitHub-Repository hochladen.
- Für MSIX/Store verwaltet Windows bzw. der Store das Launcher-Update. Spiele-Updates sind davon getrennt und bleiben im Launcher.

## 6: Supabase

Keine Änderungen nötig. Login, Profile, Skins, Ränge, Credits und Freunde bleiben über die bisherige Supabase-Anbindung zugänglich. Auf einem Windows-Testgerät nach Installation der MSIX Anmeldung, Web/Edge-Fenster, Spieleinstallationen und Launcher-Beenden testen.

Offizielle Quellen:
- https://learn.microsoft.com/en-us/windows/msix/desktop/desktop-to-uwp-manual-conversion
- https://learn.microsoft.com/en-us/windows/apps/publish/get-started
- https://learn.microsoft.com/en-us/windows/msix/app-installer/app-installer-file-overview
