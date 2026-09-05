package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/aws-finops/internal/finops"
	appi18n "github.com/puppe1990/aws-finops/internal/i18n"
	"github.com/puppe1990/aws-finops/internal/locale"
	"github.com/puppe1990/aws-finops/internal/models"
	"github.com/puppe1990/aws-finops/internal/store"
)

type workspace struct {
	User   models.User
	Tenant models.Tenant
	Role   string
}

func loadWorkspace(s store.Store, r *http.Request) (workspace, error) {
	uid, ok := session.UserID(r)
	if !ok {
		return workspace{}, errNoSession
	}
	user, err := s.FindUserByID(uid)
	if err != nil {
		return workspace{}, err
	}
	tenants, err := s.ListTenantsForUser(uid)
	if err != nil {
		return workspace{}, err
	}
	if len(tenants) == 0 {
		return workspace{User: user}, errNoTenant
	}
	tenant := tenants[0]
	if user.ActiveTenantID != 0 {
		for _, t := range tenants {
			if t.ID == user.ActiveTenantID {
				tenant = t
				break
			}
		}
	}
	role, _, err := s.MembershipRole(tenant.ID, uid)
	if err != nil {
		return workspace{}, err
	}
	if user.ActiveTenantID != tenant.ID {
		_ = s.SetActiveTenant(uid, tenant.ID)
	}
	return workspace{User: user, Tenant: tenant, Role: role}, nil
}

func createPersonalTenant(s store.Store, userID int64, email string) error {
	slug := slugFromEmail(email, userID)
	id, err := s.CreateTenant("Workspace", slug)
	if err != nil {
		return err
	}
	if err := s.AddMember(id, userID, finops.RoleOwner); err != nil {
		return err
	}
	return s.SetActiveTenant(userID, id)
}

func slugFromEmail(email string, userID int64) string {
	local := strings.ToLower(strings.Split(email, "@")[0])
	re := regexp.MustCompile(`[^a-z0-9]+`)
	local = strings.Trim(re.ReplaceAllString(local, "-"), "-")
	if local == "" {
		local = "ws"
	}
	return fmt.Sprintf("%s-%d", local, userID)
}

func requestCatalog(r *http.Request, fallback string) *i18n.Catalog {
	return appi18n.NewCatalog(locale.FromRequest(r, fallback))
}

func publicProps(site meta.Site, r *http.Request, fallback string) map[string]any {
	loc := locale.FromRequest(r, fallback)
	props := map[string]any{
		"site":     meta.ForRequest(site, r),
		"locale":   loc,
		"htmlLang": appi18n.NewCatalog(loc).HTMLLang(),
		"labels":   appi18n.Labels(loc),
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = map[string]string{msg.Kind: msg.Message}
	}
	return props
}

func shellProps(hSite meta.Site, r *http.Request, s store.Store, ws workspace) map[string]any {
	tenants, _ := s.ListTenantsForUser(ws.User.ID)
	props := publicProps(hSite, r, "en")
	props["userEmail"] = ws.User.Email
	props["tenant"] = map[string]any{"id": ws.Tenant.ID, "name": ws.Tenant.Name, "slug": ws.Tenant.Slug, "role": ws.Role}
	props["tenants"] = tenantsProps(tenants)
	props["primaryAws"] = finops.SeedAWSAccountID()
	return props
}

func tenantsProps(tenants []models.Tenant) []map[string]any {
	out := make([]map[string]any, 0, len(tenants))
	for _, t := range tenants {
		out = append(out, map[string]any{"id": t.ID, "name": t.Name, "slug": t.Slug})
	}
	return out
}

var (
	errNoSession = fmt.Errorf("not signed in")
	errNoTenant  = fmt.Errorf("no tenant")
)
