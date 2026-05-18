package company

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"

	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
	"nurmed/pkg/utils"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger logger.ILogger
	Db     interfaces.Querier
}

type repo struct {
	logger logger.ILogger
	db     interfaces.Querier
}

func New(p Params) interfaces.CompanyRepo {
	return &repo{
		logger: p.Logger,
		db:     p.Db,
	}
}

func (r *repo) CreateCompany(ctx context.Context, company structs.Company) (structs.Company, error) {
	now := time.Now().UTC()

	query := `INSERT INTO companies (name, status, created_at, updated_at)
	VALUES ($1, $2, $3, $3)
	RETURNING id, name, status, deleted_at, created_at, updated_at;`

	row := r.db.QueryRow(ctx, query, company.Name, company.Status, now)
	return scanCompany(row)
}

func (r *repo) ListCompanies(ctx context.Context, filter structs.CompanyFilter) ([]structs.Company, error) {
	filter.Validate()

	var w []string
	var v []interface{}

	if filter.Name != "" {
		w = append(w, "name ILIKE ?")
		v = append(v, "%"+filter.Name+"%")
	}
	if filter.Status != "" {
		w = append(w, "status = ?")
		v = append(v, filter.Status)
	}

	w = append(w, "deleted_at IS NULL")
	query := fmt.Sprintf(
		`SELECT id, name, status, deleted_at, created_at, updated_at FROM companies %s ORDER BY created_at DESC LIMIT ? OFFSET ?;`,
		utils.Where(w),
	)
	v = append(v, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var companies []structs.Company
	for rows.Next() {
		c, err := scanCompany(rows)
		if err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}
	return companies, nil
}

func (r *repo) UpdateCompany(ctx context.Context, id int64, req structs.UpdateCompanyRequest) (structs.Company, error) {
	var sets []string
	var v []interface{}

	if strings.TrimSpace(req.Name) != "" {
		sets = append(sets, "name = ?")
		v = append(v, strings.TrimSpace(req.Name))
	}
	if strings.TrimSpace(req.Status) != "" {
		sets = append(sets, "status = ?")
		v = append(v, strings.ToLower(strings.TrimSpace(req.Status)))
	}
	if len(sets) == 0 {
		return r.getByID(ctx, id)
	}

	sets = append(sets, "updated_at = ?")
	v = append(v, time.Now().UTC())
	v = append(v, id)

	query := fmt.Sprintf(
		`UPDATE companies SET %s WHERE id = ? AND deleted_at IS NULL
		 RETURNING id, name, status, deleted_at, created_at, updated_at;`,
		strings.Join(sets, ", "),
	)

	row := r.db.QueryRow(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	return scanCompany(row)
}

func (r *repo) DeleteCompany(ctx context.Context, id int64) error {
	query := `UPDATE companies SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL;`
	tag, err := r.db.Exec(ctx, query, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *repo) getByID(ctx context.Context, id int64) (structs.Company, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, name, status, deleted_at, created_at, updated_at FROM companies WHERE id = $1 AND deleted_at IS NULL;`,
		id,
	)
	return scanCompany(row)
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanCompany(s scanner) (structs.Company, error) {
	var (
		c         structs.Company
		deletedAt sql.NullTime
	)
	err := s.Scan(&c.ID, &c.Name, &c.Status, &deletedAt, &c.CreatedAt, &c.UpdatedAt)
	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Time
	}
	return c, err
}
