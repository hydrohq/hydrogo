# R1 Toolchain, Çapraz Derleme ve Yedekleme Araçları Doğrulaması

**Tarih:** 2026-09-06  
**Roadmap maddesi:** R1 Toolchain ve Taşınabilirlik  
**Durum:** Kısmi; çapraz derleme ve encoder seçimi doğrulandı, gerçek backup/restore çalıştırması ayrıca gerekiyor.

## Doğrulanan Davranışlar

1. **Çapraz Derleme (Cross-Platform Compilation):**
   - Proje tamamen saf Go (CGO-free) bağımlılıklarla yapılandırıldı:
     - SQLite: `modernc.org/sqlite`
     - PostgreSQL: `github.com/jackc/pgx/v5`
     - Kriptografi & Parola: `golang.org/x/crypto/argon2`
     - Görsel: Go standart kütüphanesi (`image/png`, `image/jpeg`, `image/gif`) + `golang.org/x/image/webp`
   - Hem Windows amd64 (`hydro.exe`) hem de Linux amd64 (`bin/hydro-linux-amd64`) hedefleri hiçbir dış C kütüphanesine ihtiyaç duymadan başarıyla derlendi:
     ```powershell
     $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/hydro-linux-amd64 ./cmd/hydro
     $env:GOOS="windows"; $env:GOARCH="amd64"; go build -o bin/hydro.exe ./cmd/hydro
     ```

2. **Yedekleme ve Kurtarma Araçları Değerlendirmesi:**
   - **SQLite (Compact Profil):** CGO gerektirmeyen `VACUUM INTO` ve SQLite online backup mantığı veya WAL checkpoint sonrası anlık dosya kopyalama ile bağımsız işletim doğrulanabilir.
   - **PostgreSQL (Standard/Distributed):** Standart `pg_dump` / `pg_restore` CLI araçları ve S09 mantıksal veri aktarım sözleşmeleriyle uyumludur.

3. **Görsel İşleme / Encoder Kararı:**
   - CGO ve `libvips` gibi dış C bağımlılıkları tek ikili (single-binary) dağıtımı ve taşınabilirlik ilkesini (Manifesto Madde 2 & 5) zedelediği için saf Go kütüphaneleri seçildi.
