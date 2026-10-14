package shell

// Stylesheets are linked in this order; later files override earlier ones.
var Stylesheets = []string{
	"/vendor/quill/quill.snow.css",
	"/admin-assets/css/fonts.css",
	"/admin-assets/css/base.css",
	"/admin-assets/css/overrides.css",
	"/admin-assets/css/dashboard.css",
	"/admin-assets/css/polish.css",
	"/admin-assets/css/nav-tree.css",
	"/admin-assets/css/richtext.css",
	"/admin-assets/css/item-list.css",
	"/admin-assets/css/form-layout.css",
	"/admin-assets/css/icons.css",
	"/admin-assets/css/search.css",
	"/admin-assets/css/toast.css",
	"/admin-assets/css/areas.css",
	"/admin-assets/css/hydrogo-theme.css",
	"/admin-assets/css/hydrogo-dashboard.css",
	"/admin-assets/css/form-controls.css",
	"/admin-assets/css/kit.css",
	"/admin-assets/css/media.css",
	"/admin-assets/css/editor.css",
	"/admin-assets/css/spotlight.css",
	"/admin-assets/css/responsive.css",
	"/admin-assets/css/ux.css",
	"/admin-assets/css/avatar-contrast.css",
	"/admin-assets/admin.css",
}

// Scripts are loaded with defer, so they run in this order once the document
// is parsed. shell.js registers the Alpine components and must come right
// before Alpine itself.
var Scripts = []string{
	"/vendor/htmx/htmx.min.js",
	"/admin-assets/motion.js",
	"/admin-assets/js/block-editor.js",
	"/admin-assets/js/icons.js",
	"/admin-assets/js/studio.js",
	"/admin-assets/js/ux.js",
	"/admin-assets/js/reset-link.js",
	"/admin-assets/js/draft-autosave.js",
	"/admin-assets/js/password-strength.js",
	"/admin-assets/js/password-submit.js",
	"/admin-assets/js/editor-stats.js",
	"/admin-assets/js/page-weight.js",
	"/admin-assets/js/status-filter.js",
	"/admin-assets/js/list-state.js",
	"/admin-assets/js/list-view.js",
	"/admin-assets/js/copy-text.js",
	"/admin-assets/js/hotkeys.js",
	"/admin-assets/js/staff-notice.js",
	"/admin-assets/js/realtime-chime.js",
	"/admin-assets/js/device-preview.js",
	"/admin-assets/js/tour.js",
	"/admin-assets/js/guide.js",
	"/admin-assets/js/session-warning.js",
	"/admin-assets/js/external-links.js",
	"/admin-assets/js/developer-console.js",
	"/admin-assets/js/admin-control.js",
	"/admin-assets/js/reports.js",
}

// EnglishScript translates page copy that has no English version yet; it is
// loaded only for the English admin locale.
const EnglishScript = "/admin-assets/js/english-ui.js"

// AlpineScripts come last: the component definitions, then Alpine (CSP build).
var AlpineScripts = []string{
	"/admin-assets/shell.js",
	"/vendor/alpinejs/cdn.min.js",
}

// QuillScript stays a classic blocking script: page scripts may construct
// editors while the document is still parsing.
const QuillScript = "/vendor/quill/quill.js"
