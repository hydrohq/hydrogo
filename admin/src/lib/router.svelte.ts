class Router {
  hash = $state(window.location.hash || '#/');

  constructor() {
    window.addEventListener('hashchange', () => {
      this.hash = window.location.hash || '#/';
    });
  }

  get path(): string {
    const raw = this.hash.replace(/^#/, '');
    const queryIndex = raw.indexOf('?');
    const pathOnly = queryIndex === -1 ? raw : raw.slice(0, queryIndex);
    return pathOnly.startsWith('/') ? pathOnly : `/${pathOnly}`;
  }

  get query(): URLSearchParams {
    const raw = this.hash.replace(/^#/, '');
    const queryIndex = raw.indexOf('?');
    if (queryIndex === -1) return new URLSearchParams();
    return new URLSearchParams(raw.slice(queryIndex + 1));
  }

  navigate(to: string) {
    const target = to.startsWith('#') ? to : `#${to.startsWith('/') ? to : `/${to}`}`;
    if (window.location.hash === target) {
      window.dispatchEvent(new HashChangeEvent('hashchange'));
    } else {
      window.location.hash = target;
    }
  }
}

export const router = new Router();
