import type { MessageKey } from './en';
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

export const tr: Record<MessageKey, string> = {
  ...shell.tr,
  ...auth.tr,
  ...audit.tr,
  ...common.tr,
  ...dashboard.tr,
  ...editor.tr,
  ...news.tr,
  ...pages.tr,
  ...media.tr,
  ...members.tr,
  ...system.tr,
  ...content.tr,
  ...search.tr,
  ...schema.tr,
  ...site.tr,
};
