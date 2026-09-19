# Amplify by usage chart — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a dashboard section ranking AWS Amplify spend by usage dimension (Build, Storage, DataTransferOut, SSR request count/duration, WAF) from most expensive to cheapest.

**Architecture:** Reuse the cost lines already computed per selected month. A pure `finops.AmplifyDimension` strips the CE region prefix from `USAGE_TYPE`. A new `handlers.amplifyProps` aggregates Amplify lines by dimension and sorts desc. `buildTenantView` exposes it; the dashboard template renders a bar list using the existing `withSpendPct` widths.

**Tech Stack:** Go 1.26, Amarra Views (server-rendered `html/template`), SQLite, existing CE cost lines. No new IAM or AWS calls.

**Note on commits:** The repo owner has asked not to commit unless explicitly requested. Skip the "Commit" steps below unless the user says otherwise.

---

### Task 1: `finops.AmplifyDimension`

**Files:**
- Create: `internal/finops/amplify.go`
- Test: `internal/finops/amplify_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/finops/amplify_test.go`:

```go
package finops

import "testing"

func TestAmplifyDimension_stripsRegionPrefix(t *testing.T) {
	cases := map[string]string{
		"USE1-BuildDuration":           "BuildDuration",
		"EU-DataTransferOut":           "DataTransferOut",
		"APS3-BuildDuration":           "BuildDuration",
		"APN1-DataStorage":             "DataStorage",
		"HostingComputeRequestCount":   "HostingComputeRequestCount",
		"USE1-HostingComputeRequestDuration": "HostingComputeRequestDuration",
		"":                             "NoUsageType",
	}
	for in, want := range cases {
		if got := AmplifyDimension(in); got != want {
			t.Errorf("AmplifyDimension(%q) = %q, want %q", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/finops/ -run TestAmplifyDimension -v`
Expected: FAIL — `undefined: AmplifyDimension`

- [ ] **Step 3: Write minimal implementation**

Create `internal/finops/amplify.go`:

```go
package finops

import "regexp"

const (
	// AmplifyService is the Cost Explorer SERVICE value for AWS Amplify.
	AmplifyService = "AWS Amplify"
	// NoUsageType is the dimension used when a cost line has no USAGE_TYPE.
	NoUsageType = "NoUsageType"
)

// amplifyRegionPrefix matches the region code CE prepends to some Amplify usage
// types, e.g. USE1-BuildDuration, EU-DataTransferOut, APS3-BuildDuration.
var amplifyRegionPrefix = regexp.MustCompile(`^[A-Z]{2,4}[0-9]*-`)

// AmplifyDimension collapses region variants into one billing dimension:
// USE1-BuildDuration -> BuildDuration.
func AmplifyDimension(usageType string) string {
	if usageType == "" {
		return NoUsageType
	}
	return amplifyRegionPrefix.ReplaceAllString(usageType, "")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/finops/ -run TestAmplifyDimension -v`
Expected: PASS

---

### Task 2: `handlers.amplifyProps`

**Files:**
- Create: `internal/handlers/amplify_props.go`
- Test: `internal/handlers/amplify_props_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/handlers/amplify_props_test.go`:

```go
package handlers

import (
	"testing"

	"github.com/puppe1990/aws-finops/internal/finops"
	"github.com/puppe1990/aws-finops/internal/models"
)

func TestAmplifyProps_groupsByDimensionSortedDesc(t *testing.T) {
	got := amplifyProps([]models.CostLine{
		{Service: finops.AmplifyService, UsageType: "USE1-BuildDuration", MonthlyCents: 900, Source: finops.SourceCE},
		{Service: finops.AmplifyService, UsageType: "EU-BuildDuration", MonthlyCents: 100, Source: finops.SourceCE},
		{Service: finops.AmplifyService, UsageType: "USE1-DataStorage", MonthlyCents: 300, Source: finops.SourceCE},
		{Service: "Amazon Lightsail", UsageType: "StaticIp", MonthlyCents: 5000, Source: finops.SourceCE},
	})
	if len(got) != 2 {
		t.Fatalf("dimensions = %d, want 2: %#v", len(got), got)
	}
	if got[0]["name"] != "BuildDuration" || got[0]["cents"] != int64(1000) || got[0]["usd"] != "US$ 10,00" {
		t.Fatalf("first = %#v", got[0])
	}
	if got[1]["name"] != "DataStorage" || got[1]["cents"] != int64(300) {
		t.Fatalf("second = %#v", got[1])
	}
}

func TestAmplifyProps_emptyWhenNoAmplifyLines(t *testing.T) {
	got := amplifyProps([]models.CostLine{
		{Service: "Amazon Lightsail", MonthlyCents: 1200, Source: finops.SourceEstimate},
	})
	if len(got) != 0 {
		t.Fatalf("got = %#v, want empty", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handlers/ -run TestAmplifyProps -v`
