// Studio router and shared admin behaviour: Spotlight search (⌘K), the
// fetch-and-swap navigation between admin pages (window.studioNavigate),
// toasts (window.HydroToast), the media picker, rich-text and list widgets.
// Loaded with defer from the admin frame; the light/dark switch, the user
// menu, notifications and the mobile drawer are Alpine components in
// /admin-assets/shell.js.
(()=>{
const defaultCommands = [
	{title: 'Yeni Haber Yaz', sub: 'Haber taslağı oluştur', titleEn: 'Write News', subEn: 'Create a news draft', href: '/admin/news/new', icon: 'plus', badge: 'İçerik', badgeEn: 'Content', permission: 'admin.content.write'},
	{title: 'Yeni Sayfa Ekle', sub: 'Yeni statik veya dinamik sayfa', titleEn: 'Add a Page', subEn: 'Create a static or dynamic page', href: '/admin/pages/new', icon: 'plus', badge: 'İçerik', badgeEn: 'Content', permission: 'admin.content.write'},
	{title: 'Medya Yükle', sub: 'Görsel ve dosya yükleme', titleEn: 'Upload Media', subEn: 'Upload images and files', href: '/admin/media', icon: 'media', badge: 'Medya', badgeEn: 'Media', permission: 'admin.media.manage'},
	{title: 'Üye Yönetimi', sub: 'Kullanıcı hesapları ve oturumlar', titleEn: 'Manage Members', subEn: 'User accounts and sessions', href: '/admin/members', icon: 'security', badge: 'Üyeler', badgeEn: 'Members', permission: 'admin.members.manage'},
	{title: 'Raporlar ve Analitik', sub: 'Canlı telemetri ve performans', titleEn: 'Reports & Analytics', subEn: 'Live telemetry and performance', href: '/admin/reports', icon: 'reports', badge: 'Metrik', badgeEn: 'Metrics', adminOnly: true},
	{title: 'Yönetim Araçları', sub: 'Site ayarları ve yapılandırma', titleEn: 'Administration Tools', subEn: 'Site settings and configuration', href: '/admin/control', icon: 'control', badge: 'Sistem', badgeEn: 'System', adminOnly: true},
	{title: 'Operasyon Durumu', sub: 'Kuyruk ve sistem sağlığı', titleEn: 'Operations Status', subEn: 'Queue and system health', href: '/admin/operations', icon: 'operations', badge: 'Sistem', badgeEn: 'System', adminOnly: true},
	{title: 'Yedek İndir', sub: 'Site verilerinin bir kopyasını alın', titleEn: 'Download Backup', subEn: 'Download a copy of site data', href: '/admin/backup', icon: 'backup', badge: 'Sistem', badgeEn: 'System', adminOnly: true},
	{title: 'Güvenlik & Yetkiler', sub: 'Kullanıcı izinleri ve denetim', titleEn: 'Security & Permissions', subEn: 'User permissions and audit', href: '/admin/security', icon: 'security', badge: 'Sistem', badgeEn: 'System', adminOnly: true},
	{title: 'Temayı Değiştir', sub: 'Karanlık / Aydınlık mod geçişi', titleEn: 'Change Theme', subEn: 'Switch between dark and light', action: 'toggle-theme', icon: 'sparkles', badge: 'Arayüz', badgeEn: 'Appearance'},
	{title: 'Siteyi Görüntüle', sub: 'Herkese açık yayınlanan site', titleEn: 'View Site', subEn: 'Published public website', href: '/', target: '_blank', icon: 'external', badge: 'Önizleme', badgeEn: 'Preview'}
];

const availableCommands = () => {
	const permissions = new Set((document.body?.dataset.adminPermissions || '').split(/\s+/).filter(Boolean));
	const isAdmin = document.body?.dataset.studioRole === 'admin';
	const english = document.documentElement.lang === 'en';
	return defaultCommands.filter(command => (!command.permission || permissions.has(command.permission)) && (!command.adminOnly || isAdmin)).map(command => english ? { ...command, title: command.titleEn, sub: command.subEn, badge: command.badgeEn } : command);
};

const getStorage = key => {
	try { return JSON.parse(localStorage.getItem(key) || '[]'); } catch(_) { return []; }
};

const setStorage = (key, val) => {
	try { localStorage.setItem(key, JSON.stringify(val)); } catch(_) {}
};

const recordRecent = (title, href) => {
	if (!href || href.startsWith('/login') || href.startsWith('/auth')) return;
	let recents = getStorage('hydro_recents').filter(r => r.href !== href);
	recents.unshift({ title, href, time: Date.now() });
	if (recents.length > 8) recents = recents.slice(0, 8);
	setStorage('hydro_recents', recents);
};

const toggleFavorite = (title, href, e) => {
	if (e) { e.preventDefault(); e.stopPropagation(); }
	let favs = getStorage('hydro_favs');
	const idx = favs.findIndex(f => f.href === href);
	if (idx >= 0) {
		favs.splice(idx, 1);
	} else {
		favs.push({ title, href });
	}
	setStorage('hydro_favs', favs);
	const searchShell = document.querySelector('.studio-search');
	if (searchShell && searchShell.__renderSearch) searchShell.__renderSearch();
};

const initSearch = () => {
	const searchShell = document.querySelector('.studio-search');
	if (!searchShell || searchShell.dataset.bound) return;
	searchShell.dataset.bound = 'true';

	let input = searchShell.querySelector('input');
	if (!input) {
		input = document.createElement('input');
		input.type = 'search';
		const english = document.documentElement.lang === 'en';
		input.placeholder = english ? 'Search screens and commands (⌘K)…' : 'Komut veya ekran ara (⌘K)...';
		input.setAttribute('aria-label', english ? 'Search Studio and run commands' : 'Studio içinde ara ve komut çalıştır');
		input.setAttribute('role', 'combobox');
		input.setAttribute('aria-autocomplete', 'list');
		input.setAttribute('aria-expanded', 'false');
		input.setAttribute('aria-controls', 'studio-search-results');
		const hint = searchShell.querySelector('kbd');
		searchShell.querySelector('span')?.remove();
		searchShell.insertBefore(input, hint);
	}

	let results = searchShell.querySelector('.studio-search-results');
	if (!results) {
		results = document.createElement('div');
		results.id = 'studio-search-results';
		results.className = 'studio-search-results';
		results.setAttribute('role', 'listbox');
		results.hidden = true;
		searchShell.appendChild(results);
	}
	const spotlightDialog = document.createElement('dialog');
	spotlightDialog.className = 'studio-search-dialog';
	spotlightDialog.setAttribute('aria-label', document.documentElement.lang === 'en' ? 'Search Studio' : 'Studio içinde ara');
	spotlightDialog.addEventListener('click', event => {
		const rect = spotlightDialog.getBoundingClientRect();
		const isInDialog = (rect.top <= event.clientY && event.clientY <= rect.top + rect.height &&
			rect.left <= event.clientX && event.clientX <= rect.left + rect.width);
		if (!isInDialog || event.target === spotlightDialog) spotlightDialog.close();
	});
	document.body.appendChild(spotlightDialog);
	let searchPlaceholder = null;
	spotlightDialog.addEventListener('close', () => {
		if (searchPlaceholder?.parentNode) searchPlaceholder.replaceWith(searchShell);
		searchPlaceholder = null;
		results.hidden = true;
		input.setAttribute('aria-expanded', 'false');
		input.blur();
	});

	let focusedIndex = 0;
	let searchTimer = 0;
	let searchRequest = null;

	const render = () => {
		const searchLocale = document.documentElement.lang === 'en' ? 'en' : 'tr-TR';
		const query = input.value.trim().toLocaleLowerCase(searchLocale);
		results.innerHTML = '';
		focusedIndex = -1;

		const favs = getStorage('hydro_favs');
		const recents = getStorage('hydro_recents');

		const createItem = (item) => {
			const row = document.createElement('a');
			row.className = 'search-result-item';
			row.setAttribute('role', 'option');
			row.setAttribute('tabindex', '-1');
			row.href = item.href || '#';
			if (item.target) row.target = item.target;

			const icon = document.createElement('span');
			icon.className = 'search-result-icon';
			icon.dataset.studioIcon = item.icon || 'folder';
			const text = document.createElement('div');
			text.className = 'search-result-text';
			const title = document.createElement('span');
			title.className = 'search-result-title';
			title.textContent = item.title || '';
			text.appendChild(title);
			if (item.sub) {
				const sub = document.createElement('small');
				sub.className = 'search-result-sub';
				sub.textContent = item.sub;
				text.appendChild(sub);
			}
			row.append(icon, text);
			if (item.badge) {
				const badge = document.createElement('span');
				badge.className = 'search-result-badge';
				badge.textContent = item.badge;
				row.appendChild(badge);
			}

			if (item.action === 'toggle-theme') {
				row.addEventListener('click', e => {
					e.preventDefault();
					if (spotlightDialog.open) spotlightDialog.close();
					const themeBtn = document.getElementById('theme-toggle-btn');
					if (themeBtn) themeBtn.click();
					results.hidden = true;
					input.setAttribute('aria-expanded', 'false');
				});
			} else if (item.href && !item.target) {
				row.addEventListener('click', e => {
					e.preventDefault();
					if (spotlightDialog.open) spotlightDialog.close();
					results.hidden = true;
					input.setAttribute('aria-expanded', 'false');
					window.studioNavigate(item.href, true);
				});
			}

			row.addEventListener('mouseenter', () => {
				const allItems = [...results.querySelectorAll('.search-result-item')];
				focusedIndex = allItems.indexOf(row);
				allItems.forEach((it, idx) => {
					const isF = idx === focusedIndex;
					it.classList.toggle('is-focused', isF);
					it.setAttribute('aria-selected', String(isF));
				});
			});

			return row;
		};

		if (!query) {
			const cmdHeader = document.createElement('div');
			cmdHeader.className = 'search-section-title';
			cmdHeader.textContent = document.documentElement.lang === 'en' ? 'Quick actions & commands' : 'Hızlı Menü & Komutlar';
			results.appendChild(cmdHeader);
			availableCommands().forEach(cmd => results.appendChild(createItem(cmd)));
		} else {
			const matchedCmds = availableCommands().filter(c => c.title.toLocaleLowerCase(searchLocale).includes(query) || (c.sub && c.sub.toLocaleLowerCase(searchLocale).includes(query)));
			const navLinks = [...document.querySelectorAll('.studio-sidebar nav a.nav-item')].map(a => ({
				title: a.textContent.trim(),
				href: a.href,
				badge: document.documentElement.lang === 'en' ? 'Page' : 'Sayfa'
			})).filter(n => n.title.toLocaleLowerCase(searchLocale).includes(query));

			const combined = [...matchedCmds];
			navLinks.forEach(n => {
				if (!combined.some(c => c.href === n.href)) combined.push(n);
			});

			if (combined.length > 0) {
				combined.forEach(item => results.appendChild(createItem(item)));
			} else {
				const empty = document.createElement('div');
				empty.className = 'empty';
			empty.textContent = document.documentElement.lang === 'en' ? 'No matching results.' : 'Eşleşen sonuç bulunamadı.';
				results.appendChild(empty);
			}
		}

		const footer = document.createElement('div');
		footer.className = 'search-footer';
		footer.innerHTML = document.documentElement.lang === 'en' ? '<span><kbd>↑↓</kbd> Navigate</span><span><kbd>↵</kbd> Open</span><span><kbd>ESC</kbd> Close</span>' : '<span><kbd>↑↓</kbd> Gezin</span><span><kbd>↵</kbd> Aç</span><span><kbd>ESC</kbd> Kapat</span>';
		results.appendChild(footer);

		if (window.HydroApplyIcons) window.HydroApplyIcons();
		results.hidden = false;
		input.setAttribute('aria-expanded', 'true');
		const currentItems = results.querySelectorAll('.search-result-item');
		if (currentItems.length > 0) {
			if (focusedIndex < 0 || focusedIndex >= currentItems.length) focusedIndex = 0;
			currentItems.forEach((it, idx) => {
				const isF = idx === focusedIndex;
				it.classList.toggle('is-focused', isF);
				it.setAttribute('aria-selected', String(isF));
			});
		}
		clearTimeout(searchTimer);
		if (searchRequest) searchRequest.abort();
		if (query.length >= 2) {
			const expectedQuery = query;
			searchTimer = setTimeout(async () => {
				searchRequest = new AbortController();
				try {
					const locale = document.documentElement.lang || '';
					const response = await fetch('/admin/api/search?q=' + encodeURIComponent(expectedQuery) + '&locale=' + encodeURIComponent(locale), {credentials:'same-origin',headers:{'Accept':'application/json'},signal:searchRequest.signal});
					if (!response.ok || input.value.trim().toLocaleLowerCase(searchLocale) !== expectedQuery) return;
					const data = await response.json();
					const remoteItems = Array.isArray(data.items) ? data.items : [];
					const footer = results.querySelector('.search-footer');
					if (remoteItems.length) {
						const header = document.createElement('div');
						header.className = 'search-section-title';
						header.dataset.remoteSearch = 'true';
					header.textContent = document.documentElement.lang === 'en' ? 'Content and records' : 'İçerik ve kayıtlar';
						results.insertBefore(header, footer);
						remoteItems.forEach(item => {
							if (!item.href || [...results.querySelectorAll('.search-result-item')].some(row => row.href === new URL(item.href, location.origin).href)) return;
							results.insertBefore(createItem({title:item.title,sub:item.sub,badge:item.badge,href:item.href,icon:'search'}), footer);
						});
						if (window.HydroApplyIcons) window.HydroApplyIcons();
					} else if (!results.querySelector('.search-result-item')) {
						const empty = document.createElement('div');
						empty.className = 'empty';
					empty.textContent = document.documentElement.lang === 'en' ? 'No matching results.' : 'Eşleşen sonuç bulunamadı.';
						results.insertBefore(empty, footer);
					}
				} catch (error) {
					if (error.name !== 'AbortError') console.debug('Admin search is temporarily unavailable');
				}
			}, 120);
		}
	};

	searchShell.__renderSearch = render;
	searchShell.__openSpotlight = () => {
		if (spotlightDialog.open) return;
		searchPlaceholder = document.createComment('studio-search-position');
		searchShell.parentNode.insertBefore(searchPlaceholder, searchShell);
		spotlightDialog.appendChild(searchShell);
		spotlightDialog.showModal();
		focusedIndex = 0;
		render();
		setTimeout(() => {
			input.focus();
			input.select();
		}, 30);
	};
	searchShell.addEventListener('click', () => {
		if (!spotlightDialog.open && searchShell.__openSpotlight) {
			searchShell.__openSpotlight();
		}
	});
	input.addEventListener('input', render);
	input.addEventListener('focus', render);

	input.addEventListener('keydown', e => {
		const items = results.querySelectorAll('.search-result-item');
		if (e.key === 'Escape') {
			input.value = '';
			results.hidden = true;
			input.setAttribute('aria-expanded', 'false');
			if (spotlightDialog.open) spotlightDialog.close();
			input.blur();
			return;
		}
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			if (items.length > 0) {
				focusedIndex = (focusedIndex + 1) % items.length;
				items.forEach((it, idx) => {
					it.classList.toggle('is-focused', idx === focusedIndex);
					it.setAttribute('aria-selected', String(idx === focusedIndex));
				});
				items[focusedIndex]?.scrollIntoView({ block: 'nearest' });
			}
			return;
		}
		if (e.key === 'ArrowUp') {
			e.preventDefault();
			if (items.length > 0) {
				focusedIndex = (focusedIndex - 1 + items.length) % items.length;
				items.forEach((it, idx) => {
					it.classList.toggle('is-focused', idx === focusedIndex);
					it.setAttribute('aria-selected', String(idx === focusedIndex));
				});
				items[focusedIndex]?.scrollIntoView({ block: 'nearest' });
			}
			return;
		}
		if (e.key === 'Enter' && !results.hidden) {
			e.preventDefault();
			if (focusedIndex >= 0 && items[focusedIndex]) {
				items[focusedIndex].click();
			} else if (items.length > 0) {
				items[0].click();
			}
		}
	});

	if (document.documentElement.dataset.searchGlobalBound !== 'true') {
		document.documentElement.dataset.searchGlobalBound = 'true';
		document.addEventListener('keydown', e => {
			const typing = /INPUT|TEXTAREA|SELECT/.test(document.activeElement?.tagName) || document.activeElement?.isContentEditable;
			const commandKey = (e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k';
			const slashKey = !typing && !e.ctrlKey && !e.metaKey && !e.altKey && e.key === '/';
			if (!commandKey && !slashKey) return;
			const shell = document.querySelector('.studio-search');
			if (!shell || !shell.__openSpotlight) return;
			e.preventDefault();
			shell.__openSpotlight();
		});
		document.addEventListener('click', e => {
			const shell = document.querySelector('.studio-search');
			if (!shell || shell.contains(e.target)) return;
			const activeInput = shell.querySelector('input');
			const activeResults = shell.querySelector('.studio-search-results');
			if (activeResults) activeResults.hidden = true;
			if (activeInput) activeInput.setAttribute('aria-expanded', 'false');
		});
	}
};

const getBar = () => {
	let bar = document.getElementById('studio-progress');
	if (!bar && document.body) {
		bar = document.createElement('div');
		bar.id = 'studio-progress';
		bar.style.cssText = 'position:fixed;top:0;left:0;height:3px;background:linear-gradient(90deg,#2563eb,#3b82f6,#60a5fa);z-index:99999;transition:width .2s ease,opacity .25s ease;width:0%;opacity:0;pointer-events:none;';
		document.body.appendChild(bar);
	}
	return bar;
};

const startProgress = () => {
	const bar = getBar();
	if (!bar) return;
	bar.style.transition = 'width .2s ease, opacity .1s ease';
	bar.style.opacity = '1';
	bar.style.width = '45%';
	setTimeout(() => { if (bar.style.opacity === '1') bar.style.width = '80%'; }, 200);
};

const endProgress = () => {
	const bar = getBar();
	if (!bar) return;
	bar.style.width = '100%';
	setTimeout(() => {
		bar.style.opacity = '0';
		setTimeout(() => { bar.style.width = '0%'; }, 250);
	}, 120);
};

const toastCache = new Map();
window.HydroToast = (message, type = 'info', title = '', undoCallback = null, duration = 4000) => {
	if (!message) return;
	const cacheKey = type + ':' + message;
	const now = Date.now();
	if (toastCache.has(cacheKey) && now - toastCache.get(cacheKey) < 1500) {
		return;
	}
	toastCache.set(cacheKey, now);

	let container = document.querySelector('.studio-toast-container');
	if (!container) {
		container = document.createElement('div');
		container.className = 'studio-toast-container';
		document.body.appendChild(container);
	}

	const toast = document.createElement('div');
	toast.className = 'studio-toast studio-toast-' + type;
	toast.setAttribute('role', (type === 'error' || type === 'warning') ? 'alert' : 'status');
	toast.setAttribute('aria-live', (type === 'error' || type === 'warning') ? 'assertive' : 'polite');

	const icons = {
		success: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M20 6 9 17l-5-5"/></svg>',
		error: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>',
		warning: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>',
		info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>'
	};

	const iconHtml = '<div class="studio-toast-icon">' + (icons[type] || icons.info) + '</div>';
	const closeBtnHtml = '<button type="button" class="studio-toast-close" aria-label="Kapat"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg></button>';
	const progressBar = '<div class="studio-toast-progress" style="width: 100%;"></div>';

	toast.innerHTML = iconHtml + '<div class="studio-toast-content"></div>' + closeBtnHtml + progressBar;
	const toastContent = toast.querySelector('.studio-toast-content');
	if (title) {
		const titleNode = document.createElement('strong');
		titleNode.className = 'studio-toast-title';
		titleNode.textContent = title;
		toastContent.appendChild(titleNode);
	}
	const messageNode = document.createElement('p');
	messageNode.className = 'studio-toast-message';
	messageNode.textContent = message;
	toastContent.appendChild(messageNode);
	if (typeof undoCallback === 'function') {
		const undoNode = document.createElement('button');
		undoNode.type = 'button';
		undoNode.className = 'studio-toast-undo';
		undoNode.textContent = document.documentElement.lang === 'en' ? 'Undo' : 'Geri Al';
		toastContent.appendChild(undoNode);
	}
	container.appendChild(toast);

	let isPaused = false;
	let startTime = Date.now();
	let remaining = duration;
	let timer = null;
	const barEl = toast.querySelector('.studio-toast-progress');

	const dismiss = () => {
		if (toast.classList.contains('is-hiding')) return;
		toast.classList.add('is-hiding');
		setTimeout(() => { toast.remove(); }, 240);
	};

	const startTimer = () => {
		if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
			timer = setTimeout(dismiss, duration);
			return;
		}
		if (barEl) {
			barEl.style.transition = 'width ' + remaining + 'ms linear';
			barEl.style.width = '0%';
		}
		startTime = Date.now();
		timer = setTimeout(dismiss, remaining);
	};

	startTimer();

	toast.addEventListener('mouseenter', () => {
		isPaused = true;
		clearTimeout(timer);
		if (barEl) {
			const computedWidth = window.getComputedStyle(barEl).width;
			barEl.style.transition = 'none';
			barEl.style.width = computedWidth;
		}
		remaining -= (Date.now() - startTime);
		if (remaining < 500) remaining = 500;
	});

	toast.addEventListener('mouseleave', () => {
		if (isPaused) {
			isPaused = false;
			startTimer();
		}
	});

	const closeBtn = toast.querySelector('.studio-toast-close');
	if (closeBtn) closeBtn.addEventListener('click', dismiss);

	const undoBtn = toast.querySelector('.studio-toast-undo');
	if (undoBtn && undoCallback) {
		undoBtn.addEventListener('click', () => {
			try { undoCallback(); } catch(err) { console.warn('Undo error:', err); }
			dismiss();
		});
	}
};

window.HydroAnimateKPIs = () => {
	if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
	const targets = document.querySelectorAll('.dashboard-kpi strong, .kpi-value, .meta-item strong, .wb-metric strong, .dashboard-stat strong, .resource-kpi strong, .sec-kpi strong, .ops-kpi strong');
	targets.forEach(el => {
		if (el.dataset.kpiAnimated === 'true') return;
		const rawText = el.textContent.trim();
		const num = parseFloat(rawText.replace(/[^0-9.-]+/g, ''));
		if (isNaN(num) || num <= 0 || rawText.includes('%') || rawText.includes('ms')) return;
		el.dataset.kpiAnimated = 'true';
		const duration = 400;
		const start = performance.now();
		const step = now => {
			const elapsed = now - start;
			const progress = Math.min(elapsed / duration, 1);
			// Ease out cubic
			const ease = 1 - Math.pow(1 - progress, 3);
			const current = Math.floor(ease * num);
			el.textContent = current.toLocaleString('tr-TR');
			if (progress < 1) {
				requestAnimationFrame(step);
			} else {
				el.textContent = rawText;
			}
		};
		requestAnimationFrame(step);
	});
};

window.HydroInitSpotlights = () => {
	if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
	const cards = document.querySelectorAll('.wb-metric, .resource-kpi, .dev-endpoint-card, .sec-card, .ops-card, .dashboard-kpi, .radial-chart-card, .traffic-bars-card, .report-info-card, .studio-card');
	cards.forEach(card => {
		if (card.dataset.spotlightInit === 'true') return;
		card.dataset.spotlightInit = 'true';
		card.addEventListener('mousemove', e => {
			const rect = card.getBoundingClientRect();
			const x = e.clientX - rect.left;
			const y = e.clientY - rect.top;
			card.style.setProperty('--mouse-x', x + 'px');
			card.style.setProperty('--mouse-y', y + 'px');
		});
	});
};

let activeModalOpener = null;
const setupModals = () => {
	const modals = document.querySelectorAll('.studio-modal, .admin-media-picker-modal');
	modals.forEach(modal => {
		if (modal.dataset.modalInit === 'true') return;
		modal.dataset.modalInit = 'true';

		const observer = new MutationObserver(() => {
			const isVisible = !modal.hidden && modal.style.display !== 'none';
			if (isVisible) {
				activeModalOpener = document.activeElement;
				const firstFocusable = modal.querySelector('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
				if (firstFocusable) firstFocusable.focus();
			}
		});
		observer.observe(modal, { attributes: true, attributeFilter: ['hidden', 'style'] });
	});
};

document.addEventListener('keydown', e => {
	if (e.key === 'Escape') {
		const openModal = document.querySelector('.studio-modal:not([hidden]), .admin-media-picker-modal:not([hidden])');
		if (openModal) {
			openModal.hidden = true;
			if (activeModalOpener && typeof activeModalOpener.focus === 'function') {
				activeModalOpener.focus();
				activeModalOpener = null;
			}
		}
	} else if (e.key === 'Tab') {
		const openModal = document.querySelector('.studio-modal:not([hidden]), .admin-media-picker-modal:not([hidden])');
		if (openModal) {
			const focusables = Array.from(openModal.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])')).filter(el => !el.disabled);
			if (focusables.length > 0) {
				const first = focusables[0];
				const last = focusables[focusables.length - 1];
				if (e.shiftKey && document.activeElement === first) {
					e.preventDefault();
					last.focus();
				} else if (!e.shiftKey && document.activeElement === last) {
					e.preventDefault();
					first.focus();
				}
			}
		}
	}
});

const setupFormFeedback = () => {
	document.querySelectorAll('form').forEach(form => {
		if (form.dataset.feedbackInit === 'true') return;
		form.dataset.feedbackInit = 'true';

		form.addEventListener('submit', e => {
			if (form.dataset.submitting === 'true') {
				e.preventDefault();
				return;
			}
			const submitBtn = form.querySelector('button[type="submit"], input[type="submit"], button:not([type])');
			if (submitBtn) {
				submitBtn.classList.add('is-loading');
				form.dataset.submitting = 'true';
				setTimeout(() => {
					form.dataset.submitting = 'false';
					submitBtn.classList.remove('is-loading');
				}, 4000);
			}
		});
	});
};

const updateActiveNav = url => {
	const parsed = new URL(url, window.location.origin);
	const path = parsed.pathname;
	const nav = document.querySelector('.studio-sidebar nav');
	if (!nav) return;
	const links = Array.from(nav.querySelectorAll('a.nav-item'));
	let bestMatch = null;
	let maxMatchLen = -1;
	links.forEach(link => {
		link.classList.remove('is-active');
		const linkPath = new URL(link.href, window.location.origin).pathname;
		if (path === linkPath) {
			bestMatch = link;
			maxMatchLen = 9999;
		} else if (linkPath !== '/admin' && (path === linkPath || path.startsWith(linkPath + '/')) && linkPath.length > maxMatchLen) {
			bestMatch = link;
			maxMatchLen = linkPath.length;
		}
	});
	if (bestMatch) {
		bestMatch.classList.add('is-active');
		const tree = bestMatch.closest('.nav-tree');
		if (tree) {
			tree.classList.add('is-open');
			const toggle = tree.querySelector('.nav-tree-toggle');
			if (toggle) toggle.setAttribute('aria-expanded', 'true');
		}
		recordRecent(bestMatch.textContent.trim(), bestMatch.href);
	}
};

const currentDocumentNonce = () => {
	const node = document.querySelector('head style[nonce], head script[nonce], body script[nonce], style[nonce], script[nonce]');
	if (!node) return '';
	return node.nonce || node.getAttribute('nonce') || '';
};

const adoptFragmentStyles = container => {
	const nonce = currentDocumentNonce();
	container.querySelectorAll('style').forEach(oldStyle => {
		// Normalize while the fragment is still inert. Inserting the style with
		// the fetched response's nonce would make CSP reject it in the current
		// document. Keeping it in the fragment also prevents head-style buildup.
		if (nonce) {
			oldStyle.setAttribute('nonce', nonce);
			try { oldStyle.nonce = nonce; } catch (_) {}
		}
		const newStyle = document.createElement('style');
		if (nonce) {
			newStyle.setAttribute('nonce', nonce);
			try { newStyle.nonce = nonce; } catch (_) {}
		}
		newStyle.textContent = oldStyle.textContent;
		oldStyle.parentNode.replaceChild(newStyle, oldStyle);
	});
};

const executeScripts = container => {
	const nonce = currentDocumentNonce();
	container.querySelectorAll('script').forEach(oldScript => {
		const newScript = document.createElement('script');
		Array.from(oldScript.attributes).forEach(attr => newScript.setAttribute(attr.name, attr.value));
		if (nonce) {
			newScript.setAttribute('nonce', nonce);
			try { newScript.nonce = nonce; } catch (_) {}
		}
		newScript.textContent = oldScript.textContent;
		oldScript.parentNode.replaceChild(newScript, oldScript);
	});
};

window.HydroInitQuill = () => {
	if (typeof Quill === 'undefined') {
		setTimeout(window.HydroInitQuill, 60);
		return;
	}
	document.querySelectorAll('[data-richtext]').forEach(field => {
		if (field.dataset.quillInit === 'true') return;
		const value = field.querySelector('.richtext-value');
		const editor = field.querySelector('.richtext-editor');
		const toolbar = field.querySelector('.richtext-toolbar');
		if (!editor || !toolbar || !value) return;
		field.dataset.quillInit = 'true';
		try {
			// Remove any secondary auto-generated toolbars if re-initialized
			field.querySelectorAll('.ql-toolbar:not(.richtext-toolbar)').forEach(t => t.remove());
			const quill = new Quill(editor, { theme: 'snow', modules: { toolbar: toolbar } });
			const editable = editor.querySelector('.ql-editor');
			if (editable) {
				const label = editor.getAttribute('aria-label');
				const labelledBy = editor.getAttribute('aria-labelledby');
				editable.setAttribute('role', 'textbox');
				editable.setAttribute('tabindex', '0');
				if (label) editable.setAttribute('aria-label', label);
				if (labelledBy) editable.setAttribute('aria-labelledby', labelledBy);
				editable.setAttribute('aria-multiline', 'true');
				editor.removeAttribute('role');
				editor.removeAttribute('tabindex');
				editor.removeAttribute('aria-label');
				editor.removeAttribute('aria-labelledby');
				editor.removeAttribute('aria-multiline');
			}
			// Ensure only our custom toolbar exists
			field.querySelectorAll('.ql-toolbar:not(.richtext-toolbar)').forEach(t => t.remove());
			toolbar.querySelectorAll('select.ql-header, select').forEach(s => {
				s.style.setProperty('display', 'none', 'important');
				s.hidden = true;
			});
			toolbar.querySelectorAll('.ql-picker').forEach(picker => {
				const control = picker.querySelector('.ql-picker-label');
				const pickerType = Array.from(picker.classList).find(name => name.startsWith('ql-') && name !== 'ql-picker' && name !== 'ql-expanded');
				const source = pickerType ? toolbar.querySelector('select.' + pickerType) : picker.querySelector('select');
				const name = source?.getAttribute('aria-label');
				picker.removeAttribute('aria-label');
				if (control && name) control.setAttribute('aria-label', name);
			});
			const tb = quill.getModule('toolbar');
			if (tb) {
				tb.addHandler('image', () => {
					if (typeof window.HydroOpenMediaPicker === 'function') {
						window.HydroOpenMediaPicker(assetId => {
							const range = quill.getSelection(true);
							quill.insertEmbed(range.index, 'image', '/media/' + encodeURIComponent(assetId));
							quill.setSelection(range.index + 1);
							value.value = JSON.stringify(quill.getContents());
						});
					} else {
						const url = prompt(document.documentElement.lang === 'en' ? 'Enter an image URL:' : 'Görsel URL adresini girin:');
						if (url && url.trim()) {
							const range = quill.getSelection(true);
							quill.insertEmbed(range.index, 'image', url.trim());
							quill.setSelection(range.index + 1);
							value.value = JSON.stringify(quill.getContents());
						}
					}
				});
			}
			const initial = (value.value || '').trim();
			if (initial) {
				try {
					const parsed = JSON.parse(initial);
					quill.setContents(parsed.document || parsed);
				} catch (_) {
					quill.setText(initial);
				}
			}
			editor.__quill = quill;
			quill.on('text-change', () => {
				value.value = JSON.stringify(quill.getContents());
			});

			// Close open pickers on outside click
			if (document.documentElement.dataset.quillPickerGlobalBound !== 'true') {
				document.documentElement.dataset.quillPickerGlobalBound = 'true';
				document.addEventListener('click', e => {
					const target = e.target instanceof Element ? e.target : null;
					if (!target || !target.closest('.ql-picker')) {
						document.querySelectorAll('.ql-picker.ql-expanded').forEach(p => p.classList.remove('ql-expanded'));
					}
				});
			}

			const form = field.closest('form');
			if (form && !form.dataset.quillBound) {
				form.dataset.quillBound = 'true';
				form.addEventListener('submit', () => {
					document.querySelectorAll('[data-richtext]').forEach(f => {
						const v = f.querySelector('.richtext-value');
						const ed = f.querySelector('.richtext-editor');
						if (v && ed && ed.__quill) {
							v.value = JSON.stringify(ed.__quill.getContents());
						}
					});
				});
			}
		} catch (err) {
			console.warn('Quill init error:', err);
			field.querySelectorAll('.ql-toolbar:not(.richtext-toolbar)').forEach(t => t.remove());
			if (editor) editor.__quill = undefined;
			field.removeAttribute('data-quill-init');
		}
	});
};

window.HydroInitItemLists = () => {
	document.querySelectorAll('[data-item-list]').forEach(widget => {
		if (widget.dataset.itemListInit === 'true') return;
		widget.dataset.itemListInit = 'true';
		const hidden = widget.querySelector('.item-list-hidden-input');
		const rowsContainer = widget.querySelector('.item-list-rows');
		const addBtn = widget.querySelector('.item-list-add-btn');
		if (!hidden || !rowsContainer || !addBtn) return;
		const english = document.documentElement.lang === 'en';
		const fieldLabel = widget.dataset.itemListLabel || widget.dataset.fieldName || (english ? 'Item' : 'Madde');
		const inputPlaceholder = widget.dataset.itemListInputPlaceholder || (english ? 'Enter an important item…' : 'Önemli bir madde girin…');
		const itemAriaPattern = widget.dataset.itemListItemAria || '{field} {index}';
		const removeAriaPattern = widget.dataset.itemListRemoveAria || (english ? 'Remove item {index} from {field}' : '{field} {index} maddesini sil');
		const formatRowLabel = (pattern, index) => pattern.replaceAll('{field}', () => fieldLabel).replaceAll('{index}', () => String(index + 1));
		const syncToHidden = () => {
			const inputs = rowsContainer.querySelectorAll('.item-list-input');
			const lines = [];
			inputs.forEach(inp => {
				const val = inp.value.trim();
				if (val) lines.push(val);
			});
			hidden.value = lines.join('\n');
		};
		const createRow = (value = '', autoFocus = false) => {
			const row = document.createElement('div');
			row.className = 'item-list-row';
			const dot = document.createElement('span');
			dot.className = 'item-list-bullet';
			const input = document.createElement('input');
			input.type = 'text';
			input.className = 'item-list-input';
			input.placeholder = inputPlaceholder;
			input.value = value;
			const removeBtn = document.createElement('button');
			removeBtn.type = 'button';
			removeBtn.className = 'item-list-remove-btn';
			removeBtn.setAttribute('aria-label', formatRowLabel(removeAriaPattern, rowsContainer.querySelectorAll('.item-list-row').length));
			removeBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>';
			removeBtn.addEventListener('click', () => {
				row.remove();
				updateIndices();
				syncToHidden();
			});
			input.addEventListener('input', syncToHidden);
			input.addEventListener('keydown', e => {
				if (e.key === 'Enter') {
					e.preventDefault();
					createRow('', true);
				}
			});
			row.append(dot, input, removeBtn);
			rowsContainer.appendChild(row);
			updateIndices();
			if (autoFocus) input.focus();
		};
		const updateIndices = () => {
			const rows = rowsContainer.querySelectorAll('.item-list-row');
			rows.forEach((r, idx) => {
				const dot = r.querySelector('.item-list-bullet');
				if (dot) dot.textContent = String(idx + 1);
				const input = r.querySelector('.item-list-input');
				if (input) input.setAttribute('aria-label', formatRowLabel(itemAriaPattern, idx));
				const removeBtn = r.querySelector('.item-list-remove-btn');
				if (removeBtn) removeBtn.setAttribute('aria-label', formatRowLabel(removeAriaPattern, idx));
			});
		};
		addBtn.addEventListener('click', () => {
			createRow('', true);
			syncToHidden();
		});
		const initialVal = (hidden.value || '').trim();
		if (initialVal) {
			let lines = [];
			try {
				const parsed = JSON.parse(initialVal);
				if (Array.isArray(parsed)) lines = parsed;
			} catch (_) {
				lines = initialVal.split('\n');
			}
			lines.forEach(line => {
				const trimmed = String(line).replace(/^[•\-\*]\s*/, '').trim();
				if (trimmed) createRow(trimmed);
			});
		}
	});
};

window.HydroInitAutoResizeTextareas = () => {
	document.querySelectorAll('textarea').forEach(ta => {
		if (ta.dataset.autoresizeBound === 'true') return;
		ta.dataset.autoresizeBound = 'true';
		const adjust = () => {
			ta.style.height = 'auto';
			ta.style.height = (ta.scrollHeight + 4) + 'px';
		};
		ta.addEventListener('input', adjust);
		if (ta.value) {
			setTimeout(adjust, 20);
		}
	});
};

window.HydroInitMediaSelectPreviews = () => {
	document.querySelectorAll('.media-select-field').forEach(field => {
		if (field.dataset.previewBound === 'true') return;
		field.dataset.previewBound = 'true';
		const input = field.querySelector('input[type="hidden"], input:not([type="button"])');
		const card = field.querySelector('.media-select-card');
		const previewWrap = field.querySelector('.media-select-preview');
		const img = previewWrap ? previewWrap.querySelector('img') : null;
		const nameSpan = field.querySelector('.media-select-name');
		const emptyPlaceholder = previewWrap ? previewWrap.querySelector('.media-select-empty') : null;
		const clearBtn = field.querySelector('.media-select-clear');
		const emptyLabel = document.documentElement.lang === 'en' ? 'No image selected' : 'Görsel seçilmedi';
		const selectedImageLabel = document.documentElement.lang === 'en' ? 'Selected image' : 'Seçilen görsel';
		const emptyText = emptyPlaceholder ? emptyPlaceholder.querySelector('span') : null;
		if (emptyText && emptyText.textContent.trim() === 'Görsel seçilmedi') emptyText.textContent = emptyLabel;
		if (img && img.alt === 'Seçilen görsel') img.alt = selectedImageLabel;

		const syncPreview = () => {
			const val = (input ? input.value : '').trim();
			if (val) {
				field.classList.add('has-selected');
				if (card) card.classList.add('has-selected');
				if (img) {
					img.src = '/media/' + encodeURIComponent(val);
					img.classList.remove('is-hidden');
				}
				if (nameSpan) nameSpan.textContent = val;
				if (clearBtn) clearBtn.classList.remove('is-hidden');
			} else {
				field.classList.remove('has-selected');
				if (card) card.classList.remove('has-selected');
				if (img) {
					img.src = '';
					img.classList.add('is-hidden');
				}
				if (nameSpan && ['Görsel seçilmedi', 'No image selected'].includes(nameSpan.textContent.trim())) nameSpan.textContent = emptyLabel;
				if (clearBtn) clearBtn.classList.add('is-hidden');
			}
		};

		if (input) {
			input.addEventListener('input', syncPreview);
			input.addEventListener('change', syncPreview);
			syncPreview();
		}

		if (clearBtn && input) {
			clearBtn.addEventListener('click', e => {
				e.preventDefault();
				input.value = '';
				input.dispatchEvent(new Event('input', { bubbles: true }));
				input.dispatchEvent(new Event('change', { bubbles: true }));
			});
		}
	});
};

const reinitPage = () => {
	if (window.HydroApplyIcons) window.HydroApplyIcons();
	if (window.HydroInitBlockEditor) window.HydroInitBlockEditor();
	if (window.HydroInitGeneratedBlockEditors) window.HydroInitGeneratedBlockEditors();
	window.HydroInitQuill();
	if (window.HydroInitItemLists) window.HydroInitItemLists();
	if (window.HydroInitAutoResizeTextareas) window.HydroInitAutoResizeTextareas();
	if (window.HydroInitMediaSelectPreviews) window.HydroInitMediaSelectPreviews();
	initSearch();
	setupModals();
	setupFormFeedback();
	window.HydroAnimateKPIs();
	if (window.HydroInitSpotlights) window.HydroInitSpotlights();
	window.dispatchEvent(new CustomEvent('studio:page-changed'));
};

window.studioNavigate = async (url, pushState = true) => {
	try {
		startProgress();
		const res = await fetch(url, { headers: { 'X-Requested-With': 'HydroStudio' } });
		if (!res.ok && res.status === 401) {
			window.location.assign('/login?return_to=' + encodeURIComponent(window.location.pathname));
			return;
		}
		if (res.redirected && !res.url.includes('/admin')) {
			window.location.assign(res.url);
			return;
		}
		const html = await res.text();
		const doc = new DOMParser().parseFromString(html, 'text/html');
		const newMain = doc.querySelector('.studio-main');
		const currentMain = document.querySelector('.studio-main');
		const newSidebar = doc.querySelector('.studio-sidebar');
		const currentSidebar = document.querySelector('.studio-sidebar');
		if (newSidebar && currentSidebar && newSidebar.innerHTML !== currentSidebar.innerHTML) {
			currentSidebar.innerHTML = newSidebar.innerHTML;
		}
		adoptFragmentStyles(newMain);
		document.title = doc.title || document.title;
		currentMain.classList.remove('page-entering');
		// Preserve nonce attributes by moving inert DOM nodes instead of
		// serializing innerHTML, which can hide nonce values.
		currentMain.replaceChildren(...Array.from(newMain.childNodes));
		executeScripts(currentMain);
		updateActiveNav(url);
		reinitPage();
		void currentMain.offsetWidth;
		currentMain.classList.add('page-entering');
		if (pushState) {
			history.pushState({ url }, '', url);
		}
		document.dispatchEvent(new Event('hydro:admin-navigated'));
		window.scrollTo({ top: 0, behavior: 'instant' });
	} catch (err) {
		console.warn('SPA navigation failed, falling back', err);
		window.location.assign(url);
	} finally {
		endProgress();
	}
};

document.addEventListener('click', e => {
	const toggle = e.target.closest('.nav-tree-toggle');
	if (toggle) {
		e.preventDefault();
		e.stopPropagation();
		const tree = toggle.closest('.nav-tree');
		if (tree) {
			const isOpen = tree.classList.toggle('is-open');
			toggle.setAttribute('aria-expanded', String(isOpen));
		}
		return;
	}
	const logoutBtn = e.target.closest('#studio-logout-btn, [data-admin-logout]');
	if (logoutBtn) {
		logoutBtn.disabled = true;
		logoutBtn.textContent = 'Çıkış yapılıyor…';
		const csrfCookie = document.cookie.split('; ').find(row => row.startsWith('hydro_csrf='));
		const csrfVal = csrfCookie ? csrfCookie.split('=')[1] : '';
		fetch('/auth/logout', {
			method: 'POST',
			headers: { 'X-CSRF-Token': csrfVal }
		}).finally(() => {
			try {
				localStorage.removeItem('hydro_member_state');
				sessionStorage.clear();
			} catch (_) {}
			window.location.replace('/login?return_to=/admin');
		});
		return;
	}
	if (e.target.closest('.media-picker-open')) {
		e.preventDefault();
		const opener = e.target.closest('.media-picker-open');
		const targetId = opener.dataset.mediaTarget;
		let target = targetId ? document.getElementById(targetId) : null;
		if (!target) {
			const field = opener.closest('.media-select-field');
			if (field) target = field.querySelector('input[type="hidden"], input:not([type="button"])');
		}
		if (typeof window.HydroOpenMediaPicker === 'function') {
			window.HydroOpenMediaPicker(assetId => {
				if (target) {
					target.value = target.id.startsWith('appearance-') ? '/media/' + encodeURIComponent(assetId) : assetId;
					target.dispatchEvent(new Event('input', { bubbles: true }));
					target.dispatchEvent(new Event('change', { bubbles: true }));
				}
			});
		}
		return;
	}
	if (e.target.closest('[data-media-close]')) {
		const modal = document.getElementById('admin-media-picker-modal') || document.getElementById('media-picker-modal');
		if (modal) modal.hidden = true;
		window.__hydroMediaPickerCallback = null;
		return;
	}
	const choice = e.target.closest('.admin-media-picker-item, .media-picker-item');
	if (choice) {
		const modal = document.getElementById('admin-media-picker-modal') || document.getElementById('media-picker-modal');
		if (modal) modal.hidden = true;
		if (typeof window.__hydroMediaPickerCallback === 'function') {
			window.__hydroMediaPickerCallback(choice.dataset.assetId);
			window.__hydroMediaPickerCallback = null;
		}
		return;
	}
	if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
	const a = e.target.closest('a');
	if (!a || !a.href || a.target === '_blank' || a.hasAttribute('download')) return;
	const url = new URL(a.href, window.location.origin);
	if (url.origin !== window.location.origin) return;
	if (!url.pathname.startsWith('/admin')) return;
	e.preventDefault();
	if (url.href !== window.location.href) {
		window.studioNavigate(url.href, true);
	}
});

window.HydroOpenMediaPicker = cb => {
	const modal = document.getElementById('admin-media-picker-modal') || document.getElementById('media-picker-modal');
	const grid = document.getElementById('admin-media-picker-grid') || document.getElementById('media-picker-grid');
	const searchInput = document.getElementById('admin-media-picker-search');
	if (!modal || !grid) return;
	window.__hydroMediaPickerCallback = cb;
	modal.hidden = false;
	if (searchInput) {
		searchInput.value = '';
	}
	grid.innerHTML = '<div class="studio-skeleton skeleton-card" style="grid-column:1/-1;min-height:90px;display:grid;place-items:center;padding:16px;">Görseller yükleniyor…</div>';
	fetch('/api/media', { headers: { 'Accept': 'application/json' } })
		.then(res => {
			if (!res.ok) throw new Error('library unavailable');
			return res.json();
		})
		.then(assets => {
			const list = Array.isArray(assets) ? assets : [];
			const images = list.filter(asset => asset.status === 'ready' && asset.mime_type && asset.mime_type.indexOf('image/') === 0);
			const render = (query = '') => {
				grid.innerHTML = '';
				const q = (query || '').toLowerCase().trim();
				const filtered = q ? images.filter(img => (img.original_filename || img.id).toLowerCase().includes(q)) : images;
				if (!filtered.length) {
					grid.innerHTML = '<p class="muted" style="grid-column:1/-1;text-align:center;padding:24px 0;">' + (q ? 'Aramanıza uygun görsel bulunamadı.' : 'Seçilebilecek hazır görsel yok. <a href="/admin/media" target="_blank" style="color:var(--brand);font-weight:600;">Medya yükle ↗</a>') + '</p>';
					return;
				}
				filtered.forEach(asset => {
					const card = document.createElement('div');
					card.className = 'admin-media-picker-card';
					const selectBtn = document.createElement('div');
					selectBtn.className = 'admin-media-picker-item';
					selectBtn.setAttribute('role', 'button');
					selectBtn.setAttribute('tabindex', '0');
					selectBtn.dataset.assetId = asset.id;
					selectBtn.title = 'Görseli seç: ' + (asset.original_filename || asset.id);

					const thumbWrap = document.createElement('div');
					thumbWrap.className = 'admin-media-picker-thumb';

					const image = document.createElement('img');
					image.loading = 'lazy';
					image.decoding = 'async';
					image.src = '/media/' + encodeURIComponent(asset.id);
					image.alt = asset.original_filename || asset.id;

					thumbWrap.append(image);

					const name = document.createElement('small');
					name.textContent = asset.original_filename || asset.id;
					name.title = asset.original_filename || asset.id;

					selectBtn.append(thumbWrap, name);

					const select = () => {
						if (typeof cb === 'function') cb(asset.id);
						else if (typeof window.__hydroMediaPickerCallback === 'function') window.__hydroMediaPickerCallback(asset.id);
						modal.hidden = true;
					};
					selectBtn.onclick = select;
					selectBtn.onkeydown = (e) => {
						if (e.key === 'Enter' || e.key === ' ') {
							e.preventDefault();
							select();
						}
					};

					const actions = document.createElement('div');
					actions.className = 'admin-media-picker-actions';

					const viewLink = document.createElement('a');
					viewLink.href = '/media/' + encodeURIComponent(asset.id);
					viewLink.target = '_blank';
					viewLink.rel = 'noopener noreferrer';
					viewLink.className = 'admin-media-picker-action-btn';
					viewLink.title = 'Yeni sekmede önizle';
					viewLink.innerHTML = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>';

					const copyBtn = document.createElement('button');
					copyBtn.type = 'button';
					copyBtn.className = 'admin-media-picker-action-btn';
					copyBtn.title = 'Görsel bağlantısını kopyala';
					copyBtn.innerHTML = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>';
					copyBtn.onclick = (e) => {
						e.stopPropagation();
						e.preventDefault();
						const fullUrl = window.location.origin + '/media/' + encodeURIComponent(asset.id);
						if (navigator.clipboard) {
							navigator.clipboard.writeText(fullUrl);
							copyBtn.style.color = 'var(--green)';
							setTimeout(() => { copyBtn.style.color = ''; }, 1200);
						}
					};

					actions.append(viewLink, copyBtn);
					card.append(selectBtn, actions);
					grid.append(card);
				});
			};
			render();
			if (searchInput) {
				searchInput.oninput = e => render(e.target.value);
			}
		})
		.catch(() => {
			grid.innerHTML = '<p class="muted" style="grid-column:1/-1;text-align:center;padding:24px 0;color:var(--red);">Medya kütüphanesi açılamadı.</p>';
		});
};

window.addEventListener('popstate', () => {
	if (window.location.pathname.startsWith('/admin')) {
		window.studioNavigate(window.location.href, false);
	}
});

if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', () => { initSearch(); reinitPage(); });
} else {
	initSearch();
	reinitPage();
}
})();
