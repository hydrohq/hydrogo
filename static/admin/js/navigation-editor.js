// Navigation editor (/admin/site/navigation/editor): add, remove, drag and
// reorder menu rows; keeps positions and parent choices in sync before the
// form posts. Re-run safely after Studio's in-page navigation.
(() => {
  const tree = document.getElementById("navigation-tree");
  const template = document.getElementById("navigation-row-template");
  const addButton = document.getElementById("navigation-add");
  if (!tree || !template || tree.dataset.ready) return;
  tree.dataset.ready = "1";
  const placeholder = template.dataset.rowKey || "__new__";
  const rows = () => [...tree.querySelectorAll("[data-navigation-row]")];
  const value = (row, name) => row.querySelector(`[name=${name}]`);
  const newID = () => globalThis.crypto?.randomUUID?.() || "nav-" + Date.now() + "-" + Math.random().toString(36).slice(2);

  const refresh = () => {
    const options = rows().map((row) => ({ id: value(row, "id").value, label: value(row, "label").value || "Yeni bağlantı" }));
    for (const row of rows()) {
      const select = value(row, "parent_id");
      const selected = select.value;
      const id = value(row, "id").value;
      select.replaceChildren(new Option("Ana menü", ""));
      options.filter((option) => option.id !== id).forEach((option) => select.add(new Option(option.label, option.id)));
      select.value = selected;
    }
  };
  const renumber = () => rows().forEach((row, index) => (value(row, "position").value = String(index)));

  addButton?.addEventListener("click", () => {
    const row = template.content.firstElementChild.cloneNode(true);
    const id = newID();
    row.querySelectorAll("[id], label[for]").forEach((node) => {
      if (node.id) node.id = node.id.replace(placeholder, id);
      if (node.htmlFor) node.htmlFor = node.htmlFor.replace(placeholder, id);
    });
    value(row, "id").value = id;
    tree.append(row);
    if (document.getElementById("published-page-paths")) value(row, "href").setAttribute("list", "published-page-paths");
    refresh();
    renumber();
    value(row, "label").focus();
  });

  tree.addEventListener("input", (event) => {
    if (event.target.matches("[name=label]")) refresh();
  });
  tree.addEventListener("click", (event) => {
    const row = event.target.closest("[data-navigation-row]");
    if (!row) return;
    if (event.target.closest("[data-remove-row]")) row.remove();
    if (event.target.closest("[data-move-up]") && row.previousElementSibling) tree.insertBefore(row, row.previousElementSibling);
    if (event.target.closest("[data-move-down]") && row.nextElementSibling) tree.insertBefore(row.nextElementSibling, row);
    refresh();
    renumber();
  });

  let dragging = null;
  tree.addEventListener("dragstart", (event) => {
    dragging = event.target.closest("[data-navigation-row]");
    if (dragging) {
      event.dataTransfer.setData("text/plain", "navigation-row");
      dragging.classList.add("is-dragging");
    }
  });
  tree.addEventListener("dragover", (event) => {
    const target = event.target.closest("[data-navigation-row]");
    if (!dragging || !target || target === dragging) return;
    event.preventDefault();
    const box = target.getBoundingClientRect();
    tree.insertBefore(dragging, event.clientY < box.top + box.height / 2 ? target : target.nextElementSibling);
  });
  tree.addEventListener("dragend", () => {
    dragging?.classList.remove("is-dragging");
    dragging = null;
    renumber();
  });
  tree.addEventListener("change", (event) => {
    if (event.target.matches("[name=parent_id]")) {
      const row = event.target.closest("[data-navigation-row]");
      const parent = rows().find((candidate) => value(candidate, "id").value === event.target.value);
      if (parent && row && parent.nextElementSibling !== row) tree.insertBefore(row, parent.nextElementSibling);
    }
    renumber();
  });

  refresh();
  renumber();
  if (document.getElementById("published-page-paths")) {
    tree.querySelectorAll("[name=href]").forEach((input) => input.setAttribute("list", "published-page-paths"));
  }
})();
