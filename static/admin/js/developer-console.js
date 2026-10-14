// Developer console (/admin/developer): "live test" buttons fetch an endpoint
// and show the pretty-printed response in the card's preview box. Delegated
// from the document so it keeps working after studio navigation swaps pages.
document.addEventListener("click", (event) => {
  const button = event.target.closest("[data-test-endpoint]");
  if (!button) return;
  const box = button.closest(".dev-endpoint-card")?.querySelector(".dev-preview-box");
  if (!box) return;
  button.disabled = true;
  button.textContent = "Yükleniyor…";
  fetch(button.dataset.testEndpoint, { headers: { Accept: "application/json" } })
    .then((response) => response.text())
    .then((text) => {
      try {
        text = JSON.stringify(JSON.parse(text), null, 2);
      } catch (_) {}
      box.textContent = text;
      box.style.display = "block";
      button.disabled = false;
      button.textContent = "✓ Yanıt Alındı";
    })
    .catch(() => {
      box.textContent = "Hata: İstek tamamlanamadı.";
      box.style.display = "block";
      button.disabled = false;
      button.textContent = "Tekrar Dene";
    });
});
