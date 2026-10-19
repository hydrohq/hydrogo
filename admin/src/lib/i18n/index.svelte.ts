import { api } from '$lib/api/client';
import { en, type MessageKey } from './en';
import { tr } from './tr';

export type Locale = 'en' | 'tr';
export type { MessageKey };

export const SUPPORTED_LOCALES: readonly Locale[] = ['en', 'tr'];
const catalogs: Record<Locale, Record<MessageKey, string>> = { en, tr };

const LAST_USED_KEY = 'hydro.admin.locale';
const userKey = (userId: string) => `${LAST_USED_KEY}.${userId}`;

export function isLocale(value: unknown): value is Locale {
  return typeof value === 'string' && (SUPPORTED_LOCALES as readonly string[]).includes(value);
}

function read(key: string): string | null {
  try {
    return typeof localStorage === 'undefined' ? null : localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string | null) {
  try {
    if (typeof localStorage === 'undefined') return;
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    /* storage unavailable: the choice then lasts for this page load only */
  }
}

function browserLocale(): Locale {
  const preferred = typeof navigator === 'undefined' ? '' : (navigator.language || '').toLowerCase();
  return preferred.startsWith('tr') ? 'tr' : 'en';
}

/**
 * Resolves the admin language: the signed-in user's own choice, else the
 * project default an admin set in Site settings, else the browser's language.
 */
export function resolveLocale(userChoice: string | null, projectDefault: string | null | undefined): Locale {
  if (isLocale(userChoice)) return userChoice;
  if (isLocale(projectDefault)) return projectDefault;
  return browserLocale();
}

export function interpolate(message: string, params?: Record<string, string | number>): string {
  if (!params) return message;
  return message.replace(/\{(\w+)\}/g, (whole, name) => (name in params ? String(params[name]) : whole));
}

class I18nState {
  locale = $state<Locale>('en');
  /** The signed-in user's own choice; null means "follow the project default". */
  userChoice = $state<Locale | null>(null);
  projectDefault = $state<Locale | null>(null);
  private userId = '';

  constructor() {
    const last = read(LAST_USED_KEY);
    this.locale = resolveLocale(last, null);
    this.apply();
  }

  /** Call once the user and the project default are known. */
  init(userId: string, projectDefault: string | null | undefined) {
    this.userId = userId;
    this.projectDefault = isLocale(projectDefault) ? projectDefault : null;
    const stored = userId ? read(userKey(userId)) : null;
    this.userChoice = isLocale(stored) ? stored : null;
    this.locale = resolveLocale(this.userChoice, this.projectDefault);
    this.apply();
  }

  t(key: MessageKey, params?: Record<string, string | number>): string {
    return interpolate(catalogs[this.locale][key] ?? en[key] ?? key, params);
  }

  /** Sets (or, with null, clears) the signed-in user's own language. */
  async setUserLocale(choice: Locale | null) {
    this.userChoice = choice;
    if (this.userId) write(userKey(this.userId), choice);
    this.locale = resolveLocale(choice, this.projectDefault);
    write(LAST_USED_KEY, this.locale);
    this.apply();
    if (choice) {
      // The server keeps a cookie too, so server-rendered admin messages match.
      try {
        await api.put('/api/v1/admin/account/locale', { locale: choice });
      } catch {
        /* the browser already switched; the cookie is a convenience */
      }
    }
  }

  /** Called after an admin saved a new project default. */
  setProjectDefault(value: Locale) {
    this.projectDefault = value;
    this.locale = resolveLocale(this.userChoice, value);
    this.apply();
  }

  private apply() {
    if (typeof document !== 'undefined') document.documentElement.lang = this.locale;
  }
}

export const i18n = new I18nState();

/** Translate a key in the active admin language. Reactive inside templates. */
export function t(key: MessageKey, params?: Record<string, string | number>): string {
  return i18n.t(key, params);
}
