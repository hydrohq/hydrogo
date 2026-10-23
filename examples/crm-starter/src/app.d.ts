import type { Session } from '$lib/server/sessions';
import type { Tenant } from '$lib/server/tenant';

declare global {
  namespace App {
    interface Locals {
      tenant: Tenant;
      session: Session | null;
      sid: string | null;
      lang: 'en' | 'tr';
    }
  }
}
export {};
