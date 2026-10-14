// Shows the other editors working on the same record. Hydrates every
// [data-edit-presence] box rendered by designsystem.EditPresence: it sends a
// heartbeat, lists the active editors and repeats every 15 seconds.
(() => {
  const start = (box) => {
    if (box.dataset.editPresenceBound) return;
    box.dataset.editPresenceBound = "true";
    const endpoint = box.dataset.endpoint;
    const csrf = box.dataset.csrf;
    const list = box.querySelector("[data-edit-presence-list]");
    let stopped = false;
    const refresh = async () => {
      if (stopped) return;
      try {
        await fetch(endpoint, { method: "POST", credentials: "same-origin", headers: { "X-CSRF-Token": csrf } });
        const response = await fetch(endpoint, { credentials: "same-origin", headers: { Accept: "application/json" } });
        if (!response.ok) return;
        const editors = await response.json();
        list.replaceChildren();
        for (const editor of editors) {
          const item = document.createElement("li");
          item.textContent = editor.name + " (son etkinlik " + editor.active_seconds + " sn önce)";
          list.append(item);
        }
        box.hidden = editors.length === 0;
      } catch (_) {
        // Presence is advisory; a failed refresh keeps the last state.
      }
    };
    refresh();
    const timer = setInterval(refresh, 15000);
    window.addEventListener("pagehide", () => { stopped = true; clearInterval(timer); }, { once: true });
  };
  const init = () => document.querySelectorAll("[data-edit-presence]").forEach(start);
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init, { once: true });
  else init();
})();
