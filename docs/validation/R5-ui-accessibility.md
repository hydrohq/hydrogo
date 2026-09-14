# R5 validation: UI accessibility regressions

## Local evidence

- `go test ./internal/ui`
- `TestAdminShellInteractiveElementsHaveAccessibleNames` parses the rendered
  shell and checks every anchor and button.
- `TestAdminShellAccessibilityCSSContract` checks focus-visible,
  reduced-motion, and keyboard interaction markers.
- The metadata-driven admin table emits a visually-hidden caption and
  `scope="col"` headers; `internal/adminstudio/list_test.go` guards this
  contract.
- Metadata-driven rich-text forms emit `role="textbox"`, `aria-multiline`, and
  a field label before Quill enhances the editor.

## Remaining browser gate

Screen-reader behavior, contrast ratios across real themes, touch target
measurement, and visual regression baselines still require a browser/device
matrix and are not claimed by these source-level tests.