Expected: FAIL — `undefined: amplifyProps`

- [ ] **Step 3: Write minimal implementation**

Create `internal/handlers/amplify_props.go`:

```go
package handlers

import (
	"sort"

	"github.com/puppe1990/aws-finops/internal/finops"
	"github.com/puppe1990/aws-finops/internal/models"
)

// amplifyProps ranks Amplify usage dimensions from most to least expensive.
func amplifyProps(lines []models.CostLine) []map[string]any {
	sums := map[string]int64{}
	var order []string
	for _, line := range lines {
		if line.Service != finops.AmplifyService {
			continue
		}
		dim := finops.AmplifyDimension(line.UsageType)
		if _, ok := sums[dim]; !ok {
			order = append(order, dim)
		}
		sums[dim] += line.MonthlyCents
	}
	sort.Slice(order, func(i, j int) bool {
		if sums[order[i]] == sums[order[j]] {
			return order[i] < order[j]
		}
		return sums[order[i]] > sums[order[j]]
	})
	out := make([]map[string]any, 0, len(order))
	for _, name := range order {
		out = append(out, map[string]any{
			"name": name, "cents": sums[name], "usd": formatUSD(sums[name]),
		})
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/handlers/ -run TestAmplifyProps -v`
Expected: PASS

---

### Task 3: i18n label `dash.amplify`

**Files:**
- Modify: `internal/i18n/en.go` (near `dash.by_service`, line ~94)
- Modify: `internal/i18n/pt.go` (near `dash.by_service`, line ~94)
- Test: `internal/i18n/i18n_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/i18n/i18n_test.go`:

```go
func TestLabels_amplifyByUsage(t *testing.T) {
	en := Labels("en")
	pt := Labels("pt-BR")
	if en["dash.amplify"] != "Amplify by usage" {
		t.Fatalf("en dash.amplify = %q", en["dash.amplify"])
	}
	if pt["dash.amplify"] != "Amplify por uso" {
		t.Fatalf("pt dash.amplify = %q", pt["dash.amplify"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/i18n/ -run TestLabels_amplifyByUsage -v`
Expected: FAIL — en dash.amplify = ""

- [ ] **Step 3: Add the keys**

In `internal/i18n/en.go`, after `"dash.by_service":        "By service",` add:

```go
	"dash.amplify":           "Amplify by usage",
```

In `internal/i18n/pt.go`, after `"dash.by_service":        "Por serviço",` add:

```go
	"dash.amplify":           "Amplify por uso",
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/i18n/ -run TestLabels_amplifyByUsage -v`
Expected: PASS

---

### Task 4: Expose `amplify` in dashboard view and props

**Files:**
- Modify: `internal/handlers/dashboard.go` (`tenantView` ~line 122, `buildTenantView` ~line 166, props default ~line 49, props assignment ~line 108)

- [ ] **Step 1: Add the view field**

In `internal/handlers/dashboard.go`, add `Amplify` to `tenantView`:

```go
type tenantView struct {
	Summary   map[string]any
	Services  []map[string]any
	Amplify   []map[string]any
	Resources []map[string]any
	Findings  []map[string]any
	Budgets   []map[string]any
	Accounts  []map[string]any
	LastSync  map[string]any
}
```

- [ ] **Step 2: Populate it in `buildTenantView`**

