package purchases

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

func New(p Params) interfaces.PurchaseRepo {
	return &repo{
		logger: p.Logger,
		db:     p.Db,
	}
}

func orderListFilter(filter structs.PurchaseOrderFilter) (w []string, v []interface{}) {
	if filter.ID != 0 {
		w = append(w, "id = ?")
		v = append(v, filter.ID)
	}
	if filter.CompanyID != 0 {
		w = append(w, "company_id = ?")
		v = append(v, filter.CompanyID)
	}
	if filter.DocumentNo != "" {
		w = append(w, "document_no = ?")
		v = append(v, filter.DocumentNo)
	}
	if filter.SupplierName != "" {
		w = append(w, `supplier_name ILIKE ? ESCAPE '\'`)
		v = append(v, "%"+escapeLikePattern(filter.SupplierName)+"%")
	}
	if filter.Status != "" {
		w = append(w, "status = ?")
		v = append(v, filter.Status)
	}

	return
}

func (r repo) ListOrders(ctx context.Context, filter structs.PurchaseOrderFilter) ([]structs.PurchaseOrder, error) {
	w, v := orderListFilter(filter)

	query := fmt.Sprintf(`SELECT %s FROM purchase_orders %s ORDER BY purchased_at DESC LIMIT ? OFFSET ?;`, orderColumns(), utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return []structs.PurchaseOrder{}, err
	}
	defer rows.Close()

	var orders []structs.PurchaseOrder
	for rows.Next() {
		order, scanErr := scanPurchaseOrder(rows)
		if scanErr != nil {
			return []structs.PurchaseOrder{}, scanErr
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		return []structs.PurchaseOrder{}, err
	}

	return orders, nil
}

func (r repo) CreateOrder(ctx context.Context, order structs.PurchaseOrder, items []structs.PurchaseOrderItem) (structs.PurchaseOrder, []structs.PurchaseOrderItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return structs.PurchaseOrder{}, nil, err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	now := time.Now().UTC()
	supplierName := nullableString(order.SupplierName)
	createdBy := nullableInt64Pointer(order.CreatedBy)

	createOrderQuery := `INSERT INTO purchase_orders (
		company_id,
		document_no,
		supplier_name,
		currency,
		status,
		total_amount,
		item_count,
		purchased_at,
		created_by,
		created_at,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
	RETURNING
		id,
		company_id,
		document_no,
		supplier_name,
		currency,
		status,
		total_amount,
		item_count,
		purchased_at,
		created_by,
		created_at,
		updated_at;`

	createdOrder, err := scanPurchaseOrder(tx.QueryRow(
		ctx,
		createOrderQuery,
		order.CompanyID,
		order.DocumentNo,
		supplierName,
		order.Currency,
		order.Status,
		order.TotalAmount,
		order.ItemCount,
		order.PurchasedAt,
		createdBy,
		now,
	))
	if err != nil {
		return structs.PurchaseOrder{}, nil, err
	}

	createdItems := make([]structs.PurchaseOrderItem, 0, len(items))
	for _, item := range items {
		productID := nullableInt64Pointer(item.ProductID)

		createdItem, itemErr := scanPurchaseOrderItem(tx.QueryRow(ctx, `INSERT INTO purchase_order_items (
			order_id,
			product_id,
			product_name,
			quantity,
			price,
			amount,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, order_id, product_id, product_name, quantity, price, amount, created_at;`,
			createdOrder.ID,
			productID,
			item.ProductName,
			item.Quantity,
			item.Price,
			item.Amount,
			now,
		))
		if itemErr != nil {
			return structs.PurchaseOrder{}, nil, itemErr
		}

		createdItems = append(createdItems, createdItem)
	}

	if err = tx.Commit(ctx); err != nil {
		return structs.PurchaseOrder{}, nil, err
	}
	committed = true

	return createdOrder, createdItems, nil
}

func returnListFilter(filter structs.PurchaseReturnFilter) (w []string, v []interface{}) {
	if filter.ID != 0 {
		w = append(w, "id = ?")
		v = append(v, filter.ID)
	}
	if filter.OrderID != 0 {
		w = append(w, "order_id = ?")
		v = append(v, filter.OrderID)
	}
	if filter.CompanyID != 0 {
		w = append(w, "company_id = ?")
		v = append(v, filter.CompanyID)
	}
	if filter.Status != "" {
		w = append(w, "status = ?")
		v = append(v, filter.Status)
	}

	return
}

func (r repo) ListReturns(ctx context.Context, filter structs.PurchaseReturnFilter) ([]structs.PurchaseReturn, error) {
	w, v := returnListFilter(filter)

	query := fmt.Sprintf(`SELECT %s FROM purchase_returns %s ORDER BY returned_at DESC LIMIT ? OFFSET ?;`, returnColumns(), utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return []structs.PurchaseReturn{}, err
	}
	defer rows.Close()

	var returns []structs.PurchaseReturn
	for rows.Next() {
		purchaseReturn, scanErr := scanPurchaseReturn(rows)
		if scanErr != nil {
			return []structs.PurchaseReturn{}, scanErr
		}
		returns = append(returns, purchaseReturn)
	}
	if err = rows.Err(); err != nil {
		return []structs.PurchaseReturn{}, err
	}

	return returns, nil
}

func (r repo) CreateReturn(ctx context.Context, purchaseReturn structs.PurchaseReturn) (structs.PurchaseReturn, error) {
	now := time.Now().UTC()
	reason := nullableString(purchaseReturn.Reason)
	createdBy := nullableInt64Pointer(purchaseReturn.CreatedBy)

	createQuery := `INSERT INTO purchase_returns (
		order_id,
		company_id,
		reason,
		status,
		total_amount,
		returned_at,
		created_by,
		created_at,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
	RETURNING
		id,
		order_id,
		company_id,
		reason,
		status,
		total_amount,
		returned_at,
		created_by,
		created_at,
		updated_at;`

	return scanPurchaseReturn(r.db.QueryRow(
		ctx,
		createQuery,
		purchaseReturn.OrderID,
		purchaseReturn.CompanyID,
		reason,
		purchaseReturn.Status,
		purchaseReturn.TotalAmount,
		purchaseReturn.ReturnedAt,
		createdBy,
		now,
	))
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

func orderColumns() string {
	return `id,
		company_id,
		document_no,
		supplier_name,
		currency,
		status,
		total_amount,
		item_count,
		purchased_at,
		created_by,
		created_at,
		updated_at`
}

func returnColumns() string {
	return `id,
		order_id,
		company_id,
		reason,
		status,
		total_amount,
		returned_at,
		created_by,
		created_at,
		updated_at`
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPurchaseOrder(scanner rowScanner) (structs.PurchaseOrder, error) {
	var (
		order        structs.PurchaseOrder
		supplierName sql.NullString
		createdBy    sql.NullInt64
	)

	if err := scanner.Scan(
		&order.ID,
		&order.CompanyID,
		&order.DocumentNo,
		&supplierName,
		&order.Currency,
		&order.Status,
		&order.TotalAmount,
		&order.ItemCount,
		&order.PurchasedAt,
		&createdBy,
		&order.CreatedAt,
		&order.UpdatedAt,
	); err != nil {
		return structs.PurchaseOrder{}, err
	}

	if supplierName.Valid {
		order.SupplierName = supplierName.String
	}
	if createdBy.Valid {
		order.CreatedBy = &createdBy.Int64
	}

	return order, nil
}

func scanPurchaseOrderItem(scanner rowScanner) (structs.PurchaseOrderItem, error) {
	var (
		item      structs.PurchaseOrderItem
		productID sql.NullInt64
	)

	if err := scanner.Scan(
		&item.ID,
		&item.OrderID,
		&productID,
		&item.ProductName,
		&item.Quantity,
		&item.Price,
		&item.Amount,
		&item.CreatedAt,
	); err != nil {
		return structs.PurchaseOrderItem{}, err
	}

	if productID.Valid {
		item.ProductID = &productID.Int64
	}

	return item, nil
}

func scanPurchaseReturn(scanner rowScanner) (structs.PurchaseReturn, error) {
	var (
		purchaseReturn structs.PurchaseReturn
		reason         sql.NullString
		createdBy      sql.NullInt64
	)

	if err := scanner.Scan(
		&purchaseReturn.ID,
		&purchaseReturn.OrderID,
		&purchaseReturn.CompanyID,
		&reason,
		&purchaseReturn.Status,
		&purchaseReturn.TotalAmount,
		&purchaseReturn.ReturnedAt,
		&createdBy,
		&purchaseReturn.CreatedAt,
		&purchaseReturn.UpdatedAt,
	); err != nil {
		return structs.PurchaseReturn{}, err
	}

	if reason.Valid {
		purchaseReturn.Reason = reason.String
	}
	if createdBy.Valid {
		purchaseReturn.CreatedBy = &createdBy.Int64
	}

	return purchaseReturn, nil
}
