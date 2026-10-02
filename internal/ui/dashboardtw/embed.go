// Package dashboardtw holds the Tailwind-compiled CSS for the workbench and
// nova dashboard body templates' markup (ADR-0340). Unlike the CSS consts in
// internal/ui, which still target the original class names via @apply, this
// package's output styles the `tw:`-prefixed utility classes added directly
// to adminWorkbenchDashboardTemplate's and adminNovaDashboardTemplate's HTML
// - the original class names (.wb-hero, .nova-stat-card, etc.) remain as
// hooks only, consumed by tests and the existing per-family chrome CSS.
//
// Compiled by the same standalone Tailwind CLI toolchain as ADR-0338/0339
// (not vendored; download it, run scripts/build-dashboard-tailwind.ps1,
// discard it). Scoped to admin_workbench.go and admin_nova_dashboard.go via
// input.css's @source directives, so it never scans the rest of the admin
// shell. No preflight (would reset element defaults page-wide); prefixed
// with "tw:" for consistency with the rest of the Tailwind tooling in this
// repo, though (verified) this package's output defines no unprefixed
// --radius-*/--shadow-* custom property that could collide with the
// ADR-0053 semantic token contract - only var() references to it.
package dashboardtw

import _ "embed"

//go:embed dist/dashboard.css
var CSSText string
