# R5 — SDK broker grant istemcisi

`internal/sdkdist.RequestGrant`, broker sözleşmesinin istemci tarafını sağlar:
bounded JSON request/response, HTTPS/localhost test policy, Authorization header,
redirect reddi ve non-200 hata eşlemesi. Access token response'a veya diske
yazılmaz; çağıran `ValidateGrant` ile lock bağlamı eşleşmesini ayrıca kontrol eder.
`RequestValidatedGrant` ise transport ve lock/grant doğrulamasını tek fail-closed
çağrıda birleştirir; yanlış project/Core/SDK eşleşmesini reddeder.

Windows:

```text
go test ./internal/sdkdist
PASS
```

WSL Ubuntu (`arda`):

```text
go test ./internal/sdkdist ./internal/release ./cmd/hydro
exit code 0
```

Testler `httptest` ile başarılı grant, 403 ve oversized response senaryolarını
çalıştırır. Gerçek OAuth client registration ve broker deployment kapsam dışıdır.
WSL çalıştırması ayrıca `RequestValidatedGrant` lock mismatch regresyon testini
de kapsar.
