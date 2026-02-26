package products

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

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

func New(p Params) interfaces.ProductRepo {
	return &repo{
		logger: p.Logger,
		db:     p.Db,
	}
}

func productListFilter(filter structs.ProductFilter) (w []string, v []interface{}) {
	if filter.ID != 0 {
		w = append(w, "id = ?")
		v = append(v, filter.ID)
	}
	if filter.CompanyID != 0 {
		w = append(w, "company_id = ?")
		v = append(v, filter.CompanyID)
	}
	if filter.Name != "" {
		w = append(w, `name ILIKE ? ESCAPE '\'`)
		v = append(v, "%"+escapeLikePattern(filter.Name)+"%")
	}
	if filter.SKU != "" {
		w = append(w, "sku = ?")
		v = append(v, filter.SKU)
	}
	if filter.Barcode != "" {
		w = append(w, "barcode = ?")
		v = append(v, filter.Barcode)
	}
	if filter.Status != "" {
		w = append(w, "status = ?")
		v = append(v, filter.Status)
	}

	return
}

func (r repo) ListProducts(ctx context.Context, filter structs.ProductFilter) ([]structs.Product, error) {
	w, v := productListFilter(filter)

	query := fmt.Sprintf(`SELECT %s FROM products %s ORDER BY updated_at DESC LIMIT ? OFFSET ?;`, columns(), utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return []structs.Product{}, err
	}
	defer rows.Close()

	var products []structs.Product
	for rows.Next() {
		product, scanErr := scanProduct(rows)
		if scanErr != nil {
			return []structs.Product{}, scanErr
		}
		products = append(products, product)
	}
	if err = rows.Err(); err != nil {
		return []structs.Product{}, err
	}

	return products, nil
}

func (r repo) CreateProduct(ctx context.Context, product structs.Product) (structs.Product, error) {
	now := time.Now().UTC()
	sku := nullableString(product.SKU)
	barcode := nullableString(product.Barcode)
	createdBy := nullableInt64Pointer(product.CreatedBy)

	query := `INSERT INTO products (
		company_id,
		name,
		sku,
		barcode,
		unit,
		purchase_price,
		sale_price,
		status,
		created_by,
		created_at,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
	RETURNING
		id,
		company_id,
		name,
		sku,
		barcode,
		unit,
		purchase_price,
		sale_price,
		status,
		created_by,
		created_at,
		updated_at;`

	return scanProduct(r.db.QueryRow(ctx, query,
		product.CompanyID,
		product.Name,
		sku,
		barcode,
		product.Unit,
		product.PurchasePrice,
		product.SalePrice,
		product.Status,
		createdBy,
		now,
	))
}

func columns() string {
	return `id,
		company_id,
		name,
		sku,
		barcode,
		unit,
		purchase_price,
		sale_price,
		status,
		created_by,
		created_at,
		updated_at`
}

func nullableString(v string) interface{} {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}

func escapeLikePattern(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(s)
}

func nullableInt64Pointer(v *int64) interface{} {
	if v == nil || *v == 0 {
		return nil
	}
	return *v
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanProduct(scanner rowScanner) (structs.Product, error) {
	var (
		product      structs.Product
		sku, barcode sql.NullString
		createdBy    sql.NullInt64
	)

	if err := scanner.Scan(
		&product.ID,
		&product.CompanyID,
		&product.Name,
		&sku,
		&barcode,
		&product.Unit,
		&product.PurchasePrice,
		&product.SalePrice,
		&product.Status,
		&createdBy,
		&product.CreatedAt,
		&product.UpdatedAt,
	); err != nil {
		return structs.Product{}, err
	}

	if sku.Valid {
		product.SKU = sku.String
	}
	if barcode.Valid {
		product.Barcode = barcode.String
	}
	if createdBy.Valid {
		product.CreatedBy = &createdBy.Int64
	}

	return product, nil
}
