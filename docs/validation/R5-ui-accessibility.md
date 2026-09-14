# R5 validation: UI accessibility regressions

## Local evidence

- `go test ./internal/ui`
- `TestAdminShellInteractiveElementsHaveAccessibleNames` parses the rendered
  shell and checks every anchor and button.
- `TestAdminShellAccessibilityCSSContract` checks focus-visible,
  reduced-motion, and keyboard interaction markers.

## Remaining browser gate

Screen-reader behavior, contrast ratios across real themes, touch target
measurement, and visual regression baselines still require a browser/device
matrix and are not claimed by these source-level tests.