In the returned `tenantView{...}` literal, add `Amplify: amplifyProps(costLines),` right after `Services:  serviceProps(costLines),`:

```go
		Services:  serviceProps(costLines),
		Amplify:   amplifyProps(costLines),
```

- [ ] **Step 3: Add the props default and assignment**

In `ServeHTTP`, add `"amplify": []any{},` to the initial props map (after `"services":      []any{},`):

```go
		"services":      []any{},
		"amplify":       []any{},
```

After `props["services"] = withSpendPct(view.Services)`, add:

```go
	props["services"] = withSpendPct(view.Services)
	props["amplify"] = withSpendPct(view.Amplify)
```

- [ ] **Step 4: Build and run existing dashboard tests**

Run: `go test ./internal/handlers/ -run TestDashboardHandler -count=1`
Expected: PASS (all existing dashboard tests still green)

---

### Task 5: Dashboard template section + handler test

**Files:**
- Modify: `web/templates/pages/dashboard.html` (after the "By service" section, line 44)
- Test: `internal/handlers/dashboard_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/handlers/dashboard_test.go`:

```go
func TestDashboardHandler_amplifySectionGroupsUsage(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	tenant, _ := s.FindTenantBySlug(finops.PrimaryTenantSlug)
	accounts, _ := s.ListCloudAccounts(tenant.ID)
	_ = s.ReplaceCostLines(accounts[0].ID, []models.CostLine{
		{Service: finops.AmplifyService, UsageType: "USE1-BuildDuration", MonthlyCents: 721, Source: finops.SourceCE},
		{Service: finops.AmplifyService, UsageType: "EU-BuildDuration", MonthlyCents: 100, Source: finops.SourceCE},
		{Service: finops.AmplifyService, UsageType: "USE1-DataStorage", MonthlyCents: 200, Source: finops.SourceCE},
	})

	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t))
	h.now = func() time.Time { return time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC) }

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `data-testid="amplify-bar"`) {
		t.Fatalf("missing amplify bar: %s", body)
	}
	if !strings.Contains(body, "BuildDuration") {
		t.Fatalf("missing BuildDuration dimension: %s", body)
	}
	if !strings.Contains(body, "US$ 8,21") {
		t.Fatalf("missing amortized BuildDuration total: %s", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handlers/ -run TestDashboardHandler_amplifySectionGroupsUsage -v`
Expected: FAIL — `missing amplify bar`

- [ ] **Step 3: Add the template section**

In `web/templates/pages/dashboard.html`, after the closing `</section>` of "By service" (line 44) and before `{{ if .findings }}`, insert:

```html
{{ if .amplify }}
<section class="mt-16">
  <h3 class="font-display text-2xl">{{ index .labels "dash.amplify" }}</h3>
  <ul class="mt-6 space-y-8">
    {{ range .amplify }}
    <li class="border-b border-paper-200/10 pb-8 last:border-b-0 last:pb-0">
      <div class="flex items-baseline justify-between gap-4">
        <h4 class="font-display text-xl leading-tight">{{ .name }}</h4>
        <p class="shrink-0 font-mono text-lg text-copper-400">{{ .usd }}</p>
      </div>
      <div class="mt-4 w-full bg-ink-800" style="height: 0.45rem"><div class="h-full bg-copper-400" data-testid="amplify-bar" style="width: {{ .pct }}%"></div></div>
    </li>
    {{ end }}
  </ul>
</section>
{{ end }}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/handlers/ -run TestDashboardHandler_amplifySectionGroupsUsage -v`
Expected: PASS

---

### Task 6: Full verification

- [ ] **Step 1: Run the whole suite with race detector**

Run: `make test`
Expected: all packages `ok`, no race reports.

- [ ] **Step 2: Format check and lint**

Run: `gofmt -l internal cmd && golangci-lint run ./...`
Expected: no files listed by `gofmt`, then `0 issues.`

- [ ] **Step 3: Manual smoke (optional)**

Run: `amarra-cais dev`, open `/dashboard` on a workspace whose CE data includes `AWS Amplify`, confirm the "Amplify by usage" section ranks dimensions descending.
