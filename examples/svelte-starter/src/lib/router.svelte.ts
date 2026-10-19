/** Hash routing: "#/" is the landing page, "#/docs/<section>/<page>/<heading>" a docs page. */
export type Route =
  | { name: 'home' }
  | { name: 'docs'; section: string; page: string; anchor: string };

export function docsHref(section?: string, page?: string, anchor?: string): string {
  if (!section) return '#/docs';
  let href = `#/docs/${section}`;
  if (page) href += `/${page}`;
  if (page && anchor) href += `/${anchor}`;
  return href;
}

export function parseHash(hash: string): Route {
  const parts = hash.replace(/^#\/?/, '').split('/').filter(Boolean).map(decodeURIComponent);
  if (parts[0] === 'docs') {
    return { name: 'docs', section: parts[1] ?? '', page: parts[2] ?? '', anchor: parts[3] ?? '' };
  }
  return { name: 'home' };
}

class Router {
  current = $state<Route>({ name: 'home' });

  init() {
    const sync = () => {
      this.current = parseHash(window.location.hash);
    };
    sync();
    window.addEventListener('hashchange', sync);
  }
}

export const router = new Router();
