package store

import (
	"database/sql"
	"fmt"

	"github.com/puppe1990/cifra-finops/internal/models"
)

func (s *SQLiteStore) CreateCloudAccount(acc models.CloudAccount) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO cloud_accounts
         (tenant_id, aws_account_id, alias, region, auth_mode, access_key_id, secret_cipher, is_primary)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		acc.TenantID, acc.AWSAccountID, acc.Alias, acc.Region, acc.AuthMode,
		acc.AccessKeyID, acc.SecretCipher, boolToInt(acc.IsPrimary),
	)
	if err != nil {
		return 0, fmt.Errorf("create cloud account: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) EnsureCloudAccount(acc models.CloudAccount) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM cloud_accounts WHERE tenant_id = ? AND aws_account_id = ?`,
		acc.TenantID, acc.AWSAccountID,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("ensure cloud account: %w", err)
	}
	return s.CreateCloudAccount(acc)
}

func (s *SQLiteStore) UpdateCloudAccountAuth(acc models.CloudAccount) error {
	res, err := s.db.Exec(
		`UPDATE cloud_accounts
         SET alias = ?, region = ?, auth_mode = ?, access_key_id = ?, secret_cipher = ?
         WHERE tenant_id = ? AND aws_account_id = ?`,
		acc.Alias, acc.Region, acc.AuthMode, acc.AccessKeyID, acc.SecretCipher,
		acc.TenantID, acc.AWSAccountID,
	)
	if err != nil {
		return fmt.Errorf("update cloud account auth: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update cloud account auth: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("update cloud account auth: not found")
	}
	return nil
}

func (s *SQLiteStore) FindCloudAccount(id int64) (models.CloudAccount, error) {
	return s.scanCloudAccount(
		s.db.QueryRow(`SELECT id, tenant_id, aws_account_id, alias, region, auth_mode,
            access_key_id, secret_cipher, is_primary, created_at
            FROM cloud_accounts WHERE id = ?`, id),
	)
}

func (s *SQLiteStore) FindCloudAccountForTenant(tenantID, id int64) (models.CloudAccount, error) {
	return s.scanCloudAccount(
		s.db.QueryRow(`SELECT id, tenant_id, aws_account_id, alias, region, auth_mode,
            access_key_id, secret_cipher, is_primary, created_at
            FROM cloud_accounts WHERE id = ? AND tenant_id = ?`, id, tenantID),
	)
}

func (s *SQLiteStore) ListCloudAccounts(tenantID int64) ([]models.CloudAccount, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id, aws_account_id, alias, region, auth_mode,
                access_key_id, secret_cipher, is_primary, created_at
         FROM cloud_accounts WHERE tenant_id = ? ORDER BY is_primary DESC, alias`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list cloud accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []models.CloudAccount
	for rows.Next() {
		acc, err := scanCloudAccountRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, acc)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) scanCloudAccount(row *sql.Row) (models.CloudAccount, error) {
	acc, err := scanCloudAccountRow(row)
	if err != nil {
		return models.CloudAccount{}, fmt.Errorf("find cloud account: %w", err)
	}
	return acc, nil
}

type accountRow interface {
	Scan(dest ...any) error
}

func scanCloudAccountRow(row accountRow) (models.CloudAccount, error) {
	var acc models.CloudAccount
	var primary int
	err := row.Scan(
		&acc.ID, &acc.TenantID, &acc.AWSAccountID, &acc.Alias, &acc.Region,
		&acc.AuthMode, &acc.AccessKeyID, &acc.SecretCipher, &primary, &acc.CreatedAt,
	)
	if err != nil {
		return models.CloudAccount{}, err
	}
	acc.IsPrimary = primary == 1
	return acc, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
