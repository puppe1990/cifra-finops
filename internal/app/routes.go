package app

import (
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/middleware"

	"github.com/puppe1990/aws-finops/internal/handlers"
)

func registerRoutes(r *cais.Router, deps Deps, cfg cais.Config) {
	home := handlers.NewHomeHandler(nil, deps.Site, deps.Catalog, cfg, deps.Views)
	contact := handlers.NewContactHandler(nil, deps.Store, deps.Site, deps.Catalog, cfg, deps.Views)
	dashboard := handlers.NewDashboardHandler(nil, deps.Store, deps.Site, cfg, deps.Views).WithSyncer(deps.Syncer)
	compare := handlers.NewCompareHandler(deps.Store, deps.Site, cfg, deps.Views).WithSyncer(deps.Syncer)
	anomalies := handlers.NewAnomaliesHandler(deps.Store, deps.Site, cfg, deps.Views).WithSyncer(deps.Syncer)
	auth := handlers.NewAuthHandler(nil, deps.Store, deps.Site, deps.Store.Sessions(), cfg, deps.Catalog, deps.Views)
	accounts := handlers.NewAccountsHandler(deps.Store, deps.Site, cfg, deps.Views, deps.AppSecret).WithSyncer(deps.Syncer)
	resources := handlers.NewResourcesHandler(deps.Store, deps.Site, cfg, deps.Views)
	budgets := handlers.NewBudgetsHandler(deps.Store, deps.Site, cfg, deps.Views)
	tenants := handlers.NewTenantsHandler(deps.Store, deps.Site, cfg, deps.Views)
	settings := handlers.NewSettingsHandler(deps.Store, deps.Site, cfg, deps.Views)
	loc := handlers.NewLocaleHandler(cfg, deps.Views)

	loginLimit := middleware.NewRateLimiter(10, cfg)
	resetLimit := middleware.NewRateLimiter(10, cfg)
	contactLimit := middleware.NewRateLimiter(20, cfg)

	r.Get("/", home.ServeHTTP)
	r.Get("/contact", contact.Get)
	r.Post("/contact", contactLimit.Middleware(http.HandlerFunc(contact.Post)).ServeHTTP)
	r.Get("/login", auth.Login)
	r.Post("/login", loginLimit.Middleware(http.HandlerFunc(auth.LoginPost)).ServeHTTP)
	r.Get("/signup", auth.SignUp)
	r.Post("/signup", loginLimit.Middleware(http.HandlerFunc(auth.SignUpPost)).ServeHTTP)
	r.Get("/forgot-password", auth.ForgotPassword)
	r.Post("/forgot-password", resetLimit.Middleware(http.HandlerFunc(auth.ForgotPasswordPost)).ServeHTTP)
	r.Get("/reset-password", auth.ResetPassword)
	r.Post("/reset-password", resetLimit.Middleware(http.HandlerFunc(auth.ResetPasswordPost)).ServeHTTP)
	r.Post("/logout", auth.LogoutPost)
	r.Post("/locale", loc.Post)

	authN := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.RequireAuthFunc("/login", next)
	}
	r.Get("/dashboard", authN(dashboard.ServeHTTP))
	r.Get("/compare", authN(compare.ServeHTTP))
	r.Get("/anomalies", authN(anomalies.ServeHTTP))
	r.Get("/resources", authN(resources.List))
	r.Get("/accounts", authN(accounts.List))
	r.Post("/accounts", authN(accounts.Create))
	r.Post("/sync", authN(accounts.Sync))
	r.Get("/budgets", authN(budgets.List))
	r.Post("/budgets", authN(budgets.Create))
	r.Get("/tenants", authN(tenants.List))
	r.Post("/tenants", authN(tenants.Create))
	r.Post("/tenants/switch", authN(tenants.Switch))
	r.Get("/settings", authN(settings.Get))
}
