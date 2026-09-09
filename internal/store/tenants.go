package store

import (
	"database/sql"
	"fmt"

	"github.com/puppe1990/cifra-finops/internal/models"
)

func (s *SQLiteStore) CreateTenant(name, slug string) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO tenants (name, slug) VALUES (?, ?)",
		name, slug,
	)
	if err != nil {
		return 0, fmt.Errorf("create tenant: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) FindTenantBySlug(slug string) (models.Tenant, error) {
	var t models.Tenant
	err := s.db.QueryRow(
		"SELECT id, name, slug, created_at FROM tenants WHERE slug = ?",
		slug,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt)
	if err != nil {
		return models.Tenant{}, fmt.Errorf("find tenant %s: %w", slug, err)
	}
	return t, nil
}

func (s *SQLiteStore) FindTenantByID(id int64) (models.Tenant, error) {
	var t models.Tenant
	err := s.db.QueryRow(
		"SELECT id, name, slug, created_at FROM tenants WHERE id = ?",
		id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt)
	if err != nil {
		return models.Tenant{}, fmt.Errorf("find tenant %d: %w", id, err)
	}
	return t, nil
}

func (s *SQLiteStore) AddMember(tenantID, userID int64, role string) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO tenant_members (tenant_id, user_id, role) VALUES (?, ?, ?)`,
		tenantID, userID, role,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (s *SQLiteStore) MembershipRole(tenantID, userID int64) (string, bool, error) {
	var role string
	err := s.db.QueryRow(
		"SELECT role FROM tenant_members WHERE tenant_id = ? AND user_id = ?",
		tenantID, userID,
	).Scan(&role)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("membership: %w", err)
	}
	return role, true, nil
}

func (s *SQLiteStore) ListTenantsForUser(userID int64) ([]models.Tenant, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.name, t.slug, t.created_at
         FROM tenants t
         JOIN tenant_members m ON m.tenant_id = t.id
         WHERE m.user_id = ?
         ORDER BY t.name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []models.Tenant
	for rows.Next() {
		var t models.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) SetActiveTenant(userID, tenantID int64) error {
	_, err := s.db.Exec("UPDATE users SET active_tenant_id = ? WHERE id = ?", tenantID, userID)
	if err != nil {
		return fmt.Errorf("set active tenant: %w", err)
	}
	return nil
}

func (s *SQLiteStore) FindUserByID(id int64) (models.User, error) {
	var u models.User
	var active sql.NullInt64
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, created_at, active_tenant_id
         FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &active)
	if err != nil {
		return models.User{}, fmt.Errorf("find user: %w", err)
	}
	if active.Valid {
		u.ActiveTenantID = active.Int64
	}
	return u, nil
}
