// Mirrors the title field of a split editor form (form[data-editor-live-title])
// into the editor header (.editor-live-title) while the user types.
(() => {
  const init = () => {
    const input = document.querySelector('form[data-editor-live-title] input[name="title"]');
    const title = document.querySelector(".editor-live-title");
    if (!input || !title || input.dataset.liveTitleBound) return;
    input.dataset.liveTitleBound = "true";
    input.addEventListener("input", () => {
      title.textContent = input.value.trim() || "[Başlıksız]";
    });
  };
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init, { once: true });
  else init();
})();
