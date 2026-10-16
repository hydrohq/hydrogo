class Router {
  currentPath = $state(typeof window !== 'undefined' ? this.extractSubpath() : '/');
  currentSearch = $state(typeof window !== 'undefined' ? window.location.search : '');

  // Unsaved changes protection
  isDirty = $state(false);
  dirtyMessage = $state('Kaydedilmemiş değişiklikleriniz var. Ayrılmak istediğinizden emin misiniz?');
  showUnsavedModal = $state(false);
  pendingTarget = $state<string | null>(null);

  setDirty(dirty: boolean, message?: string) {
    this.isDirty = dirty;
    if (message) this.dirtyMessage = message;
    if (!dirty) {
      this.showUnsavedModal = false;
      this.pendingTarget = null;
    }
  }

  confirmLeave() {
    this.isDirty = false;
    this.showUnsavedModal = false;
    if (this.pendingTarget) {
      const target = this.pendingTarget;
      this.pendingTarget = null;
      this.navigate(target, true);
    }
  }

  cancelLeave() {
    this.showUnsavedModal = false;
    this.pendingTarget = null;
  }

  private extractSubpath(): string {
    if (typeof window === 'undefined') return '/';

    // If an old hash URL is accessed (e.g. /admin/#/system), upgrade to clean path
    if (window.location.hash && window.location.hash.startsWith('#/')) {
      const hashSub = window.location.hash.slice(1);
      const queryIdx = hashSub.indexOf('?');
      const cleanPath = queryIdx === -1 ? hashSub : hashSub.slice(0, queryIdx);
      const cleanSearch = queryIdx === -1 ? '' : hashSub.slice(queryIdx);
      const newUrl = `/admin${cleanPath === '/' ? '' : cleanPath}${cleanSearch}`;
      try {
        window.history.replaceState(null, '', newUrl);
      } catch {}
      return cleanPath.startsWith('/') ? cleanPath : `/${cleanPath}`;
    }

    let pathname = window.location.pathname;
    if (pathname.startsWith('/admin/')) {
      pathname = pathname.slice('/admin'.length);
    } else if (pathname === '/admin') {
      pathname = '/';
    }
    return pathname.startsWith('/') ? pathname : `/${pathname}`;
  }

  constructor() {
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', (e) => {
        if (this.isDirty) {
          e.preventDefault();
          e.returnValue = this.dirtyMessage;
          return this.dirtyMessage;
        }
      });

      window.addEventListener('popstate', () => {
        if (this.isDirty) {
          try {
            window.history.pushState(null, '', `/admin${this.currentPath}${this.currentSearch}`);
          } catch {}
          this.pendingTarget = this.extractSubpath();
          this.showUnsavedModal = true;
          return;
        }
        this.currentPath = this.extractSubpath();
        this.currentSearch = window.location.search;
      });

      window.addEventListener('hashchange', () => {
        if (this.isDirty) {
          this.pendingTarget = this.extractSubpath();
          this.showUnsavedModal = true;
          return;
        }
        this.currentPath = this.extractSubpath();
        this.currentSearch = window.location.search;
      });

      // Intercept anchor clicks inside Admin SPA
      window.addEventListener('click', (e) => {
        const a = (e.target as HTMLElement)?.closest('a');
        if (!a || a.target === '_blank' || a.hasAttribute('download') || e.ctrlKey || e.metaKey || e.shiftKey) {
          return;
        }
        const href = a.getAttribute('href');
        if (!href) return;

        if (href.startsWith('#/')) {
          e.preventDefault();
          this.navigate(href.slice(1));
        } else if (href.startsWith('/admin')) {
          e.preventDefault();
          const targetSub = href.slice('/admin'.length);
          this.navigate(targetSub || '/');
        }
      });
    }
  }

  get path(): string {
    const raw = this.currentPath;
    const queryIndex = raw.indexOf('?');
    const pathOnly = queryIndex === -1 ? raw : raw.slice(0, queryIndex);
    return pathOnly.startsWith('/') ? pathOnly : `/${pathOnly}`;
  }

  get query(): URLSearchParams {
    if (this.currentSearch) {
      return new URLSearchParams(this.currentSearch);
    }
    const raw = this.currentPath;
    const queryIndex = raw.indexOf('?');
    if (queryIndex === -1) return new URLSearchParams();
    return new URLSearchParams(raw.slice(queryIndex + 1));
  }

  // Backwards compatibility getter/setter for tests and older scripts
  get hash(): string {
    return `#${this.currentPath}`;
  }

  set hash(val: string) {
    if (val.startsWith('#')) {
      this.currentPath = val.replace(/^#/, '');
    } else {
      this.currentPath = val;
    }
  }

  navigate(to: string, force = false) {
    let target = to.startsWith('#') ? to.slice(1) : to;
    if (target.startsWith('/admin')) {
      target = target.slice('/admin'.length);
    }
    if (!target.startsWith('/')) {
      target = `/${target}`;
    }

    if (this.isDirty && !force && target !== this.currentPath) {
      this.pendingTarget = target;
      this.showUnsavedModal = true;
      return;
    }

    const queryIndex = target.indexOf('?');
    const pathOnly = queryIndex === -1 ? target : target.slice(0, queryIndex);
    const searchOnly = queryIndex === -1 ? '' : target.slice(queryIndex);

    if (typeof window !== 'undefined') {
      const fullUrl = `/admin${pathOnly === '/' ? '' : pathOnly}${searchOnly}`;
      try {
        if (typeof window.history !== 'undefined' && typeof window.history.pushState === 'function') {
          if (window.location.pathname + window.location.search !== fullUrl) {
            window.history.pushState(null, '', fullUrl);
          }
        }
      } catch {}
    }

    this.currentPath = target;
    this.currentSearch = searchOnly;
  }
}

export const router = new Router();
