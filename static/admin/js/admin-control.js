// Management tools (/admin/control): the settings, redirect, media and invite
// forms post with fetch and report through toasts; a created invite opens the
// share dialog. Listeners are delegated from the document so they keep
// working after studio navigation swaps the page.
(() => {
  const csrf = () => decodeURIComponent((document.cookie.match(/(?:^|; )hydro_csrf=([^;]+)/) || [])[1] || "");
  const notify = (message, type = "info", title = "") => {
    if (typeof window.HydroToast === "function") window.HydroToast(message, type, title);
  };

  // send posts one form and toggles its submit button's loading state. The
  // response is returned only when it succeeded; failures throw.
  const send = async (form, url, init) => {
    const button = form.querySelector('button[type="submit"]');
    if (button) button.classList.add("is-loading");
    try {
      const response = await fetch(url, { method: "POST", ...init });
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "HTTP " + response.status);
      }
      return response;
    } finally {
      if (button) button.classList.remove("is-loading");
    }
  };

  const postJSON = async (form, url, data, successMessage) => {
    try {
      await send(form, url, {
        headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf() },
        body: JSON.stringify(data),
      });
      notify(successMessage, "success", "İşlem Başarılı");
      form.reset();
    } catch (_) {
      notify("İşlem sırasında bir hata oluştu.", "error", "Hata");
    }
  };

  const handlers = {
    "setting-form": (form) => {
      const fields = new FormData(form);
      postJSON(form, "/admin/site/settings", { key: fields.get("key"), value: fields.get("value"), public: true }, "Site ayarı başarıyla kaydedildi.");
    },
    "redirect-form": (form) => {
      const fields = new FormData(form);
      postJSON(
        form,
        "/admin/site/redirects",
        { source_path: fields.get("source_path"), target_path: fields.get("target_path"), status: Number(fields.get("status")) },
        "URL yönlendirme kuralı başarıyla kaydedildi.",
      );
    },
    "media-form": async (form) => {
      try {
        await send(form, "/api/media", { headers: { "X-CSRF-Token": csrf() }, body: new FormData(form) });
        notify("Medya başarıyla yüklendi ve işleme alındı.", "success", "Yükleme Başarılı");
        form.reset();
      } catch (_) {
        notify("Medya yüklenirken hata oluştu.", "error", "Yükleme Başarısız");
      }
    },
    "invite-form": async (form) => {
      try {
        const response = await send(form, "/admin/auth/invites", {
          headers: { "X-CSRF-Token": csrf() },
          body: new URLSearchParams(new FormData(form)),
        });
        const result = await response.json();
        const dialog = document.querySelector("[data-invite-share-dialog]");
        const message = document.querySelector("[data-invite-share-message]");
        if (dialog && message) {
          message.value = "Merhaba, Admin Studio davetiniz hazır.\nGiriş adresi: " + location.origin + "/admin\nDavet bağlantısı: " + result.url + "\nE-posta: " + result.email;
          dialog.showModal();
        }
        notify("Davet e-postası gönderildi. Paylaşım mesajı hazır.", "success", "Davet Gönderildi");
        form.reset();
      } catch (_) {
        notify("Davet gönderilemedi.", "error", "İşlem Başarısız");
      }
    },
  };

  document.addEventListener("submit", (event) => {
    const form = event.target;
    const handler = form instanceof HTMLFormElement && handlers[form.id];
    if (!handler || !form.closest(".control-container")) return;
    event.preventDefault();
    handler(form);
  });

  const clearShareMessage = (dialog) => {
    const message = dialog.querySelector("[data-invite-share-message]");
    if (message) message.value = "";
  };

  document.addEventListener("click", async (event) => {
    const dialog = event.target.closest("[data-invite-share-dialog]");
    if (!dialog) return;
    if (event.target.closest("[data-invite-share-close]")) {
      clearShareMessage(dialog);
      dialog.close();
      return;
    }
    if (!event.target.closest("[data-invite-share-copy]")) return;
    const text = dialog.querySelector("[data-invite-share-message]");
    const status = dialog.querySelector("[data-invite-share-status]");
    try {
      await navigator.clipboard.writeText(text.value);
      status.textContent = "Mesaj panoya kopyalandı.";
    } catch {
      text.focus();
      text.select();
      status.textContent = "Mesaj seçildi. Kopyalamak için Ctrl+C veya ⌘C kullanın.";
    }
  });

  // "close" does not bubble, so listen in the capture phase.
  document.addEventListener(
    "close",
    (event) => {
      if (event.target instanceof HTMLDialogElement && event.target.matches("[data-invite-share-dialog]")) clearShareMessage(event.target);
    },
    true,
  );
})();
