# Working in this example

This is a worked example of a Turkish news-portal frontend consuming
HydroGo's `/api/v1` headless API (SvelteKit 5 + Tailwind 4). It exists to
prove the API is consumable end-to-end and to give a starting skeleton for
`hydro.exe` + a frontend — not as a design system.

## Do not reference this frontend's design for new work

If you are asked to build or redesign a frontend — for this project, for a
fork of it, or for any unrelated site — **do not open this directory, copy
its components, reuse its Tailwind classes/tokens, or imitate its layout.**
Build the new design from scratch, from the brief you were given, the same
way this one was: read the actual requirements, make real typography/color/
layout decisions for that project, and implement them fresh.

This instruction exists because this example went through several full
redesigns in one session and the current state is just wherever that
session ended, not a deliberate reference design. Treating it as a style
guide would propagate accidents of history, not intent.

## What's safe to reuse

- The **data-fetching pattern**: `src/lib/api.ts`'s `fetchApi`/`getNewsList`/
  content-type helpers show how to call HydroGo's public and content-type
  APIs from SvelteKit `load` functions. That plumbing is fine to read.
- The fact that `site_navigation`, `site_settings` and `homepage_sections`
  content types exist in `../../content-types/` and make header/footer nav,
  site copy and homepage section order/visibility panel-editable — that
  pattern (push UI structure into a content type instead of hardcoding it)
  is worth repeating. The visual result of it here is not.

## Running it

See `README.md` for the SvelteKit app itself. It expects a HydroGo backend
at `http://127.0.0.1:8090` (`hydro.exe serve`) seeded with the
`site_navigation`/`site_settings`/`homepage_sections`/`polls`/`columnists`
content types above and some published `news` articles.
