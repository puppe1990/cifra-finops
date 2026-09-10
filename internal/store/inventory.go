package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/puppe1990/cifra-finops/internal/models"
)

func (s *SQLiteStore) ReplaceResources(accountID int64, resources []models.CloudResource) error {
	if _, err := s.db.Exec("DELETE FROM cloud_resources WHERE cloud_account_id = ?", accountID); err != nil {
		return fmt.Errorf("clear resources: %w", err)
	}
	for _, r := range resources {
		if r.MetaJSON == "" {
			r.MetaJSON = "{}"
		}
		_, err := s.db.Exec(
			`INSERT INTO cloud_resources
             (cloud_account_id, kind, name, region, state, monthly_cents, source, currency, external_id, meta_json)
             VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			accountID, r.Kind, r.Name, r.Region, r.State, r.MonthlyCents, r.Source, r.Currency, r.ExternalID, r.MetaJSON,
		)
		if err != nil {
			return fmt.Errorf("insert resource %s: %w", r.Name, err)
		}
	}
	return nil
}

func (s *SQLiteStore) ListResourcesForTenant(tenantID int64) ([]models.CloudResource, error) {
	rows, err := s.db.Query(
		`SELECT r.id, r.cloud_account_id, r.kind, r.name, r.region, r.state,
                r.monthly_cents, r.source, r.currency, r.external_id, r.meta_json
         FROM cloud_resources r
         JOIN cloud_accounts a ON a.id = r.cloud_account_id
         WHERE a.tenant_id = ?
         ORDER BY r.monthly_cents DESC, r.name`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanResources(rows)
}

func (s *SQLiteStore) ListResources(accountID int64) ([]models.CloudResource, error) {
	rows, err := s.db.Query(
		`SELECT id, cloud_account_id, kind, name, region, state,
                monthly_cents, source, currency, external_id, meta_json
         FROM cloud_resources WHERE cloud_account_id = ?
         ORDER BY monthly_cents DESC, name`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("list account resources: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanResources(rows)
}

func scanResources(rows *sql.Rows) ([]models.CloudResource, error) {
	var out []models.CloudResource
	for rows.Next() {
		var r models.CloudResource
		if err := rows.Scan(
			&r.ID, &r.CloudAccountID, &r.Kind, &r.Name, &r.Region, &r.State,
			&r.MonthlyCents, &r.Source, &r.Currency, &r.ExternalID, &r.MetaJSON,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ReplaceCostLines(accountID int64, lines []models.CostLine) error {
	if _, err := s.db.Exec("DELETE FROM cost_lines WHERE cloud_account_id = ?", accountID); err != nil {
		return fmt.Errorf("clear cost lines: %w", err)
	}
	for _, line := range lines {
		_, err := s.db.Exec(
			`INSERT INTO cost_lines (cloud_account_id, service, usage_type, monthly_cents, source, currency, period_start, period_end)
             VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			accountID, line.Service, line.UsageType, line.MonthlyCents, line.Source, line.Currency, line.PeriodStart, line.PeriodEnd,
		)
		if err != nil {
			return fmt.Errorf("insert cost line: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) ListCostLines(accountID int64) ([]models.CostLine, error) {
	rows, err := s.db.Query(
		`SELECT id, cloud_account_id, service, usage_type, monthly_cents, source, currency, period_start, period_end
         FROM cost_lines WHERE cloud_account_id = ? ORDER BY monthly_cents DESC`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("list cost lines: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []models.CostLine
	for rows.Next() {
		var line models.CostLine
		if err := rows.Scan(
			&line.ID, &line.CloudAccountID, &line.Service, &line.UsageType, &line.MonthlyCents,
			&line.Source, &line.Currency, &line.PeriodStart, &line.PeriodEnd,
		); err != nil {
			return nil, err
		}
		out = append(out, line)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ReplaceFindings(accountID int64, findings []models.Finding) error {
	if _, err := s.db.Exec("DELETE FROM findings WHERE cloud_account_id = ?", accountID); err != nil {
		return fmt.Errorf("clear findings: %w", err)
	}
	for _, f := range findings {
		_, err := s.db.Exec(
			`INSERT INTO findings (cloud_account_id, kind, severity, title, detail)
             VALUES (?, ?, ?, ?, ?)`,
			accountID, f.Kind, f.Severity, f.Title, f.Detail,
		)
		if err != nil {
			return fmt.Errorf("insert finding: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) ListFindings(accountID int64) ([]models.Finding, error) {
	rows, err := s.db.Query(
		`SELECT id, cloud_account_id, kind, severity, title, detail
         FROM findings WHERE cloud_account_id = ? ORDER BY id`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []models.Finding
	for rows.Next() {
		var f models.Finding
		if err := rows.Scan(&f.ID, &f.CloudAccountID, &f.Kind, &f.Severity, &f.Title, &f.Detail); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListFindingsForTenant(tenantID int64) ([]models.Finding, error) {
	rows, err := s.db.Query(
		`SELECT f.id, f.cloud_account_id, f.kind, f.severity, f.title, f.detail
         FROM findings f
         JOIN cloud_accounts a ON a.id = f.cloud_account_id
         WHERE a.tenant_id = ? ORDER BY f.id`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tenant findings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []models.Finding
	for rows.Next() {
		var f models.Finding
		if err := rows.Scan(&f.ID, &f.CloudAccountID, &f.Kind, &f.Severity, &f.Title, &f.Detail); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) StartSyncRun(accountID int64) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO sync_runs (cloud_account_id, status) VALUES (?, 'running')`,
		accountID,
	)
	if err != nil {
		return 0, fmt.Errorf("start sync: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) FinishSyncRun(id int64, status, source, warning, errText string) error {
	_, err := s.db.Exec(
		`UPDATE sync_runs SET status = ?, source = ?, warning = ?, error = ?, finished_at = CURRENT_TIMESTAMP
         WHERE id = ?`,
		status, source, warning, errText, id,
	)
	if err != nil {
		return fmt.Errorf("finish sync: %w", err)
	}
	return nil
}

func (s *SQLiteStore) LastSyncRun(accountID int64) (models.SyncRun, error) {
	var run models.SyncRun
	var finished sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, cloud_account_id, status, source, error, warning, started_at, finished_at
         FROM sync_runs WHERE cloud_account_id = ? ORDER BY id DESC LIMIT 1`,
		accountID,
	).Scan(&run.ID, &run.CloudAccountID, &run.Status, &run.Source, &run.Error, &run.Warning, &run.StartedAt, &finished)
	if err == sql.ErrNoRows {
		return models.SyncRun{}, nil
	}
	if err != nil {
		return models.SyncRun{}, fmt.Errorf("last sync: %w", err)
	}
	if finished.Valid {
		run.FinishedAt = finished.Time
	}
	return run, nil
}

func (s *SQLiteStore) CreateBudget(b models.Budget) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO budgets (tenant_id, cloud_account_id, name, amount_cents, period)
         VALUES (?, ?, ?, ?, ?)`,
		b.TenantID, nullIfZero(b.CloudAccountID), b.Name, b.AmountCents, defaultPeriod(b.Period),
	)
	if err != nil {
		return 0, fmt.Errorf("create budget: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) ListBudgets(tenantID int64) ([]models.Budget, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id, cloud_account_id, name, amount_cents, period
         FROM budgets WHERE tenant_id = ? ORDER BY id`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []models.Budget
	for rows.Next() {
		var b models.Budget
		var acc sql.NullInt64
		if err := rows.Scan(&b.ID, &b.TenantID, &acc, &b.Name, &b.AmountCents, &b.Period); err != nil {
			return nil, err
		}
		if acc.Valid {
			b.CloudAccountID = acc.Int64
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) EnsureBudget(b models.Budget) error {
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM budgets WHERE tenant_id = ? AND name = ?`,
		b.TenantID, b.Name,
	).Scan(&id)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	_, err = s.CreateBudget(b)
	return err
}

func nullIfZero(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func defaultPeriod(p string) string {
	if strings.TrimSpace(p) == "" {
		return "monthly"
	}
	return p
}

func MonthBounds(now time.Time) (string, string) {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}
