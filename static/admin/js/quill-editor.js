// Quill/HTMX editor fixture (internal/ui/editor.templ): mounts Quill on
// #editor-body and keeps the hidden Delta field in sync on submit.
document.addEventListener("DOMContentLoaded", () => {
  htmx.config.allowEval = false;
  htmx.config.allowScriptTags = false;
  const deltaField = document.getElementById("editor-delta");
  const form = document.querySelector(".hydro-editor");
  if (!deltaField || !form) return;
  const quill = new Quill("#editor-body", { theme: "snow", modules: { toolbar: "#editor-toolbar" } });
  if (deltaField.value) {
    try {
      quill.setContents(JSON.parse(deltaField.value));
    } catch (_) {
      deltaField.value = JSON.stringify(quill.getContents());
    }
  }
  form.addEventListener("submit", () => {
    deltaField.value = JSON.stringify(quill.getContents());
  });
});
