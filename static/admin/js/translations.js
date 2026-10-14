// Translation editor (/admin/translations): filters the key/locale matrix as
// the admin types. Re-run safely after Studio's in-page navigation.
(() => {
  const input = document.getElementById("translation-search");
  if (!input || input.dataset.ready) return;
  input.dataset.ready = "1";
  input.addEventListener("input", () => {
    const query = input.value.trim().toLocaleLowerCase();
    document.querySelectorAll("[data-translation-row]").forEach((row) => {
      row.hidden = !(row.dataset.search || "").includes(query);
    });
  });
})();
