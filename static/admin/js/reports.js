// Reports (/admin/reports): the range select reloads the page for the chosen
// range, and the route table filters by path and kind. Delegated from the
// document so it keeps working after studio navigation swaps the page.
(() => {
  const filterRoutes = () => {
    const path = document.getElementById("report-path-filter");
    const kind = document.getElementById("report-kind-filter");
    if (!path || !kind) return;
    const needle = path.value.toLowerCase();
    document.querySelectorAll("[data-report-row]").forEach((row) => {
      const match = row.dataset.path.toLowerCase().includes(needle);
      const visible = kind.value === "views" ? Number(row.dataset.views) > 0 : kind.value === "errors" ? Number(row.dataset.errors) > 0 : true;
      row.hidden = !(match && visible);
    });
  };

  document.addEventListener("input", (event) => {
    if (event.target.id === "report-path-filter") filterRoutes();
  });

  document.addEventListener("change", (event) => {
    const target = event.target;
    if (target.id === "report-kind-filter") {
      filterRoutes();
      return;
    }
    if (!target.matches("[data-reports-range]") || target.value === "custom") return;
    const url = "/admin/reports?range=" + encodeURIComponent(target.value);
    if (window.studioNavigate) window.studioNavigate(url, true);
    else window.location.search = "?range=" + encodeURIComponent(target.value);
  });
})();
