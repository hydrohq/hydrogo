// Copies an AI suggestion (designsystem.AIAssistantSuggestion) into the draft
// editor on the same page. It never submits or saves; it only records that the
// suggestion was applied at the panel's audit endpoint.
(() => {
  if (window.__hydroAIAssistantBound) return;
  window.__hydroAIAssistantBound = true;
  document.addEventListener("click", (event) => {
    const button = event.target.closest("[data-ai-apply]");
    if (!button) return;
    const panel = button.closest("[data-ai-suggestion]");
    const title = document.querySelector('input[name="title"]');
    const body = document.querySelector("[data-richtext] .richtext-value");
    const editor = document.querySelector("[data-richtext] .richtext-editor");
    const nextTitle = panel.querySelector("[data-ai-title]").textContent;
    const nextBody = panel.querySelector("[data-ai-body]").textContent;
    if (title && nextTitle) {
      title.value = nextTitle;
      title.dispatchEvent(new Event("input", { bubbles: true }));
    }
    if (editor && editor.__quill) {
      editor.__quill.setText(nextBody);
    } else if (body) {
      body.value = nextBody;
      body.dispatchEvent(new Event("input", { bubbles: true }));
    }
    button.textContent = "Taslak düzenleyici güncellendi";
    fetch(button.dataset.auditAction, { method: "POST", headers: { "X-CSRF-Token": button.dataset.csrf } })
      .then((response) => {
        if (!response.ok) throw new Error();
      })
      .catch(() => {
        panel.querySelector("[data-ai-apply-status]").textContent = "Taslak güncellendi ancak işlem denetim kaydı tamamlanamadı.";
      });
  });
})();
