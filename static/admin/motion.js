// HydroGo admin micro-interactions. Everything is delegated from document so
// it keeps working after the studio router swaps page content.
(() => {
  if (window.__hydroMotion) return;
  window.__hydroMotion = true;

  // Mark fields the browser considers invalid so the CSS error state shows,
  // and clear it as soon as the value becomes valid.
  document.addEventListener("invalid", (event) => {
    if (event.target instanceof HTMLElement) event.target.setAttribute("aria-invalid", "true");
  }, true);
  document.addEventListener("input", (event) => {
    const field = event.target;
    if (field instanceof HTMLInputElement || field instanceof HTMLTextAreaElement || field instanceof HTMLSelectElement) {
      if (field.getAttribute("aria-invalid") === "true" && field.checkValidity()) field.removeAttribute("aria-invalid");
    }
  });

  // Toasts from the server: respond with `HX-Trigger: {"hydro:toast": {"message": "Saved"}}`.
  document.addEventListener("hydro:toast", (event) => {
    const detail = event.detail || {};
    const message = typeof detail === "string" ? detail : detail.message || detail.value;
    if (message && typeof window.HydroToast === "function") window.HydroToast(message, detail.type || "success");
  });

  // Brief confirmation pulse on copy buttons.
  document.addEventListener("click", (event) => {
    const target = event.target instanceof Element ? event.target.closest("[data-copy-text], [data-copy-url]") : null;
    if (!target) return;
    target.removeAttribute("data-mi-flash");
    void target.offsetWidth;
    target.setAttribute("data-mi-flash", "");
  });
})();
