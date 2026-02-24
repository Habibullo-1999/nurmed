package sales

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

func New(p Params) interfaces.SalesRepo {
	return &repo{
		logger: p.Logger,
		db:     p.Db,
	}
}

func orderListFilter(filter structs.SalesOrderFilter, channel *string) (w []string, v []interface{}) {
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
	if filter.CustomerName != "" {
		w = append(w, "customer_name ILIKE ?")
		v = append(v, "%"+filter.CustomerName+"%")
	}
	if filter.Status != "" {
		w = append(w, "status = ?")
		v = append(v, filter.Status)
	}
	if channel != nil {
		w = append(w, "channel = ?")
		v = append(v, *channel)
	}

	return
}

func (r repo) ListOrders(ctx context.Context, filter structs.SalesOrderFilter, channel *string) ([]structs.SalesOrder, error) {
	w, v := orderListFilter(filter, channel)

	query := fmt.Sprintf(`SELECT %s FROM sales_orders %s ORDER BY sold_at DESC LIMIT ? OFFSET ?;`, orderColumns(), utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return []structs.SalesOrder{}, err
	}
	defer rows.Close()

	var orders []structs.SalesOrder
	for rows.Next() {
		order, scanErr := scanSalesOrder(rows)
		if scanErr != nil {
			return []structs.SalesOrder{}, scanErr
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (r repo) CreateOrder(ctx context.Context, order structs.SalesOrder, items []structs.SalesOrderItem) (structs.SalesOrder, []structs.SalesOrderItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return structs.SalesOrder{}, nil, err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	now := time.Now().UTC()
	customerName := nullableString(order.CustomerName)
	createdBy := nullableInt64Pointer(order.CreatedBy)

	createOrderQuery := `INSERT INTO sales_orders (
		company_id,
		channel,
		document_no,
		customer_name,
		currency,
		status,
		total_amount,
		item_count,
		sold_at,
		created_by,
		created_at,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
	RETURNING
		id,
		company_id,
		channel,
		document_no,
		customer_name,
		currency,
		status,
		total_amount,
		item_count,
		sold_at,
		created_by,
		created_at,
		updated_at;`

	createdOrder, err := scanSalesOrder(tx.QueryRow(
		ctx,
		createOrderQuery,
		order.CompanyID,
		order.Channel,
		order.DocumentNo,
		customerName,
		order.Currency,
		order.Status,
		order.TotalAmount,
		order.ItemCount,
		order.SoldAt,
		createdBy,
		now,
	))
	if err != nil {
		return structs.SalesOrder{}, nil, err
	}

	createdItems := make([]structs.SalesOrderItem, 0, len(items))
	for _, item := range items {
		productID := nullableInt64Pointer(item.ProductID)

		createdItem, itemErr := scanSalesOrderItem(tx.QueryRow(ctx, `INSERT INTO sales_order_items (
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
			return structs.SalesOrder{}, nil, itemErr
		}

		createdItems = append(createdItems, createdItem)
	}

	if err = tx.Commit(ctx); err != nil {
		return structs.SalesOrder{}, nil, err
	}
	committed = true

	return createdOrder, createdItems, nil
}

func returnListFilter(filter structs.SalesReturnFilter) (w []string, v []interface{}) {
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

func (r repo) ListReturns(ctx context.Context, filter structs.SalesReturnFilter) ([]structs.SalesReturn, error) {
	w, v := returnListFilter(filter)

	query := fmt.Sprintf(`SELECT %s FROM sales_returns %s ORDER BY returned_at DESC LIMIT ? OFFSET ?;`, returnColumns(), utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return []structs.SalesReturn{}, err
	}
	defer rows.Close()

	var returns []structs.SalesReturn
	for rows.Next() {
		salesReturn, scanErr := scanSalesReturn(rows)
		if scanErr != nil {
			return []structs.SalesReturn{}, scanErr
		}
		returns = append(returns, salesReturn)
	}

	return returns, nil
}

func (r repo) CreateReturn(ctx context.Context, salesReturn structs.SalesReturn) (structs.SalesReturn, error) {
	now := time.Now().UTC()
	reason := nullableString(salesReturn.Reason)
	createdBy := nullableInt64Pointer(salesReturn.CreatedBy)

	createQuery := `INSERT INTO sales_returns (
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

	return scanSalesReturn(r.db.QueryRow(
		ctx,
		createQuery,
		salesReturn.OrderID,
		salesReturn.CompanyID,
		reason,
		salesReturn.Status,
		salesReturn.TotalAmount,
		salesReturn.ReturnedAt,
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

func nullableInt64Pointer(v *int64) interface{} {
	if v == nil || *v == 0 {
		return nil
	}
	return *v
}

func orderColumns() string {
	return `id,
		company_id,
		channel,
		document_no,
		customer_name,
		currency,
		status,
		total_amount,
		item_count,
		sold_at,
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

func scanSalesOrder(scanner rowScanner) (structs.SalesOrder, error) {
	var (
		order        structs.SalesOrder
		customerName sql.NullString
		createdBy    sql.NullInt64
	)

	if err := scanner.Scan(
		&order.ID,
		&order.CompanyID,
		&order.Channel,
		&order.DocumentNo,
		&customerName,
		&order.Currency,
		&order.Status,
		&order.TotalAmount,
		&order.ItemCount,
		&order.SoldAt,
		&createdBy,
		&order.CreatedAt,
		&order.UpdatedAt,
	); err != nil {
		return structs.SalesOrder{}, err
	}

	if customerName.Valid {
		order.CustomerName = customerName.String
	}
	if createdBy.Valid {
		order.CreatedBy = &createdBy.Int64
	}

	return order, nil
}

func scanSalesOrderItem(scanner rowScanner) (structs.SalesOrderItem, error) {
	var (
		item      structs.SalesOrderItem
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
		return structs.SalesOrderItem{}, err
	}

	if productID.Valid {
		item.ProductID = &productID.Int64
	}

	return item, nil
}

func scanSalesReturn(scanner rowScanner) (structs.SalesReturn, error) {
	var (
		salesReturn structs.SalesReturn
		reason      sql.NullString
		createdBy   sql.NullInt64
	)

	if err := scanner.Scan(
		&salesReturn.ID,
		&salesReturn.OrderID,
		&salesReturn.CompanyID,
		&reason,
		&salesReturn.Status,
		&salesReturn.TotalAmount,
		&salesReturn.ReturnedAt,
		&createdBy,
		&salesReturn.CreatedAt,
		&salesReturn.UpdatedAt,
	); err != nil {
		return structs.SalesReturn{}, err
	}

	if reason.Valid {
		salesReturn.Reason = reason.String
	}
	if createdBy.Valid {
		salesReturn.CreatedBy = &createdBy.Int64
	}

	return salesReturn, nil
}
