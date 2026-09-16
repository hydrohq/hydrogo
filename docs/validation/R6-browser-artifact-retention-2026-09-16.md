# Browser E2E CI artifact kanıtı (2026-09-16)

`e2e/package-lock.json` Playwright sürümünü 1.63.0 olarak kilitler. CI browser
job'ı `npm ci` ve `npx playwright install chromium` ile bu kilitli toolchain'i
kullanır. Test konfigürasyonu başarısız testlerde trace, screenshot ve video
üretir; job sonucu ne olursa olsun `actions/upload-artifact` ile
`e2e/artifacts`, `e2e/test-results` ve `e2e/playwright-report` yolları yüklenir.

Bu düzenleme yalnızca CI gözlemlenebilirliğini artırır; artifact retention ve
erişim süresi GitHub Actions repository politikasına tabidir.
