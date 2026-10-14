// Sign-in page theme bootstrap. Loaded as a blocking script in <head> so the
// saved light/dark choice applies before first paint without an inline script.
(() => {
  try {
    const saved = localStorage.getItem("hydro-ui-theme-v2");
    const variant = localStorage.getItem("hydro-theme") === "light" ? "light" : "dark";
    if (saved && saved.endsWith("-light")) document.documentElement.setAttribute("data-theme", "light");
    else if (saved && saved.endsWith("-dark")) document.documentElement.setAttribute("data-theme", "dark");
    else document.documentElement.setAttribute("data-theme", variant);
  } catch (_) {}
})();
