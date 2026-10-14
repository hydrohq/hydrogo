# Optional Social Sign-In

Social sign-in is disabled unless at least one provider is configured. It adds
two browser routes per configured provider and does not start a worker or
maintain an in-memory token cache.

Configure each provider with `HYDRO_OAUTH_<PROVIDER>_CLIENT_ID`,
`HYDRO_OAUTH_<PROVIDER>_CLIENT_SECRET`, and
`HYDRO_OAUTH_<PROVIDER>_REDIRECT_URL`. Set `HYDRO_OAUTH_STATE_KEY` to a
deployment secret containing at least 32 bytes. Register the exact callback
URL with the provider and use HTTPS in deployed environments. Google and
GitHub use the authorization-code flow with PKCE; Apple identity tokens are
signature-checked against Apple's published keys.

The callback links only a verified provider email to an already active HydroGo
administrator/editor account in the current tenant. It does not provision
accounts or store provider access tokens. Mobile OAuth exchange and member
profile provisioning remain disabled pending their application-service and
session contracts. Existing password login is unchanged.

The identity mapping migration is additive for SQLite and PostgreSQL. It stores
the provider subject, HydroGo user ID, tenant ID, and creation time. Removing
the mapping table requires backing up linked identities first. No session,
media, background-job, or connection-pool behavior changes.
