# Orbit – Sicherheitsarchitektur & Richtlinien

Die Sicherheit eines Docker-Management- und Monitoring-Systems hat höchste Priorität, da der Zugriff auf den Docker-Socket (`/var/run/docker.sock`) auf dem Host praktisch Root-Rechten gleichkommen kann.

Orbit implementiert daher ein mehrschichtiges Sicherheitskonzept (Defense-in-Depth).

---

## 1. Wichtiger Sicherheitshinweis zum Docker-Socket

> [!WARNING]
> Der Mount `/var/run/docker.sock:/var/run/docker.sock:ro` mit dem `:ro`-Flag schützt **nicht** vor mutierenden Docker-API-Aufrufen!
> 
> Das `:ro`-Flag auf Dateisystem-Ebene verhindert lediglich das Ändern des Socket-Dateiknotens selbst, schränkt aber die über den Socket gesendeten HTTP/REST-Befehle an die Docker Engine **nicht** ein. Ein über einen read-only Socket abgesetzter `POST /containers/create` wird vom Docker Engine Daemon wie gewohnt ausgeführt.

### Orbits Sicherheitsmaßnahmen:
1. **Kein generischer Docker-Proxy**: Orbit schleift niemals rohe Docker-API-Anfragen durch. Es gibt keinen ungeschützten Proxy-Endpunkt.
2. **Strikte Gateway-Allowlist**: Sämtliche Docker-Aufrufe erfolgen ausschließlich über ein internes Go-Interface (`Gateway`), das nur fest vorgegebene Operationen kennt.
3. **Zweistufige Feature-Gates**:
   - `ORBIT_ENABLE_ACTIONS=false` (Standard): Alle Verwaltungsaufrufe (`start`, `stop`, `restart`, `pause`) werden im Backend abgewiesen (HTTP 403 Forbidden).
   - `ORBIT_ENABLE_DESTRUCTIVE=false` (Standard): Löschen und Bereinigen (`remove`, `prune`) ist zusätzlich gesperrt.
4. **Schutz bestehender Container**:
   - Orbit verändert, löscht oder stoppt niemals eigenmächtig Container.
   - Volumes, die aktuell von einem Container genutzt werden, können nicht versehentlich gelöscht werden.

---

## 2. Authentifizierung & Passwörter

- **Keine Standardpasswörter**: Ein System ohne vorkonfigurierte Passwörter verhindert automatisierte Übernahmen.
- **Kryptografischer Setup-Token**: Der erste Admin-Account kann nur mit dem einmaligen, 32-stelligen Token aus den Server-Logs erstellt werden.
- **Argon2id**: Passwörter werden mit Argon2id (64 MB Speicher, 3 Iterationen, 4 Threads) gehasht.
- **Session-Sicherheit**:
  - Zufällige 256-Bit-Session-Tokens.
  - Speicherung als Hash in SQLite.
  - Cookies mit `HttpOnly`, `SameSite=Strict`, und `Secure` bei HTTPS.
  - Konfigurierbare Ablauf- und Inaktivitäts-Timeouts.

---

## 3. Schutz vor Web-Angriffen (CSRF, Brute-Force, CORS)

- **CSRF-Schutz**: Für alle zustandsändernden HTTP-Methoden (`POST`, `PUT`, `DELETE`, `PATCH`) ist ein synchronisiertes CSRF-Token im `X-CSRF-Token`-Header zwingend erforderlich.
- **Rate-Limiting**: Fehlgeschlagene Anmeldeversuche werden pro IP-Adresse gedrosselt (max. 5 Fehlversuche pro Minute).
- **Origin-Prüfung**: WebSocket-Upgrades prüfen die `Origin`-Header gegen `ORBIT_TRUSTED_ORIGINS`.
- **Sicherheits-Header**:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Permissions-Policy: camera=(), microphone=(), geolocation=()`

---

## 4. Berechtigungen & Least-Privilege

- Orbit läuft standardmäßig als unprivilegierter Non-Root-Benutzer (`UID 10001: GID 10001`).
- `no-new-privileges: true` verhindert Privilege Escalation.
- Alle Host-Mounts (`/host/proc`, `/host/sys`, `/host/etc/os-release`) sind strikt `read-only` (`:ro`).
- Sensitive Daten wie Environment-Variablen im Inspect-Tab sind für Nicht-Administratoren unkenntlich gemacht.
- Vollständiges Audit-Logging (`audit_log`): Jede administrative Aktion wird mit Zeitstempel, Benutzer, IP und Ergebnis protokolliert.

