package ui

// AdminHydroGoCSS gives "hydrogo" its own distinct flagship visual identity
// (ADR-0341), replacing the undifferentiated base tokens it silently
// inherited from AdminCSS before this change (hydrogo and the former
// "workbench" family were, in effect, the same unstyled default). It
// restyles the shared, mode-agnostic sidebar/topbar/card/button/input markup
// the same way AdminShadcnCSS/AdminNovaCSS/AdminAuroraCSS do (ADR-0041) -
// only hydrogo's *dashboard body* gets its own isolated Tailwind-authored
// markup (internal/ui/hydrogoui), matching the shadcn/Aurora precedent.
//
// Design brief executed here: one confident, saturated brand hue (a cobalt
// blue distinct from the old default sky-blue, and deliberately not a
// purple-blue gradient, not teal, not monochrome), soft real elevation via
// brand-tinted ambient shadows (not flat, not neobrutalist hard-offset
// shadows), slightly more generous corner radii than the shared default for
// a confident rounded-lg/xl feel, and a light-first palette with a proper
// dark companion (not an afterthought invert).
const AdminHydroGoCSS = `html[data-ui-theme=hydrogo]{--bg:#f7f8fb;--surface:#fff;--surface-raised:#fff;--surface-hover:#f1f4fb;--bg-subtle:#eef1f8;--ink:#101425;--ink-secondary:#3c4255;--muted:#596579;--line:#e4e7f1;--line-light:#d3d8e8;--brand:#2954ff;--brand-hover:#1e40e0;--brand-light:#2954ff;--brand-subtle:#eaefff;--brand-glow:#2954ff29;--brand-ink:#fff;--green:#087443;--green-subtle:#e8f8f0;--red:#e0293d;--red-subtle:#fdecee;--amber:#8a4b08;--amber-subtle:#fdf3e3;--purple:#7c5cff;--purple-subtle:#f1edff;--radius-xs:7px;--radius-sm:10px;--radius-md:14px;--radius-lg:18px;--radius-xl:28px;--shadow-xs:0 1px 1px #10142508;--shadow-sm:0 1px 2px #2954ff0f,0 1px 2px #10142508;--shadow-md:0 10px 28px -8px #2954ff1f,0 3px 8px #1014250d;--shadow-lg:0 24px 56px -16px #2954ff26,0 10px 20px #10142512;--card-highlight:inset 0 1px 0 #ffffffb3}
html[data-ui-theme=hydrogo][data-theme=dark]{--bg:#0a0d16;--surface:#12162380;--surface:#121623;--surface-raised:#161b2a;--surface-hover:#1a2032;--bg-subtle:#0e1220;--ink:#eef1fa;--ink-secondary:#c3c9dc;--muted:#8992ad;--line:#232a3d;--line-light:#313a53;--brand:#6b89ff;--brand-hover:#88a0ff;--brand-light:#6b89ff;--brand-subtle:#6b89ff22;--brand-glow:#6b89ff3d;--brand-ink:#0a0d16;--green:#36c98a;--green-subtle:#36c98a1f;--red:#f0566a;--red-subtle:#f0566a1f;--amber:#e0a63a;--amber-subtle:#e0a63a1f;--purple:#9d8bff;--purple-subtle:#9d8bff1f;--shadow-xs:0 1px 1px #00000040;--shadow-sm:0 1px 3px #00000059;--shadow-md:0 14px 32px -10px #000000a3,0 0 0 1px #6b89ff14;--shadow-lg:0 30px 64px -18px #000000bf;--card-highlight:inset 0 1px 0 #6b89ff1a}
html[data-ui-theme=hydrogo] body{background-color:var(--bg);color:var(--ink)}
html[data-ui-theme=hydrogo] .studio-sidebar{background-color:var(--surface);border-right:1px solid var(--line)}
html[data-ui-theme=hydrogo] .sidebar-header{border-bottom-color:var(--line)}
html[data-ui-theme=hydrogo] .brand-logo-mark{background-image:none;background-color:var(--brand);box-shadow:none;border-radius:var(--radius-md)}
html[data-ui-theme=hydrogo] .brand-text span b{color:var(--brand)}
html[data-ui-theme=hydrogo] .nav-item.is-active{background-color:var(--brand-subtle)!important;color:var(--brand)!important;box-shadow:none!important}
html[data-ui-theme=hydrogo][data-theme=dark] .nav-item.is-active{color:var(--brand)!important;background-color:var(--brand-subtle)!important}
html[data-ui-theme=hydrogo] .sidebar-bottom{background-color:var(--bg-subtle);border-top:1px solid var(--line)}
html[data-ui-theme=hydrogo] .studio-main{background-image:none}
html[data-ui-theme=hydrogo][data-theme=dark] .studio-main{background-image:radial-gradient(ellipse 70% 45% at 50% -10%,#6b89ff14,transparent)}
html[data-ui-theme=hydrogo] .studio-topbar{background-color:var(--surface);border-bottom:1px solid var(--line)}
html[data-ui-theme=hydrogo] .studio-card,html[data-ui-theme=hydrogo] .dashboard-card,html[data-ui-theme=hydrogo] .kpi-card,html[data-ui-theme=hydrogo] .meta-item{box-shadow:var(--shadow-sm);border-color:var(--line)}
html[data-ui-theme=hydrogo] .studio-card:hover,html[data-ui-theme=hydrogo] .kpi-card:hover{box-shadow:var(--shadow-md);border-color:var(--line-light)}
html[data-ui-theme=hydrogo] .user-avatar{background-color:var(--brand);color:var(--brand-ink)}
html[data-ui-theme=hydrogo] .studio-form button,html[data-ui-theme=hydrogo] .button,html[data-ui-theme=hydrogo] .studio-card button{box-shadow:0 1px 2px #2954ff33}
html[data-ui-theme=hydrogo][data-theme=dark] .studio-form button,html[data-ui-theme=hydrogo][data-theme=dark] .button,html[data-ui-theme=hydrogo][data-theme=dark] .studio-card button{box-shadow:0 1px 2px #00000059}
`

