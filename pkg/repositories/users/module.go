package users

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/fx"

	"github.com/jmoiron/sqlx"
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

func New(p Params) interfaces.UserRepo {
	return &repo{
		logger: p.Logger,
		db:     p.Db,
	}
}

func userListFilter(filter structs.UserFilter) (w []string, v []interface{}) {
	if filter.ID != 0 {
		w = append(w, "id = ?")
		v = append(v, filter.ID)
	}

	if filter.CompanyID != 0 {
		w = append(w, "company_id = ?")
		v = append(v, filter.CompanyID)
	}

	if filter.UserName != "" {
		w = append(w, "username = ?")
		v = append(v, filter.UserName)
	}

	if filter.Phone != "" {
		w = append(w, "phone = ?")
		v = append(v, filter.Phone)
	}

	if filter.Email != "" {
		w = append(w, "email = ?")
		v = append(v, filter.Email)
	}

	if filter.Status != "" {
		w = append(w, "status = ?")
		v = append(v, filter.Status)
	}

	if filter.FirstName != "" {
		w = append(w, "first_name = ?")
		v = append(v, filter.FirstName)
	}

	if filter.LastName != "" {
		w = append(w, "last_name = ?")
		v = append(v, filter.LastName)
	}

	return
}

func (r repo) GetUsers(ctx context.Context, filter structs.UserFilter) (users []structs.User, err error) {
	w, v := userListFilter(filter)

	query := fmt.Sprintf(`SELECT %s FROM users u %s ORDER BY updated_at DESC LIMIT ? OFFSET ?;`, columns(), utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return []structs.User{}, err
	}
	defer rows.Close()
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return []structs.User{}, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r repo) CreateUserWithRoleScope(ctx context.Context, user structs.User, role structs.UserRoleAssignment) (structs.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return structs.User{}, err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	now := time.Now().UTC()

	var (
		companyID interface{}
		phone     interface{}
		email     interface{}
		lastName  interface{}
	)

	if user.CompanyID > 0 {
		companyID = user.CompanyID
	}
	if strings.TrimSpace(user.Phone) != "" {
		phone = user.Phone
	}
	if strings.TrimSpace(user.Email) != "" {
		email = user.Email
	}
	if strings.TrimSpace(user.LastName) != "" {
		lastName = user.LastName
	}

	createUserQuery := `INSERT INTO users (
		company_id,
		username,
		phone,
		email,
		password_hash,
		first_name,
		last_name,
		status,
		is_super_admin,
		created_at,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
	RETURNING
		id,
		company_id,
		username,
		phone,
		email,
		password_hash,
		first_name,
		last_name,
		status,
		is_super_admin,
		last_login_at,
		created_at,
		updated_at;`

	user, err = scanUser(tx.QueryRow(ctx, createUserQuery,
		companyID,
		user.UserName,
		phone,
		email,
		user.PasswordHash,
		user.FirstName,
		lastName,
		user.Status,
		user.IsSuperAdmin,
		now,
	))
	if err != nil {
		return structs.User{}, err
	}

	var roleID int64
	if err = tx.QueryRow(ctx, "SELECT id FROM roles WHERE code = $1 LIMIT 1;", role.RoleCode).Scan(&roleID); err != nil {
		return structs.User{}, err
	}

	var scopeID interface{}
	if role.ScopeID != nil {
		scopeID = *role.ScopeID
	}

	_, err = tx.Exec(ctx, `INSERT INTO user_roles (
		user_id,
		role_id,
		scope_type,
		scope_id,
		own_only
	) VALUES ($1, $2, $3, $4, $5);`,
		user.ID,
		roleID,
		role.ScopeType,
		scopeID,
		role.OwnOnly,
	)
	if err != nil {
		return structs.User{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return structs.User{}, err
	}
	committed = true

	return user, nil
}

func columns() string {
	return `u.id,
		u.company_id,
		u.username,
		u.phone,
		u.email,
		u.password_hash,
		u.first_name,
		u.last_name,
		u.status,
		u.is_super_admin,
		u.last_login_at,
		u.created_at,
		u.updated_at
		`
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(scanner rowScanner) (structs.User, error) {
	var (
		user                   structs.User
		companyID              sql.NullInt64
		phone, email, lastName sql.NullString
		lastLoginAt            sql.NullTime
	)

	if err := scanner.Scan(
		&user.ID,
		&companyID,
		&user.UserName,
		&phone,
		&email,
		&user.PasswordHash,
		&user.FirstName,
		&lastName,
		&user.Status,
		&user.IsSuperAdmin,
		&lastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return structs.User{}, err
	}

	if companyID.Valid {
		user.CompanyID = companyID.Int64
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.Time
	}

	return user, nil
}
