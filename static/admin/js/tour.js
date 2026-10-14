(function(){
	const dismissedKey = 'hydro-onboarding-dismissed';

	function initTour() {
		const role = document.body?.dataset.studioRole || 'admin';
		const seenKey = 'hydro-tour-seen:' + role;
		const tour = document.querySelector('[data-onboarding-tour]');
		if (!tour) return;
		const dialog = tour.closest('.admin-ux-dialog');
		if (!dialog) return;

		const steps = [...tour.querySelectorAll('[data-tour-step]')];
		if (!steps.length) return;

		const prev = tour.querySelector('[data-tour-prev]');
		const next = tour.querySelector('[data-tour-next]');
		const counter = tour.querySelector('[data-tour-counter]');
		const dots = [...tour.querySelectorAll('[data-tour-dots] .onboarding-dot')];
		let index = 0;

		const titles = [
			'Studio’ya Hoş Geldiniz',
			'İçerik & Yayın Yönetimi',
			'Geliştirici & Sistem Araçları',
			'Kısayollar & Başarı'
		];

		function show(i) {
			index = Math.max(0, Math.min(steps.length - 1, i));
			steps.forEach((step, idx) => {
				step.hidden = idx !== index;
			});
			dots.forEach((dot, idx) => {
				dot.classList.toggle('is-active', idx === index);
			});
			if (prev) prev.disabled = index === 0;
			if (next) {
				next.textContent = index === steps.length - 1 ? 'Rehberi Tamamla ✓' : 'İleri →';
				next.classList.toggle('button-success', index === steps.length - 1);
				next.classList.toggle('button-primary', index !== steps.length - 1);
			}
			if (counter) counter.textContent = 'Adım ' + (index + 1) + ' / ' + steps.length;
			const titleEl = tour.querySelector('[data-tour-title]');
			if (titleEl && titles[index]) titleEl.textContent = titles[index];
		}

		function dismiss() {
			try {
				localStorage.setItem(dismissedKey, '1');
				localStorage.setItem(dismissedKey + ':' + role, '1');
			} catch (_) {}
			dialog.hidden = true;
		}

		tour.addEventListener('click', e => {
			if (e.target.closest('[data-tour-next]')) {
				if (index < steps.length - 1) {
					show(index + 1);
				} else {
					dismiss();
				}
				return;
			}
			if (e.target.closest('[data-tour-prev]') && index > 0) {
				show(index - 1);
				return;
			}
			if (e.target.closest('[data-onboarding-dismiss]')) {
				dismiss();
				return;
			}
		});

		dialog.addEventListener('click', e => {
			if (e.target === dialog || e.target.closest('[data-ux-close]')) {
				dismiss();
			}
		});

		show(0);

		try {
			const isDismissed = localStorage.getItem(dismissedKey) === '1' || localStorage.getItem(dismissedKey + ':' + role) === '1';
			if (isDismissed) {
				dialog.hidden = true;
			} else if (!sessionStorage.getItem(seenKey)) {
				sessionStorage.setItem(seenKey, '1');
				setTimeout(() => { dialog.hidden = false; }, 400);
			}
		} catch (_) {
			dialog.hidden = true;
		}
	}

	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', initTour, { once: true });
	} else {
		initTour();
	}
	document.addEventListener('hydro:admin-navigated', initTour);
})();
