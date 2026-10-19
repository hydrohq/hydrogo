// English is the source catalog, assembled from the per-area files in
// messages/. Each area file pairs its English text with the Turkish text and
// is typed so a missing Turkish key is a compile error.
import * as shell from './messages/shell';
import * as auth from './messages/auth';
import * as audit from './messages/audit';
import * as common from './messages/common';
import * as dashboard from './messages/dashboard';
import * as editor from './messages/editor';
import * as news from './messages/news';
import * as pages from './messages/pages';
import * as media from './messages/media';
import * as members from './messages/members';
import * as system from './messages/system';
import * as content from './messages/content';
import * as search from './messages/search';
import * as schema from './messages/schema';
import * as site from './messages/site';

export const en = {
  ...shell.en,
  ...auth.en,
  ...audit.en,
  ...common.en,
  ...dashboard.en,
  ...editor.en,
  ...news.en,
  ...pages.en,
  ...media.en,
  ...members.en,
  ...system.en,
  ...content.en,
  ...search.en,
  ...schema.en,
  ...site.en,
} as const;

export type MessageKey = keyof typeof en;
