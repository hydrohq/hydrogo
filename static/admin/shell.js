// Admin shell behaviour as Alpine.js components. The shell loads the CSP build
// of Alpine (no eval), so every directive in the markup names a property or a
// method registered here; nothing in the HTML is evaluated as arbitrary JS.
// This file must load before /vendor/alpinejs/cdn.min.js (both deferred) so the
// alpine:init listener is in place when Alpine starts.
(() => {
  const english = () => document.documentElement.lang === "en";
  const MOBILE_BREAKPOINT = 840;

  // Moves focus between the menu items of a dropdown with the arrow keys.
  const moveFocus = (container, step) => {
    if (!container) return;
    const items = [...container.querySelectorAll("a[href], button:not([disabled])")];
    if (!items.length) return;
    const current = items.indexOf(document.activeElement);
    const next = current < 0 ? (step > 0 ? 0 : items.length - 1) : (current + step + items.length) % items.length;
    items[next].focus();
  };

  document.addEventListener("alpine:init", () => {
    const Alpine = window.Alpine;

    // Mobile navigation: the toggle (topbar), the backdrop and the sidebar
    // links are separate elements, and the topbar is re-rendered by the studio
    // router on every navigation, so the open state lives in a store.
    Alpine.store("studioNav", {
      open: false,
      set(open) {
        this.open = Boolean(open);
      },
      toggle() {
        this.open = !this.open;
      },
    });
    const nav = () => Alpine.store("studioNav");
    Alpine.effect(() => {
      const open = nav().open;
      document.documentElement.classList.toggle("studio-mobile-nav-open", open);
      document.body?.classList.toggle("studio-mobile-nav-open", open);
    });
    document.addEventListener("keydown", (event) => {
      if (event.key !== "Escape" || !nav().open) return;
      nav().set(false);
      document.querySelector(".studio-mobile-menu-toggle")?.focus();
    });
    window.addEventListener("resize", () => {
      if (window.innerWidth > MOBILE_BREAKPOINT && nav().open) nav().set(false);
    });

    Alpine.data("navToggle", () => ({
      get expanded() {
        return String(nav().open);
      },
      toggle() {
        nav().toggle();
      },
    }));

    Alpine.data("navBackdrop", () => ({
      close() {
        nav().set(false);
      },
    }));

    Alpine.data("studioSidebar", () => ({
      // Following a sidebar link closes the mobile drawer; the studio router
      // then handles the navigation itself.
      follow(event) {
        if (event.target instanceof Element && event.target.closest("a")) nav().set(false);
      },
    }));

    Alpine.data("userMenu", () => ({
      open: false,
      get expanded() {
        return String(this.open);
      },
      get closed() {
        return !this.open;
      },
      toggle() {
        this.open = !this.open;
      },
      close() {
        this.open = false;
      },
      // Escape closes the menu and gives focus back to its button.
      dismiss() {
        if (!this.open) return;
        this.open = false;
        this.$refs.button.focus();
      },
      next() {
        if (!this.open) this.open = true;
        this.$nextTick(() => moveFocus(this.$refs.menu, 1));
      },
      previous() {
        if (!this.open) this.open = true;
        this.$nextTick(() => moveFocus(this.$refs.menu, -1));
      },
    }));

    Alpine.data("notifications", () => ({
      open: false,
      count: 0,
      items: [],
      loading: false,
      failed: false,
      init() {
        this.refresh();
      },
      async refresh() {
        this.loading = true;
        try {
          const response = await fetch("/admin/notifications/unread", { credentials: "same-origin", headers: { Accept: "application/json" } });
          if (!response.ok) throw new Error("notifications unavailable");
          const data = await response.json();
          this.count = Number(data.unread) || 0;
          this.items = (Array.isArray(data.items) ? data.items : []).map((item) => ({
            id: String(item.id || ""),
            title: String(item.title || ""),
            message: String(item.message || ""),
            href: typeof item.href === "string" && item.href.startsWith("/") && !item.href.startsWith("//") ? item.href : "/admin/notifications",
            state: item.read ? "is-read" : "is-unread",
          }));
          this.failed = false;
        } catch (_) {
          this.failed = true;
        } finally {
          this.loading = false;
        }
      },
      get badge() {
        return this.count > 99 ? "99+" : String(this.count);
      },
      get noBadge() {
        return this.count === 0;
      },
      get label() {
        if (this.count === 0) return english() ? "Notifications" : "Bildirimler";
        return english() ? `${this.count} unread notifications` : `${this.count} okunmamış bildirim`;
      },
      get expanded() {
        return String(this.open);
      },
      get closed() {
        return !this.open;
      },
      get empty() {
        return !this.loading && !this.failed && this.items.length === 0;
      },
      toggle() {
        this.open = !this.open;
        if (this.open) this.refresh();
      },
      close() {
        this.open = false;
      },
      dismiss() {
        if (!this.open) return;
        this.open = false;
        this.$refs.button.focus();
      },
      next() {
        moveFocus(this.$refs.panel, 1);
      },
      previous() {
        moveFocus(this.$refs.panel, -1);
      },
    }));

    // Light/dark switch. The pre-paint bootstrap in <head> already applied the
    // saved choice; this keeps data-theme, localStorage and the label in sync.
    Alpine.data("themeToggle", () => ({
      dark: document.documentElement.getAttribute("data-theme") === "dark",
      get variantName() {
        if (english()) return this.dark ? "dark" : "light";
        return this.dark ? "koyu" : "açık";
      },
      get title() {
        return english() ? `HydroGo ${this.variantName} theme` : `HydroGo ${this.variantName} tema`;
      },
      get label() {
        return english() ? `${this.title}; click to change` : `${this.title}; değiştirmek için tıklayın`;
      },
      toggle() {
        this.dark = !this.dark;
        const variant = this.dark ? "dark" : "light";
        document.documentElement.setAttribute("data-theme", variant);
        try {
          localStorage.setItem("hydro-theme", variant);
          localStorage.removeItem("hydro-ui-theme-v2");
          localStorage.removeItem("hydro-ui-theme");
        } catch (_) {}
        if (typeof window.HydroToast === "function") {
          const name = this.variantName.charAt(0).toLocaleUpperCase(english() ? "en" : "tr") + this.variantName.slice(1);
          window.HydroToast(english() ? `${name} theme applied.` : `${name} tema uygulandı.`, "info", english() ? "Appearance updated" : "Görünüm Güncellendi", null, 1500);
        }
      },
    }));
  });
})();