// Initials avatars use a fixed palette independent of the active Admin theme.
// Theme-specific .user-avatar rules must not replace those colors: in dark
// mode their --brand token may be too light for white initials, while
// monochrome themes set --brand-ink to near-black for white primary buttons.
const AdminAvatarContrastCSS = `html[data-ui-theme] .user-avatar.ds-avatar.ds-avatar--0{background-color:#315bd6;color:#fff}
html[data-ui-theme] .user-avatar.ds-avatar.ds-avatar--1{background-color:#087f8c;color:#fff}
html[data-ui-theme] .user-avatar.ds-avatar.ds-avatar--2{background-color:#7c3aed;color:#fff}
html[data-ui-theme] .user-avatar.ds-avatar.ds-avatar--3{background-color:#b45309;color:#fff}
html[data-ui-theme] .user-avatar.ds-avatar.ds-avatar--4{background-color:#be123c;color:#fff}
html[data-ui-theme] .user-avatar.ds-avatar.ds-avatar--5{background-color:#047857;color:#fff}
html[data-ui-theme] .user-avatar.ds-avatar.ds-avatar--6{background-color:#4338ca;color:#fff}
html[data-ui-theme] .user-avatar.ds-avatar.ds-avatar--7{background-color:#a21caf;color:#fff}`

// AdminShadcnDashboardLayoutCSS toggles which dashboard body markup is
// visible for the shadcn family, following the same client-only
// display-toggle pattern AdminNovaDashboardLayoutCSS/
// AdminAuroraDashboardLayoutCSS established (ADR-0041's family selection has
// no server-side signal to branch on, so every family's dashboard body
// renders unconditionally and CSS hides the inactive ones). hydrogo has no
// self-hiding rule of its own - it is the default/flagship family and stays
// visible unless another family explicitly hides it, exactly like the old
// "workbench" family it replaced.
const AdminShadcnDashboardLayoutCSS = `html:not([data-ui-theme=shadcn]) .shadcn-dashboard{display:none}
html[data-ui-theme=shadcn] .hydrogo-dashboard{display:none}`
