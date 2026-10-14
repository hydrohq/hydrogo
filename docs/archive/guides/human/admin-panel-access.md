# How to Access the Admin Studio (Human Guide)

## Important: No Automatic Admin Account

Neither the local development environment (`hydro run`) nor a fresh live deployment creates an admin account automatically. Demo data (`seed-demo`) **deliberately** creates only a sample member account, not an admin account — there are no default credentials. You must **create the admin account yourself**.

## Creating an Admin Account

If you ask an AI assistant (like Claude Code or Gemini) to create an admin account, it will run the following command for you:

```sh
hydro create-admin --email <email> --password-stdin --config <config.yaml>
```

When successful, you will see `Created admin user: ...`.

If you are on shared hosting without SSH access (as in typical shared hosting setups), you cannot run this command directly in a terminal. Instead, use the one-shot trigger file method described in [`docs/guides/human/deploy-ftp-shared-hosting.md`](deploy-ftp-shared-hosting.md).

## Logging In

- Local development: `http://127.0.0.1:8090/login`
- Live site: `https://your-domain.com/login`

If you navigate to `/admin`, you will be automatically redirected to `/login` if not authenticated.

## Roles

There are three built-in roles: **Admin**, **Editor**, and **Viewer**. Accounts created with `create-admin` always start with the Admin role. To change another user's role, navigate to the user management section in the Admin Studio.

## Two-Factor Authentication (MFA)

Admin Studio supports TOTP-based two-factor authentication (compatible with Google Authenticator, 1Password, etc.). It is optional by default and can be enabled per-user in the panel. It can also be enforced for specific roles via `config.yaml`.

**If you get locked out during local development** (e.g., forgetting your test password or MFA code), the fastest solution is to run `create-admin` with a new email address, or delete the disposable `data/demo.sqlite` file to start fresh.
