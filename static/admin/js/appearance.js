// Brand and appearance (/admin/site/appearance): live contrast feedback and a
// preview of colours, font and logo. Messages come from data-message-*
// attributes on #appearance-contrast so the copy stays in the template.
(() => {
  const out = document.getElementById("appearance-contrast");
  const preview = document.getElementById("appearance-preview");
  if (!out || !preview || out.dataset.ready) return;
  out.dataset.ready = "1";
  const ids = ["appearance-primary", "appearance-accent", "appearance-dark-primary", "appearance-dark-accent"];
  const field = (id) => document.getElementById(id);
  const luminance = (hex) => {
    const channels = (hex.slice(1).match(/../g) || ["00", "00", "00"])
      .map((part) => parseInt(part, 16) / 255)
      .map((c) => (c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)));
    return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
  };
  const update = () => {
    const ratios = ids.map((id) => field(id)).filter(Boolean).map((input) => 1.05 / (luminance(input.value) + 0.05));
    if (ratios.length === 0) return;
    const min = Math.min(...ratios);
    const state = min < 4.5 ? "low" : min >= 7 ? "aaa" : "aa";
    const template = out.dataset["message" + state.charAt(0).toUpperCase() + state.slice(1)] || "";
    out.textContent = template.replace("{ratio}", min.toFixed(2));
    out.dataset.state = state;
    preview.style.setProperty("--appearance-primary", field(ids[0])?.value || "");
    preview.style.setProperty("--appearance-accent", field(ids[1])?.value || "");
    const font = field("appearance-font");
    if (font) preview.style.setProperty("--appearance-font", font.value + ", system-ui");
  };
  ids.forEach((id) => field(id)?.addEventListener("input", update));
  field("appearance-font")?.addEventListener("change", update);
  ["appearance-logo", "appearance-dark-logo"].forEach((id) =>
    field(id)?.addEventListener("input", () => {
      const img = document.getElementById("appearance-preview-logo");
      const path = field(id).value;
      if (!img) return;
      if (path.startsWith("/") && !path.startsWith("//")) img.src = path;
      img.hidden = !path;
    }),
  );
  update();
})();
