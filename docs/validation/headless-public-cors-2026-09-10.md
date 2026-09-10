# Headless public CORS doğrulaması — 2026-09-10

İlgili karar: [ADR-0021](../architecture/0021-headless-browser-session-boundary.md). Bu kayıt credentialed browser session desteği kanıtı değildir; yalnız published read allowlist davranışını doğrular.

## Otomatik kanıt

- `go test ./internal/config ./internal/security ./internal/server ./cmd/hydro`
- `go test ./...`

`internal/security/cors_test.go` aşağıdaki sınırları kapsar:

- allowlisted origin, `/api/v1/news` gibi published read `GET` yanıtında exact `Access-Control-Allow-Origin` ve `Vary: Origin` alır;
- `Access-Control-Allow-Credentials` hiçbir zaman üretilmez;
- bilinmeyen origin, `POST` ve `OPTIONS` isteği allow header almaz;
- wildcard, path, user-info ve HTTP(S) olmayan config origin'leri reddedilir.

## Yerel seed smoke

Güncel binary `HYDRO_PUBLIC_API_ORIGINS=http://localhost:3000` ile seed edilmiş SQLite demo verisi üzerinde çalıştırıldı.

| İstek | Sonuç |
|---|---|
| `GET /api/v1/news?locale=tr`, `Origin: http://localhost:3000` | `200`; iki yayınlanmış haber; `Access-Control-Allow-Origin: http://localhost:3000`, `Vary: Origin`; credential header yok |
| `GET /api/v1/commerce/cart`, aynı origin | `401 unauthorized`; CORS allow/credential header yok |

Bu sınır Admin, auth, private media, webhooks, operations veya mutation endpoint'lerine CORS erişimi açmaz.
