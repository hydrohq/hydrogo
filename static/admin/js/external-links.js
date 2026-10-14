// Links that open a new tab must not hand the admin window to the target page.
(() => {
  const harden = () => {
    document.querySelectorAll('a[target="_blank"]').forEach((link) => link.setAttribute("rel", "noopener noreferrer"));
  };
  harden();
  document.addEventListener("hydro:admin-navigated", harden);
})();
