import type { Session } from '$lib/server/sessions';

declare global {
  namespace App {
    interface Locals {
      session: Session | null;
      sid: string | null;
      lang: 'en' | 'tr';
    }
  }
}
export {};
