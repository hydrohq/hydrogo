export type ThemeChoice = 'system' | 'light' | 'dark';

const KEY = 'hydro-docs-theme';

function readSaved(): ThemeChoice {
  try {
    const v = localStorage.getItem(KEY);
    if (v === 'light' || v === 'dark' || v === 'system') return v;
  } catch {
    /* storage can be blocked; fall back to system */
  }
  return 'system';
}

class Theme {
  choice = $state<ThemeChoice>('system');
  systemDark = $state(false);
  private media: MediaQueryList | null = null;

  /** True when the page is currently rendered dark. */
  get dark(): boolean {
    return this.choice === 'dark' || (this.choice === 'system' && this.systemDark);
  }

  init() {
    this.choice = readSaved();
    this.media = window.matchMedia('(prefers-color-scheme: dark)');
    this.systemDark = this.media.matches;
    this.media.addEventListener('change', (e) => {
      this.systemDark = e.matches;
      this.apply();
    });
    this.apply();
  }

  set(choice: ThemeChoice) {
    this.choice = choice;
    try {
      localStorage.setItem(KEY, choice);
    } catch {
      /* ignore */
    }
    this.apply();
  }

  /** system -> light -> dark -> system */
  cycle() {
    this.set(this.choice === 'system' ? 'light' : this.choice === 'light' ? 'dark' : 'system');
  }

  private apply() {
    document.documentElement.classList.toggle('dark', this.dark);
  }
}

export const theme = new Theme();
