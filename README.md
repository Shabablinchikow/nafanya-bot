# nafanya-bot

Щитпост бот для Telegram на основе ChatGPT

## Deployment (Scaleway, since 2026-09)

Runs on the Kapsule cluster from [`shaba-scaleway`](../shaba-scaleway), namespace `default`.
Push to `master` builds `rg.nl-ams.scw.cloud/shaba/nafanya-bot:<sha>` and runs
`helm upgrade` via `.github/workflows/build-docker.yml` — no manual step.

- Cluster access in CI: `scaleway/action-scw` + `scw k8s kubeconfig install`, creds in the
  `SCW_*` repo secrets (IAM application `github-actions`, managed in shaba-scaleway).
- Image pull: `imagePullSecrets: [shaba]` in `values.yaml`; the secret is created in every
  app namespace by `shaba-scaleway/k8s-bootstrap`.
- Database: Scaleway RDB (PostgreSQL 17), db/user `nafanya`, reached over the private network
  (`DB_HOST`/`DB_PORT`/`DB_SSLMODE` repo vars, `DB_PASS` secret). `sslmode=require`.
- Secrets are passed to helm through `env`, not inline `${{ }}` — the DB password has shell
  metacharacters, and the chart quotes `db_pass`.
