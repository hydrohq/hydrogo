# Frontend state conventions

Every interactive surface exposes one of five states: `idle`, `loading`,
`success`, `empty`, or `error`. Loading and empty states are not errors. Error
responses carry a stable machine-readable `error_code` and a short,
localizable `user_message`; infrastructure details stay in server logs.

Optimistic updates are explicit. A snapshot may be marked `optimistic` only
while loading, after a tentative success, or while showing a recoverable error.
The UI must preserve the last confirmed value, disable duplicate submission,
and provide a retry path when the state is `error`. These rules apply equally
to server-rendered forms, HTMX fragments, and a future separate frontend.

`ui.FrontendStateSnapshot.Validate` enforces the structural portion of this
contract before a state is serialized or handed to a renderer.
