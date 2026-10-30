# HydroGo deployment presets

These configs make the enabled module boundary explicit. Copy a preset, then
set secrets and deployment-specific values through environment variables or a
protected config file.

An omitted module must not add routes, migrations, assets, or workers. The
runtime applies the same module set to migration checks and route registration;
the preset files are intentionally small so a deployment does not start from
the all-modules demonstration config.

- `compact-site.yaml`: pages, media and site publishing on Compact SQLite.
- `editorial.yaml`: compact site plus News scheduling and publishing.
- `commerce.yaml`: compact site plus Commerce catalog and checkout routes.
- `headless.yaml`: API-first pages/site/media surface without member UI.

Use, for example:

```sh
hydro serve --config config/presets/editorial.yaml
```

The files contain no credentials. `HYDRO_DATABASE_DSN` and several other
settings read from the deployment environment (see
`internal/config/load.go`); notification credentials (such as Telegram) have
no environment variable override and must be set in a protected, untracked
copy of the config file instead.
