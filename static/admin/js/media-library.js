// Media library (/admin/media): copy buttons, client-side search, the upload
// dropzone and the settings dialog. Uploads themselves are handled by
// /admin/media/static/uploader.js. Studio re-runs this file after in-page
// navigation, so document-level handlers are installed only once and page
// elements are bound once each.
(() => {
  const flash = (element, className, restore) => {
    element.classList.add(className);
    setTimeout(() => {
      element.classList.remove(className);
      restore?.();
    }, 1400);
  };

  if (!window.HydroMediaLibraryBound) {
    window.HydroMediaLibraryBound = true;
    document.addEventListener("click", (event) => {
      const copyButton = event.target.closest(".media-copy");
      if (copyButton && navigator.clipboard) {
        navigator.clipboard.writeText(copyButton.dataset.copy || "");
        const label = copyButton.textContent;
        copyButton.textContent = "Kopyalandı";
        flash(copyButton, "is-copied", () => (copyButton.textContent = label));
        return;
      }
      const copyLink = event.target.closest(".media-copy-link");
      if (copyLink && navigator.clipboard) {
        navigator.clipboard.writeText(window.location.origin + (copyLink.dataset.url || ""));
        copyLink.title = "Bağlantı kopyalandı";
        flash(copyLink, "is-copied", () => (copyLink.title = "Bağlantıyı kopyala"));
        return;
      }
      const dialog = document.querySelector(".media-settings-dialog");
      if (!dialog) return;
      if (event.target.closest("[data-media-settings-open]")) dialog.showModal();
      else if (event.target.closest("[data-media-settings-close]") || event.target === dialog) dialog.close();
    });
  }

  const searchInput = document.getElementById("media-search");
  const countBadge = document.getElementById("media-count-badge");
  if (searchInput && !searchInput.dataset.ready) {
    searchInput.dataset.ready = "1";
    searchInput.addEventListener("input", () => {
      const query = searchInput.value.toLowerCase().trim();
      let visible = 0;
      document.querySelectorAll(".media-card").forEach((card) => {
        const match = !query || (card.dataset.filename || "").toLowerCase().includes(query) || (card.dataset.id || "").toLowerCase().includes(query);
        card.classList.toggle("is-hidden", !match);
        if (match) visible++;
      });
      if (countBadge) countBadge.textContent = visible + " asset";
    });
  }

  const dropzone = document.getElementById("media-dropzone");
  const fileInput = document.getElementById("media-file-input");
  if (dropzone && fileInput && !dropzone.dataset.ready) {
    dropzone.dataset.ready = "1";
    const stop = (event) => {
      event.preventDefault();
      event.stopPropagation();
    };
    ["dragenter", "dragover"].forEach((name) =>
      dropzone.addEventListener(name, (event) => {
        stop(event);
        dropzone.classList.add("is-dragover");
      }),
    );
    ["dragleave", "drop"].forEach((name) =>
      dropzone.addEventListener(name, (event) => {
        stop(event);
        dropzone.classList.remove("is-dragover");
      }),
    );
    dropzone.addEventListener("drop", (event) => {
      if (event.dataTransfer?.files?.length) fileInput.files = event.dataTransfer.files;
    });
  }
})();
