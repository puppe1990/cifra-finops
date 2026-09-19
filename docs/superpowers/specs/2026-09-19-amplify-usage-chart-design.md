# Amplify by usage chart

Date: 2026-09-19
Status: approved (design)
Scope: dashboard section only

## Problem

The dashboard lists spend by service. When "AWS Amplify" shows up, the user cannot
see which billing dimension drives it (build minutes, storage, transfer, SSR,
WAF). Amplify has no per-app cost in Cost Explorer, but the per-usage-type
breakdown already exists in the cost lines Cifra stores.

## Goal

Add a dashboard section that ranks Amplify usage dimensions from most expensive
to cheapest, using the cost lines already loaded for the selected month.

## Non-goals

- Per-app / per-instance Amplify cost (needs cost allocation tags).
- Collecting Amplify apps via `ListApps`.
- New IAM permissions or Cost Explorer calls.

## Approach (A)

1. Filter cost lines where `Service == "AWS Amplify"`.
2. Normalize each `UsageType` by stripping the leading region code so region
   variants merge into one dimension:
   `USE1-BuildDuration` -> `BuildDuration`, `EU-DataTransferOut` -> `DataTransferOut`.
   Implemented as `finops.AmplifyDimension(usageType string) string` using the
   prefix regex `^[A-Z]{2,4}[0-9]*-`. Empty usage type falls back to
   `NoUsageType`.
3. Aggregate cents per dimension, sort desc (ties by name asc), format USD.
4. Reuse `withSpendPct` for the bar width, same as "By service".

## Components

- `internal/finops/amplify.go`: `AmplifyService = "AWS Amplify"`,
  `AmplifyDimension`, `NoUsageType`.
- `internal/handlers/amplify_props.go`: `amplifyProps(lines []models.CostLine) []map[string]any`
  in the same shape as `serviceProps` (`name`, `cents`, `usd`).
- `internal/handlers/dashboard.go`: add `Amplify []map[string]any` to
  `tenantView`; set it in `buildTenantView`; default `"amplify": []any{}` in the
  handler props; `props["amplify"] = withSpendPct(view.Amplify)`.
- `web/templates/pages/dashboard.html`: new `<section>` after "By service",
  rendered only when `.amplify`, with `data-testid="amplify-bar"`.
- i18n: `dash.amplify` = "Amplify by usage" (en) / "Amplify por uso" (pt).

## Data flow

Current month: stored cost lines per account. Past months: Cost Explorer overlay.
Both feed `monthSpend` -> `buildTenantView`, so the section works for any month
without extra requests.

## Testing

- `finops.AmplifyDimension` table test: region prefixes, no prefix, empty.
- `amplifyProps`: aggregation across region variants, desc order, ignores other
  services.
- Dashboard handler test: rendered HTML contains the Amplify section and bars
  when Amplify lines exist; absent otherwise.
- i18n test: `dash.amplify` present in en and pt.

## Risks

- Region prefix regex could over-strip an unusual usage type; test with the
  documented Amplify codes (`BuildDuration`, `DataStorage`, `DataTransferOut`,
  `HostingComputeRequestCount`, `HostingComputeRequestDuration`).
- Only dimensional accuracy is claimed, never per-app attribution.
