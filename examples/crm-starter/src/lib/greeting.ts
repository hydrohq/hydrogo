import type { Lang } from './i18n';

/** "Good morning" / "Good afternoon" / "Good evening" by the hour of day. */
export function greetingKey(hour: number): 'greetMorning' | 'greetAfternoon' | 'greetEvening' {
  return hour < 12 ? 'greetMorning' : hour < 18 ? 'greetAfternoon' : 'greetEvening';
}

/** A friendly first name taken from an e-mail address: "jane.doe@x.com" -> "Jane". */
export function firstName(email: string | null | undefined): string {
  const local = (email ?? '').split('@')[0].split(/[._+-]/)[0];
  return local ? local.charAt(0).toLocaleUpperCase() + local.slice(1) : '';
}

export type { Lang };
