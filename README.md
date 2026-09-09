# Cifra — multi cloud FinOps

Multi-tenant FinOps, built with [amarra-cais](https://github.com/puppe1990/amarra-cais) (Go + Amarra Views + Drive + SQLite).

The app UI is available in **English** and **Brazilian Portuguese**. Use the **EN / PT** control in the sidebar (and on the public pages) to switch. The choice is stored in the `cifra_locale` cookie.

## What it does

- Isolates customers in workspaces (tenants): accounts, inventory, budgets, and findings never leak
- Syncs Cost Explorer when IAM allows `ce:GetCostAndUsage`
- Without CE, estimates monthly burn from inventory: Lightsail (instances + IPs) and S3
- Shows the minimum IAM policy needed for real billing data
- Lets each workspace link more AWS accounts (default credential chain or encrypted access keys)

Development login: `demo@example.com` / `password`.

## Stack

- Go 1.26 + [amarra-cais](https://github.com/puppe1990/amarra-cais) (Amarra Views + Drive)
- Tailwind 3 (no Vite / Svelte)
- SQLite
- AWS SDK v2 (Cost Explorer, Lightsail, S3, STS)

## Quick start

```bash
export PATH="$HOME/go/bin:$PATH"
export LOCALE=en
cp .env.example .env   # set CIFRA_SEED_AWS_ACCOUNT_ID if you want a seeded account
amarra-cais install
amarra-cais dev        # http://localhost:8080
```

The first visit to the ledger syncs accounts on the active workspace (local `~/.aws` chain or tenant access keys).

## Seed your account (optional, local only)

Do not commit account IDs. In `.env` (gitignored):

```
CIFRA_SEED_AWS_ACCOUNT_ID=xxxxxxxxxxxx
```

Without that variable the Demo workspace starts empty and you link the account in the UI.

## Recommended IAM

Paste the policy from `/settings` onto a FinOps read role. Without `ce:GetCostAndUsage`, Cifra estimates from inventory.

## Tests

```bash
go test ./... -count=1
```
