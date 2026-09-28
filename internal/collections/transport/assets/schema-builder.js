(() => {
	const fields = document.querySelector("#collection-fields[data-current][data-max]");
	const add = document.querySelector("[data-add-collection-field]");
	const template = document.querySelector("template[data-collection-field-template]");
	if (!fields || !add || !template) return;
	add.addEventListener("click", () => {
		const index = Number(fields.dataset.current);
		const max = Number(fields.dataset.max);
		if (!Number.isInteger(index) || !Number.isInteger(max) || index >= max) return;
		const row = template.content.cloneNode(true);
		row.querySelectorAll("[name]").forEach((input) => {
			input.name = input.name.replaceAll("-1", String(index));
		});
		fields.append(row);
		fields.dataset.current = String(index + 1);
		add.disabled = index + 1 >= max;
	});
})();
