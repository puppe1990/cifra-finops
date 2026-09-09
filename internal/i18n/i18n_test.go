package i18n

import "testing"

func TestDefaultCatalog_english(t *testing.T) {
	c := DefaultCatalog()
	if got := c.T("auth.welcome"); got != "Welcome!" {
		t.Errorf("T(auth.welcome) = %q", got)
	}
	if got := c.T("auth.signup_prompt"); got != "Don't have an account?" {
		t.Errorf("T(auth.signup_prompt) = %q", got)
	}
}

func TestLabels_dashboardBudgetLine(t *testing.T) {
	en := Labels("en")
	if en["dash.budget_line"] == "" {
		t.Fatal("missing dash.budget_line")
	}
	if _, ok := en["dash.costliest"]; ok {
		t.Fatal("dash.costliest should be removed")
	}
	if _, ok := en["dash.col_month"]; ok {
		t.Fatal("dash.col_month should be removed")
	}
}

func TestLabels_compareKeys(t *testing.T) {
	en := Labels("en")
	pt := Labels("pt-BR")
	if en["nav.compare"] != "Compare" || pt["nav.compare"] != "Comparativo" {
		t.Fatalf("nav.compare en=%q pt=%q", en["nav.compare"], pt["nav.compare"])
	}
	if en["cmp.title"] == "" || pt["cmp.title"] == "" {
		t.Fatal("missing cmp.title")
	}
	if en["cmp.by_service"] != "By service" || pt["cmp.by_service"] != "Por serviço" {
		t.Fatalf("by_service en=%q pt=%q", en["cmp.by_service"], pt["cmp.by_service"])
	}
}

func TestLabels_anomaliesKeys(t *testing.T) {
	en := Labels("en")
	pt := Labels("pt-BR")
	if en["nav.anomalies"] != "Anomalies" || pt["nav.anomalies"] != "Anomalias" {
		t.Fatalf("nav.anomalies en=%q pt=%q", en["nav.anomalies"], pt["nav.anomalies"])
	}
	if en["ano.spike"] != "Spend spike" || pt["ano.spike"] != "Salto de gasto" {
		t.Fatalf("ano.spike en=%q pt=%q", en["ano.spike"], pt["ano.spike"])
	}
}

func TestLabels_forecastLine(t *testing.T) {
	en := Labels("en")
	pt := Labels("pt-BR")
	if en["dash.forecast"] != "Forecast %s · %s" {
		t.Fatalf("en=%q", en["dash.forecast"])
	}
	if pt["dash.forecast"] != "Previsão %s · %s" {
		t.Fatalf("pt=%q", pt["dash.forecast"])
	}
}

func TestLabels_ledgerMonthKeys(t *testing.T) {
	en := Labels("en")
	pt := Labels("pt-BR")
	if en["dash.month_fmt"] != "%s %d" || pt["dash.month_fmt"] != "%s %d" {
		t.Fatalf("month_fmt en=%q pt=%q", en["dash.month_fmt"], pt["dash.month_fmt"])
	}
	if en["dash.m08"] != "Aug" || pt["dash.m08"] != "ago" {
		t.Fatalf("m08 en=%q pt=%q", en["dash.m08"], pt["dash.m08"])
	}
}

func TestLabels_multiCloudLedgerCopy(t *testing.T) {
	en := Labels("en")
	pt := Labels("pt-BR")
	if en["auth.tagline"] != "The multi cloud finops ledger" {
		t.Fatalf("auth.tagline en=%q", en["auth.tagline"])
	}
	if en["home.rails_heading"] != "The multi cloud finops ledger." {
		t.Fatalf("home.rails_heading en=%q", en["home.rails_heading"])
	}
	if pt["auth.tagline"] != "O livro-caixa FinOps multi-cloud" {
		t.Fatalf("auth.tagline pt=%q", pt["auth.tagline"])
	}
	if pt["home.rails_heading"] != "O livro-caixa FinOps multi-cloud." {
		t.Fatalf("home.rails_heading pt=%q", pt["home.rails_heading"])
	}
}

func TestLabels_passwordToggleKeys(t *testing.T) {
	en := Labels("en")
	pt := Labels("pt-BR")
	if en["auth.show_password"] == "" || pt["auth.show_password"] == "" {
		t.Fatal("missing auth.show_password")
	}
	if en["auth.hide_password"] == "" || pt["auth.hide_password"] == "" {
		t.Fatal("missing auth.hide_password")
	}
	if _, ok := en["auth.demo_hint"]; ok {
		t.Fatal("auth.demo_hint should be removed")
	}
}

func TestLabels_sameKeysEnAndPt(t *testing.T) {
	en := Labels("en")
	pt := Labels("pt-BR")
	if len(en) == 0 || len(pt) == 0 {
		t.Fatal("empty catalog")
	}
	for k := range en {
		if _, ok := pt[k]; !ok {
			t.Errorf("pt missing key %q", k)
		}
	}
	for k := range pt {
		if _, ok := en[k]; !ok {
			t.Errorf("en missing key %q", k)
		}
	}
}

func TestNewCatalog_portuguese(t *testing.T) {
	c := NewCatalog("pt-BR")
	if got := c.T("auth.welcome"); got != "Bem-vindo!" {
		t.Errorf("T(auth.welcome) = %q", got)
	}
	if c.HTMLLang() != "pt-BR" {
		t.Errorf("HTMLLang() = %q, want pt-BR", c.HTMLLang())
	}
}
