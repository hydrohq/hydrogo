# S03 UI Stack, CSP and Media Upload Validation

**Date:** 2026-09-06  
**Roadmap item:** R1 / S03  
**Status:** Succeeded; local Quill/HTMX assets and CSP compliant editor fixture were verified.

## Validated Behaviors

1. **Templ and View Slot Contract:**
   - The `extension.Theme` contract (`RenderPage`, `RenderHeader`, `RenderFooter`) was defined.
   - Two different themes, `DefaultTheme` and `MinimalTheme`, were implemented in `examples/client-edition`.
   - Compiled as an external Go module without any dependency on Core packages (`internal/`).

2. **Rich Text & HTML Sanitizer:**
   - `HTMLSanitizer` was created based on the `bluemonday` UGC policy.
   - It was proven with unit tests (`internal/ui/sanitizer_test.go`) that `<script>`, `onload`, `onerror`, `javascript:` URIs, and embedded SVG/XSS attack vectors are cleaned; safe HTML tags are preserved.

3. **Strict CSP & Dynamic Nonce:**
   - A dynamic random base64 nonce is generated per request and safely passed to the inline `<style nonce="...">` and `<script nonce="...">` blocks of templates.

4. **Secure Media Upload Experiment:**
   - A secure upload pipeline was established with `internal/media.Uploader`:
      - The actual MIME type was verified (JPEG, PNG, WebP) via content sniffing (`http.DetectContentType`) over the first 512 bytes.
      - PHP/executable files and disallowed MIME types were rejected.
      - Visual integrity and dimensions were validated with standard Go `image.DecodeConfig`; corrupt images were rejected with temporary file cleanup.
      - Path traversal attacks were prevented; content-based SHA-256 file naming was implemented.
      - Maximum file size limits were tested.

5. **Quill/HTMX local asset and CSP fixture:**
   - Quill 2.0.3 (`static/vendor/quill`) and HTMX 2.0.10 (`static/vendor/htmx`) were pinned to the repo without requiring a CDN.
   - `internal/ui.RenderEditor` generates the Quill Delta JSON field, CSRF field, and HTMX form target with nonce-based inline initialization.
   - HTMX dynamic evaluation was disabled with `allowEval=false` and `allowScriptTags=false` settings; CDN URLs and inline event handlers are rejected in the test.

## Executed Commands

```text
go test -v ./internal/ui/...
go test -v ./internal/media/...
cd examples/client-edition && go build ./...
go run ./tools/importguard -root examples/client-edition
```
All steps passed without error. Versioned third-party licenses are located under `static/vendor/*/LICENSE`.
