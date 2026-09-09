# Architecture

Cifra is a Cais app. HTTP handlers talk to a SQLite `Store`. AWS is reached only through `awsinv.Collector`. Hetzner Cloud is reached only through `hetznerinv.Collector`.

```
browser (Inertia/Svelte)
        │
   handlers (workspace scoped)
        │
   store (tenant_id on every query)
        │
   syncer → awsinv.Collector / hetznerinv.Collector (Live or stub)
```

## Tenancy

- `tenants` + `tenant_members`
- `users.active_tenant_id` is the current workspace
- Cloud accounts, resources, cost lines, findings, and budgets belong to a tenant
- `ListResourcesForTenant` joins through `cloud_accounts` so a tenant never sees another tenant's rows

## Seeded account

`seed.EnsurePrimaryWorkspace` creates tenant `demo`. It attaches an AWS account **only** when `CIFRA_SEED_AWS_ACCOUNT_ID` is a 12-digit id (local `.env`, never committed).

## Cost model

- Prefer Cost Explorer unblended monthly totals grouped by SERVICE
- If CE is denied, sum Lightsail bundle prices + idle static IPs + S3 standard storage
- Amounts are integer USD cents
- Dashboard prorates estimates by day-of-month; CE amounts are treated as month-to-date

## Secrets

Access keys stored on a cloud account are AES-GCM sealed with `APP_SECRET`. The default chain never stores secrets.
