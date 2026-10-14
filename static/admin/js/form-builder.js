// Form builder (/admin/site/forms): adds and removes field rows and keeps the
// hidden "schema" input in sync with the visible controls. Field properties
// the builder does not edit (policy text, steps, cascades) are carried over
// from the saved schema by field id. Re-run safely after in-page navigation.
(() => {
  const root = document.querySelector("[data-form-builder]");
  if (!root || root.dataset.ready) return;
  root.dataset.ready = "1";
  const section = root.closest("[data-form-schema]");
  const list = root.querySelector("[data-form-fields]");
  const template = section?.querySelector("[data-form-field-template]");
  const schemaInput = root.querySelector("[data-form-schema-input]");
  if (!section || !list || !template || !schemaInput) return;
  let original = {};
  try {
    original = JSON.parse(section.dataset.formSchema || "{}");
  } catch (_) {
    original = {};
  }
  const savedFields = Object.fromEntries((original.fields || []).map((field) => [field.id, field]));
  const control = (scope, name) => scope.querySelector(`[name=${name}]`);
  const text = (name) => control(root, name)?.value ?? "";
  const checked = (name) => Boolean(control(root, name)?.checked);
  const lines = (value) => value.split(/\r?\n/).map((v) => v.trim()).filter(Boolean);

  const sync = () => {
    const fields = [...list.querySelectorAll("[data-form-field]")].map((row) => {
      const id = control(row, "field_id").value.trim();
      return {
        ...(savedFields[id] || {}),
        id,
        type: control(row, "field_type").value,
        label: control(row, "field_label").value.trim(),
        required: control(row, "field_required").checked,
        options: control(row, "field_options").value.split(",").map((v) => v.trim()).filter(Boolean),
      };
    });
    schemaInput.value = JSON.stringify({
      ...original,
      id: original.id || section.dataset.formId || "",
      title: text("title"),
      fields,
      notification_recipients: lines(text("notification_recipients")),
      autoresponder: checked("autoresponder"),
      autoresponder_subject: text("autoresponder_subject"),
      autoresponder_body: text("autoresponder_body"),
      turnstile_enabled: checked("turnstile_enabled"),
      conversion_tracking: checked("conversion_tracking"),
      google_ads_conversion_id: text("google_ads_conversion_id").trim(),
      google_ads_conversion_label: text("google_ads_conversion_label").trim(),
      meta_pixel_event_name: text("meta_pixel_event_name").trim(),
    });
  };

  root.addEventListener("input", sync);
  root.addEventListener("change", sync);
  root.addEventListener("submit", sync);
  root.addEventListener("click", (event) => {
    const remove = event.target.closest("[data-form-remove-field]");
    if (remove) {
      remove.closest("[data-form-field]")?.remove();
      sync();
    }
    if (event.target.closest("[data-form-add-field]")) {
      list.append(template.content.cloneNode(true));
      list.lastElementChild?.querySelector("[name=field_id]")?.focus();
      sync();
    }
  });
  sync();
})();
