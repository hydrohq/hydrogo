export type Theme = 'light' | 'dark' | 'system';

class ThemeState {
  current = $state<Theme>('system');
  isDark = $state(false);

  constructor() {
    if (typeof window !== 'undefined') {
      const saved = (localStorage.getItem('hydrogo-theme') as Theme) || 'system';
      this.current = saved;
      this.apply(saved);

      window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (this.current === 'system') {
          this.apply('system');
        }
      });
    }
  }

  setTheme(newTheme: Theme) {
    this.current = newTheme;
    if (typeof window !== 'undefined') {
      localStorage.setItem('hydrogo-theme', newTheme);
      this.apply(newTheme);
    }
  }

  toggle() {
    if (this.isDark) {
      this.setTheme('light');
    } else {
      this.setTheme('dark');
    }
  }

  private apply(t: Theme) {
    if (typeof window === 'undefined') return;

    let dark = false;
    if (t === 'dark') {
      dark = true;
    } else if (t === 'light') {
      dark = false;
    } else {
      dark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    }

    this.isDark = dark;
    if (dark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }
}

export const theme = new ThemeState();
