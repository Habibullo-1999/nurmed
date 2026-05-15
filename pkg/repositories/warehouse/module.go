package warehouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"

	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	warehousesvc "nurmed/internal/warehouse"
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

func New(p Params) interfaces.WarehouseRepo {
	return &repo{
		logger: p.Logger,
		db:     p.Db,
	}
}

// ---- Warehouses ----

func (r *repo) ListWarehouses(ctx context.Context, companyID int64) ([]structs.Warehouse, error) {
	query := sqlx.Rebind(sqlx.DOLLAR, `
		SELECT id, company_id, name, COALESCE(address, ''), status, created_by, created_at, updated_at
		FROM warehouses
		WHERE company_id = ? AND status = 'active'
		ORDER BY name ASC`)

	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []structs.Warehouse
	for rows.Next() {
		var w structs.Warehouse
		var createdBy sql.NullInt64
		if err = rows.Scan(&w.ID, &w.CompanyID, &w.Name, &w.Address, &w.Status, &createdBy, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		if createdBy.Valid {
			w.CreatedBy = &createdBy.Int64
		}
		result = append(result, w)
	}
	return result, rows.Err()
}

// ---- Stock ----

func (r *repo) GetStock(ctx context.Context, filter structs.WarehouseStockFilter) ([]structs.WarehouseStockResponse, error) {
	w, v := stockFilter(filter)
	query := fmt.Sprintf(`
		SELECT
			ws.id,
			ws.warehouse_id,
			wh.name,
			ws.product_id,
			p.name,
			COALESCE(p.sku, ''),
			COALESCE(p.barcode, ''),
			p.unit,
			ws.quantity,
			ws.cost_price,
			ws.quantity * ws.cost_price
		FROM warehouse_stocks ws
		JOIN warehouses wh ON wh.id = ws.warehouse_id
		JOIN products p ON p.id = ws.product_id
		%s
		ORDER BY p.name ASC
		LIMIT ? OFFSET ?`, utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []structs.WarehouseStockResponse
	for rows.Next() {
		var s structs.WarehouseStockResponse
		if err = rows.Scan(
			&s.ID, &s.WarehouseID, &s.WarehouseName,
			&s.ProductID, &s.ProductName, &s.SKU, &s.Barcode,
			&s.Unit, &s.Quantity, &s.CostPrice, &s.TotalCostPrice,
		); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func stockFilter(f structs.WarehouseStockFilter) (w []string, v []interface{}) {
	w = append(w, "ws.quantity > 0")
	if f.CompanyID != 0 {
		w = append(w, "ws.company_id = ?")
		v = append(v, f.CompanyID)
	}
	if f.WarehouseID != 0 {
		w = append(w, "ws.warehouse_id = ?")
		v = append(v, f.WarehouseID)
	}
	if f.Search != "" {
		escaped := escapeLike(f.Search)
		w = append(w, `(p.name ILIKE ? ESCAPE '\' OR p.barcode = ? OR p.sku = ?)`)
		v = append(v, "%"+escaped+"%", f.Search, f.Search)
	}
	return
}

// ---- Inventory ----

func (r *repo) ListInventories(ctx context.Context, filter structs.InventoryOrderFilter) ([]structs.InventoryOrderResponse, error) {
	w, v := inventoryListFilter(filter)
	query := fmt.Sprintf(`
		SELECT
			io.id, io.company_id, io.warehouse_id, wh.name,
			io.document_no, io.status, io.surplus_deficit,
			COALESCE(io.note, ''),
			io.created_by,
			COALESCE(TRIM(cu.first_name || ' ' || COALESCE(cu.last_name, '')), ''),
			io.updated_by,
			COALESCE(TRIM(uu.first_name || ' ' || COALESCE(uu.last_name, '')), ''),
			io.created_at, io.updated_at
		FROM inventory_orders io
		JOIN warehouses wh ON wh.id = io.warehouse_id
		LEFT JOIN users cu ON cu.id = io.created_by
		LEFT JOIN users uu ON uu.id = io.updated_by
		%s
		ORDER BY io.created_at DESC
		LIMIT ? OFFSET ?`, utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []structs.InventoryOrderResponse
	for rows.Next() {
		resp, scanErr := scanInventoryRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, resp)
	}
	return result, rows.Err()
}

func (r *repo) CreateInventory(ctx context.Context, order structs.InventoryOrder, items []structs.InventoryOrderItem) (structs.InventoryOrder, []structs.InventoryOrderItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return structs.InventoryOrder{}, nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	now := time.Now().UTC()
	var createdOrder structs.InventoryOrder
	var createdBy, updatedBy sql.NullInt64

	err = tx.QueryRow(ctx, `
		INSERT INTO inventory_orders (company_id, warehouse_id, document_no, status, surplus_deficit, note, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,0,$5,$6,$7,$7)
		RETURNING id, company_id, warehouse_id, document_no, status, surplus_deficit, COALESCE(note,''), created_by, updated_by, created_at, updated_at`,
		order.CompanyID, order.WarehouseID, order.DocumentNo, order.Status,
		nullableStr(order.Note), nullableInt64Ptr(order.CreatedBy), now,
	).Scan(
		&createdOrder.ID, &createdOrder.CompanyID, &createdOrder.WarehouseID,
		&createdOrder.DocumentNo, &createdOrder.Status, &createdOrder.SurplusDeficit,
		&createdOrder.Note, &createdBy, &updatedBy,
		&createdOrder.CreatedAt, &createdOrder.UpdatedAt,
	)
	if err != nil {
		return structs.InventoryOrder{}, nil, err
	}
	if createdBy.Valid {
		createdOrder.CreatedBy = &createdBy.Int64
	}

	createdItems := make([]structs.InventoryOrderItem, 0, len(items))
	for _, item := range items {
		var ci structs.InventoryOrderItem
		var pid sql.NullInt64
		err = tx.QueryRow(ctx, `
			INSERT INTO inventory_order_items (order_id, product_id, product_name, expected_qty, actual_qty, cost_price, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			RETURNING id, order_id, product_id, product_name, expected_qty, actual_qty, cost_price, created_at`,
			createdOrder.ID, nullableInt64Ptr(item.ProductID), item.ProductName,
			item.ExpectedQty, item.ActualQty, item.CostPrice, now,
		).Scan(&ci.ID, &ci.OrderID, &pid, &ci.ProductName, &ci.ExpectedQty, &ci.ActualQty, &ci.CostPrice, &ci.CreatedAt)
		if err != nil {
			return structs.InventoryOrder{}, nil, err
		}
		if pid.Valid {
			ci.ProductID = &pid.Int64
		}
		createdItems = append(createdItems, ci)
	}

	if err = tx.Commit(ctx); err != nil {
		return structs.InventoryOrder{}, nil, err
	}
	committed = true
	return createdOrder, createdItems, nil
}

func (r *repo) GetInventory(ctx context.Context, id int64, companyID int64) (structs.InventoryOrderResponse, error) {
	query := sqlx.Rebind(sqlx.DOLLAR, `
		SELECT
			io.id, io.company_id, io.warehouse_id, wh.name,
			io.document_no, io.status, io.surplus_deficit,
			COALESCE(io.note, ''),
			io.created_by,
			COALESCE(TRIM(cu.first_name || ' ' || COALESCE(cu.last_name, '')), ''),
			io.updated_by,
			COALESCE(TRIM(uu.first_name || ' ' || COALESCE(uu.last_name, '')), ''),
			io.created_at, io.updated_at
		FROM inventory_orders io
		JOIN warehouses wh ON wh.id = io.warehouse_id
		LEFT JOIN users cu ON cu.id = io.created_by
		LEFT JOIN users uu ON uu.id = io.updated_by
		WHERE io.id = ? AND io.company_id = ?`)

	resp, err := scanInventoryRow(r.db.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return structs.InventoryOrderResponse{}, err
	}

	itemRows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, `
		SELECT id, order_id, product_id, product_name, expected_qty, actual_qty, cost_price, created_at
		FROM inventory_order_items WHERE order_id = ? ORDER BY id ASC`), id)
	if err != nil {
		return structs.InventoryOrderResponse{}, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item structs.InventoryOrderItemResponse
		var pid sql.NullInt64
		var orderID int64
		if err = itemRows.Scan(&item.ID, &orderID, &pid, &item.ProductName, &item.ExpectedQty, &item.ActualQty, &item.CostPrice, new(time.Time)); err != nil {
			return structs.InventoryOrderResponse{}, err
		}
		if pid.Valid {
			item.ProductID = &pid.Int64
		}
		resp.Items = append(resp.Items, item)
	}
	if err = itemRows.Err(); err != nil {
		return structs.InventoryOrderResponse{}, err
	}
	return resp, nil
}

func (r *repo) PostInventory(ctx context.Context, id int64, companyID int64, updatedBy *int64) (structs.InventoryOrderResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return structs.InventoryOrderResponse{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var currentStatus string
	var warehouseID int64
	if err = tx.QueryRow(ctx, `SELECT status, warehouse_id FROM inventory_orders WHERE id = $1 AND company_id = $2 FOR UPDATE`, id, companyID).
		Scan(&currentStatus, &warehouseID); err != nil {
		return structs.InventoryOrderResponse{}, err
	}
	if currentStatus != structs.InventoryStatusDraft {
		return structs.InventoryOrderResponse{}, warehousesvc.ErrOrderAlreadyPosted
	}

	type itemRow struct {
		productID   *int64
		expectedQty float64
		actualQty   float64
		costPrice   float64
	}
	rows, err := tx.Query(ctx, `SELECT product_id, expected_qty, actual_qty, cost_price FROM inventory_order_items WHERE order_id = $1`, id)
	if err != nil {
		return structs.InventoryOrderResponse{}, err
	}

	var items []itemRow
	surplusDeficit := 0.0
	for rows.Next() {
		var item itemRow
		var pid sql.NullInt64
		if err = rows.Scan(&pid, &item.expectedQty, &item.actualQty, &item.costPrice); err != nil {
			rows.Close()
			return structs.InventoryOrderResponse{}, err
		}
		if pid.Valid {
			item.productID = &pid.Int64
		}
		surplusDeficit += (item.actualQty - item.expectedQty) * item.costPrice
		items = append(items, item)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return structs.InventoryOrderResponse{}, err
	}

	for _, item := range items {
		if item.productID == nil {
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_stocks (company_id, warehouse_id, product_id, quantity, cost_price, updated_at)
			VALUES (
				(SELECT company_id FROM warehouses WHERE id = $1),
				$1, $2, $3, $4, NOW()
			)
			ON CONFLICT (warehouse_id, product_id) DO UPDATE
				SET quantity = EXCLUDED.quantity,
				    cost_price = EXCLUDED.cost_price,
				    updated_at = NOW()`,
			warehouseID, *item.productID, item.actualQty, item.costPrice)
		if err != nil {
			return structs.InventoryOrderResponse{}, err
		}
	}

	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		UPDATE inventory_orders
		SET status = $1, surplus_deficit = $2, updated_by = $3, updated_at = $4
		WHERE id = $5`,
		structs.InventoryStatusPosted, surplusDeficit, nullableInt64Ptr(updatedBy), now, id)
	if err != nil {
		return structs.InventoryOrderResponse{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return structs.InventoryOrderResponse{}, err
	}
	committed = true

	return r.GetInventory(ctx, id, companyID)
}

// ---- Transfer ----

func (r *repo) ListTransfers(ctx context.Context, filter structs.TransferOrderFilter) ([]structs.TransferOrderResponse, error) {
	w, v := transferListFilter(filter)
	query := fmt.Sprintf(`
		SELECT
			t.id, t.company_id, t.document_no,
			t.from_warehouse_id, fw.name,
			t.to_warehouse_id, tw.name,
			t.status, COALESCE(t.note, ''),
			t.transferred_at, t.received_at,
			t.created_by, t.created_at, t.updated_at
		FROM transfer_orders t
		JOIN warehouses fw ON fw.id = t.from_warehouse_id
		JOIN warehouses tw ON tw.id = t.to_warehouse_id
		%s
		ORDER BY t.transferred_at DESC
		LIMIT ? OFFSET ?`, utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []structs.TransferOrderResponse
	for rows.Next() {
		resp, scanErr := scanTransferRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, resp)
	}
	return result, rows.Err()
}

func (r *repo) CreateTransfer(ctx context.Context, order structs.TransferOrder, items []structs.TransferOrderItem) (structs.TransferOrder, []structs.TransferOrderItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return structs.TransferOrder{}, nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	now := time.Now().UTC()
	var createdOrder structs.TransferOrder
	var createdBy sql.NullInt64

	err = tx.QueryRow(ctx, `
		INSERT INTO transfer_orders (company_id, document_no, from_warehouse_id, to_warehouse_id, status, note, transferred_at, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
		RETURNING id, company_id, document_no, from_warehouse_id, to_warehouse_id, status, COALESCE(note,''), transferred_at, received_at, created_by, created_at, updated_at`,
		order.CompanyID, order.DocumentNo, order.FromWarehouseID, order.ToWarehouseID,
		order.Status, nullableStr(order.Note), order.TransferredAt,
		nullableInt64Ptr(order.CreatedBy), now,
	).Scan(
		&createdOrder.ID, &createdOrder.CompanyID, &createdOrder.DocumentNo,
		&createdOrder.FromWarehouseID, &createdOrder.ToWarehouseID,
		&createdOrder.Status, &createdOrder.Note,
		&createdOrder.TransferredAt, &createdOrder.ReceivedAt,
		&createdBy, &createdOrder.CreatedAt, &createdOrder.UpdatedAt,
	)
	if err != nil {
		return structs.TransferOrder{}, nil, err
	}
	if createdBy.Valid {
		createdOrder.CreatedBy = &createdBy.Int64
	}

	createdItems := make([]structs.TransferOrderItem, 0, len(items))
	for _, item := range items {
		var ci structs.TransferOrderItem
		var pid sql.NullInt64
		err = tx.QueryRow(ctx, `
			INSERT INTO transfer_order_items (order_id, product_id, product_name, quantity, cost_price, created_at)
			VALUES ($1,$2,$3,$4,$5,$6)
			RETURNING id, order_id, product_id, product_name, quantity, cost_price, created_at`,
			createdOrder.ID, nullableInt64Ptr(item.ProductID), item.ProductName,
			item.Quantity, item.CostPrice, now,
		).Scan(&ci.ID, &ci.OrderID, &pid, &ci.ProductName, &ci.Quantity, &ci.CostPrice, &ci.CreatedAt)
		if err != nil {
			return structs.TransferOrder{}, nil, err
		}
		if pid.Valid {
			ci.ProductID = &pid.Int64
		}
		createdItems = append(createdItems, ci)
	}

	if err = tx.Commit(ctx); err != nil {
		return structs.TransferOrder{}, nil, err
	}
	committed = true
	return createdOrder, createdItems, nil
}

func (r *repo) GetTransfer(ctx context.Context, id int64, companyID int64) (structs.TransferOrderResponse, error) {
	query := sqlx.Rebind(sqlx.DOLLAR, `
		SELECT
			t.id, t.company_id, t.document_no,
			t.from_warehouse_id, fw.name,
			t.to_warehouse_id, tw.name,
			t.status, COALESCE(t.note, ''),
			t.transferred_at, t.received_at,
			t.created_by, t.created_at, t.updated_at
		FROM transfer_orders t
		JOIN warehouses fw ON fw.id = t.from_warehouse_id
		JOIN warehouses tw ON tw.id = t.to_warehouse_id
		WHERE t.id = ? AND t.company_id = ?`)

	resp, err := scanTransferRow(r.db.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return structs.TransferOrderResponse{}, err
	}

	itemRows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, `
		SELECT id, order_id, product_id, product_name, quantity, cost_price, created_at
		FROM transfer_order_items WHERE order_id = ? ORDER BY id ASC`), id)
	if err != nil {
		return structs.TransferOrderResponse{}, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item structs.TransferOrderItemResponse
		var pid sql.NullInt64
		var orderID int64
		if err = itemRows.Scan(&item.ID, &orderID, &pid, &item.ProductName, &item.Quantity, &item.CostPrice, new(time.Time)); err != nil {
			return structs.TransferOrderResponse{}, err
		}
		if pid.Valid {
			item.ProductID = &pid.Int64
		}
		resp.Items = append(resp.Items, item)
	}
	if err = itemRows.Err(); err != nil {
		return structs.TransferOrderResponse{}, err
	}
	return resp, nil
}

func (r *repo) PostTransfer(ctx context.Context, id int64, companyID int64) (structs.TransferOrderResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return structs.TransferOrderResponse{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var currentStatus string
	var fromWarehouseID, toWarehouseID int64
	if err = tx.QueryRow(ctx, `
		SELECT status, from_warehouse_id, to_warehouse_id
		FROM transfer_orders WHERE id = $1 AND company_id = $2 FOR UPDATE`, id, companyID).
		Scan(&currentStatus, &fromWarehouseID, &toWarehouseID); err != nil {
		return structs.TransferOrderResponse{}, err
	}
	if currentStatus != structs.TransferStatusDraft {
		return structs.TransferOrderResponse{}, warehousesvc.ErrOrderAlreadyPosted
	}

	type transferItem struct {
		productID   *int64
		productName string
		quantity    float64
		costPrice   float64
	}
	rows, err := tx.Query(ctx, `SELECT product_id, product_name, quantity, cost_price FROM transfer_order_items WHERE order_id = $1`, id)
	if err != nil {
		return structs.TransferOrderResponse{}, err
	}

	var items []transferItem
	for rows.Next() {
		var item transferItem
		var pid sql.NullInt64
		if err = rows.Scan(&pid, &item.productName, &item.quantity, &item.costPrice); err != nil {
			rows.Close()
			return structs.TransferOrderResponse{}, err
		}
		if pid.Valid {
			item.productID = &pid.Int64
		}
		items = append(items, item)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return structs.TransferOrderResponse{}, err
	}

	for _, item := range items {
		if item.productID == nil {
			continue
		}

		var currentQty float64
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(quantity, 0) FROM warehouse_stocks
			WHERE warehouse_id = $1 AND product_id = $2 FOR UPDATE`,
			fromWarehouseID, *item.productID).Scan(&currentQty)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return structs.TransferOrderResponse{}, err
		}
		if currentQty < item.quantity {
			return structs.TransferOrderResponse{}, warehousesvc.ErrInsufficientStock
		}

		_, err = tx.Exec(ctx, `
			UPDATE warehouse_stocks SET quantity = quantity - $1, updated_at = NOW()
			WHERE warehouse_id = $2 AND product_id = $3`,
			item.quantity, fromWarehouseID, *item.productID)
		if err != nil {
			return structs.TransferOrderResponse{}, err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO warehouse_stocks (company_id, warehouse_id, product_id, quantity, cost_price, updated_at)
			VALUES (
				(SELECT company_id FROM warehouses WHERE id = $1),
				$1, $2, $3, $4, NOW()
			)
			ON CONFLICT (warehouse_id, product_id) DO UPDATE
				SET quantity = warehouse_stocks.quantity + EXCLUDED.quantity,
				    updated_at = NOW()`,
			toWarehouseID, *item.productID, item.quantity, item.costPrice)
		if err != nil {
			return structs.TransferOrderResponse{}, err
		}
	}

	_, err = tx.Exec(ctx, `UPDATE transfer_orders SET status = $1, updated_at = NOW() WHERE id = $2`,
		structs.TransferStatusPosted, id)
	if err != nil {
		return structs.TransferOrderResponse{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return structs.TransferOrderResponse{}, err
	}
	committed = true

	return r.GetTransfer(ctx, id, companyID)
}

// ---- Writeoff ----

func (r *repo) ListWriteoffs(ctx context.Context, filter structs.WriteoffOrderFilter) ([]structs.WriteoffOrderResponse, error) {
	w, v := writeoffListFilter(filter)
	query := fmt.Sprintf(`
		SELECT
			wo.id, wo.company_id, wo.warehouse_id, wh.name,
			wo.document_no, wo.status, wo.total_amount,
			COALESCE(wo.object_name, ''), COALESCE(wo.counterparty_name, ''),
			COALESCE(wo.note, ''), wo.written_off_at,
			wo.created_by,
			COALESCE(TRIM(u.first_name || ' ' || COALESCE(u.last_name, '')), ''),
			wo.created_at, wo.updated_at
		FROM writeoff_orders wo
		JOIN warehouses wh ON wh.id = wo.warehouse_id
		LEFT JOIN users u ON u.id = wo.created_by
		%s
		ORDER BY wo.written_off_at DESC
		LIMIT ? OFFSET ?`, utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []structs.WriteoffOrderResponse
	for rows.Next() {
		resp, scanErr := scanWriteoffRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, resp)
	}
	return result, rows.Err()
}

func (r *repo) CreateWriteoff(ctx context.Context, order structs.WriteoffOrder, items []structs.WriteoffOrderItem) (structs.WriteoffOrder, []structs.WriteoffOrderItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return structs.WriteoffOrder{}, nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	now := time.Now().UTC()
	var createdOrder structs.WriteoffOrder
	var createdBy sql.NullInt64

	err = tx.QueryRow(ctx, `
		INSERT INTO writeoff_orders (company_id, warehouse_id, document_no, status, total_amount, object_name, counterparty_name, note, written_off_at, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)
		RETURNING id, company_id, warehouse_id, document_no, status, total_amount,
		          COALESCE(object_name,''), COALESCE(counterparty_name,''), COALESCE(note,''),
		          written_off_at, created_by, created_at, updated_at`,
		order.CompanyID, order.WarehouseID, order.DocumentNo, order.Status, order.TotalAmount,
		nullableStr(order.ObjectName), nullableStr(order.CounterpartyName), nullableStr(order.Note),
		order.WrittenOffAt, nullableInt64Ptr(order.CreatedBy), now,
	).Scan(
		&createdOrder.ID, &createdOrder.CompanyID, &createdOrder.WarehouseID,
		&createdOrder.DocumentNo, &createdOrder.Status, &createdOrder.TotalAmount,
		&createdOrder.ObjectName, &createdOrder.CounterpartyName, &createdOrder.Note,
		&createdOrder.WrittenOffAt, &createdBy,
		&createdOrder.CreatedAt, &createdOrder.UpdatedAt,
	)
	if err != nil {
		return structs.WriteoffOrder{}, nil, err
	}
	if createdBy.Valid {
		createdOrder.CreatedBy = &createdBy.Int64
	}

	createdItems := make([]structs.WriteoffOrderItem, 0, len(items))
	for _, item := range items {
		var ci structs.WriteoffOrderItem
		var pid sql.NullInt64
		err = tx.QueryRow(ctx, `
			INSERT INTO writeoff_order_items (order_id, product_id, product_name, quantity, cost_price, amount, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			RETURNING id, order_id, product_id, product_name, quantity, cost_price, amount, created_at`,
			createdOrder.ID, nullableInt64Ptr(item.ProductID), item.ProductName,
			item.Quantity, item.CostPrice, item.Amount, now,
		).Scan(&ci.ID, &ci.OrderID, &pid, &ci.ProductName, &ci.Quantity, &ci.CostPrice, &ci.Amount, &ci.CreatedAt)
		if err != nil {
			return structs.WriteoffOrder{}, nil, err
		}
		if pid.Valid {
			ci.ProductID = &pid.Int64
		}
		createdItems = append(createdItems, ci)
	}

	if err = tx.Commit(ctx); err != nil {
		return structs.WriteoffOrder{}, nil, err
	}
	committed = true
	return createdOrder, createdItems, nil
}

func (r *repo) GetWriteoff(ctx context.Context, id int64, companyID int64) (structs.WriteoffOrderResponse, error) {
	query := sqlx.Rebind(sqlx.DOLLAR, `
		SELECT
			wo.id, wo.company_id, wo.warehouse_id, wh.name,
			wo.document_no, wo.status, wo.total_amount,
			COALESCE(wo.object_name, ''), COALESCE(wo.counterparty_name, ''),
			COALESCE(wo.note, ''), wo.written_off_at,
			wo.created_by,
			COALESCE(TRIM(u.first_name || ' ' || COALESCE(u.last_name, '')), ''),
			wo.created_at, wo.updated_at
		FROM writeoff_orders wo
		JOIN warehouses wh ON wh.id = wo.warehouse_id
		LEFT JOIN users u ON u.id = wo.created_by
		WHERE wo.id = ? AND wo.company_id = ?`)

	resp, err := scanWriteoffRow(r.db.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return structs.WriteoffOrderResponse{}, err
	}

	itemRows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, `
		SELECT id, order_id, product_id, product_name, quantity, cost_price, amount, created_at
		FROM writeoff_order_items WHERE order_id = ? ORDER BY id ASC`), id)
	if err != nil {
		return structs.WriteoffOrderResponse{}, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item structs.WriteoffOrderItemResponse
		var pid sql.NullInt64
		var orderID int64
		if err = itemRows.Scan(&item.ID, &orderID, &pid, &item.ProductName, &item.Quantity, &item.CostPrice, &item.Amount, new(time.Time)); err != nil {
			return structs.WriteoffOrderResponse{}, err
		}
		if pid.Valid {
			item.ProductID = &pid.Int64
		}
		resp.Items = append(resp.Items, item)
	}
	if err = itemRows.Err(); err != nil {
		return structs.WriteoffOrderResponse{}, err
	}
	return resp, nil
}

func (r *repo) PostWriteoff(ctx context.Context, id int64, companyID int64) (structs.WriteoffOrderResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return structs.WriteoffOrderResponse{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var currentStatus string
	var warehouseID int64
	if err = tx.QueryRow(ctx, `SELECT status, warehouse_id FROM writeoff_orders WHERE id = $1 AND company_id = $2 FOR UPDATE`, id, companyID).
		Scan(&currentStatus, &warehouseID); err != nil {
		return structs.WriteoffOrderResponse{}, err
	}
	if currentStatus != structs.WriteoffStatusDraft {
		return structs.WriteoffOrderResponse{}, warehousesvc.ErrOrderAlreadyPosted
	}

	type writeoffItem struct {
		productID *int64
		quantity  float64
	}
	rows, err := tx.Query(ctx, `SELECT product_id, quantity FROM writeoff_order_items WHERE order_id = $1`, id)
	if err != nil {
		return structs.WriteoffOrderResponse{}, err
	}

	var items []writeoffItem
	for rows.Next() {
		var item writeoffItem
		var pid sql.NullInt64
		if err = rows.Scan(&pid, &item.quantity); err != nil {
			rows.Close()
			return structs.WriteoffOrderResponse{}, err
		}
		if pid.Valid {
			item.productID = &pid.Int64
		}
		items = append(items, item)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return structs.WriteoffOrderResponse{}, err
	}

	for _, item := range items {
		if item.productID == nil {
			continue
		}

		var currentQty float64
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(quantity, 0) FROM warehouse_stocks
			WHERE warehouse_id = $1 AND product_id = $2 FOR UPDATE`,
			warehouseID, *item.productID).Scan(&currentQty)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return structs.WriteoffOrderResponse{}, err
		}
		if currentQty < item.quantity {
			return structs.WriteoffOrderResponse{}, warehousesvc.ErrInsufficientStock
		}

		_, err = tx.Exec(ctx, `
			UPDATE warehouse_stocks SET quantity = quantity - $1, updated_at = NOW()
			WHERE warehouse_id = $2 AND product_id = $3`,
			item.quantity, warehouseID, *item.productID)
		if err != nil {
			return structs.WriteoffOrderResponse{}, err
		}
	}

	_, err = tx.Exec(ctx, `UPDATE writeoff_orders SET status = $1, updated_at = NOW() WHERE id = $2`,
		structs.WriteoffStatusPosted, id)
	if err != nil {
		return structs.WriteoffOrderResponse{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return structs.WriteoffOrderResponse{}, err
	}
	committed = true

	return r.GetWriteoff(ctx, id, companyID)
}

// ---- Filter builders ----

func inventoryListFilter(f structs.InventoryOrderFilter) (w []string, v []interface{}) {
	if f.ID != 0 {
		w = append(w, "io.id = ?")
		v = append(v, f.ID)
	}
	if f.CompanyID != 0 {
		w = append(w, "io.company_id = ?")
		v = append(v, f.CompanyID)
	}
	if f.WarehouseID != 0 {
		w = append(w, "io.warehouse_id = ?")
		v = append(v, f.WarehouseID)
	}
	if f.Status != "" {
		w = append(w, "io.status = ?")
		v = append(v, f.Status)
	}
	if f.DateFrom != nil {
		w = append(w, "io.created_at >= ?")
		v = append(v, f.DateFrom)
	}
	if f.DateTo != nil {
		w = append(w, "io.created_at < ?")
		v = append(v, f.DateTo.Add(24*time.Hour))
	}
	if f.Search != "" {
		w = append(w, `io.document_no ILIKE ? ESCAPE '\'`)
		v = append(v, "%"+escapeLike(f.Search)+"%")
	}
	return
}

func transferListFilter(f structs.TransferOrderFilter) (w []string, v []interface{}) {
	if f.ID != 0 {
		w = append(w, "t.id = ?")
		v = append(v, f.ID)
	}
	if f.CompanyID != 0 {
		w = append(w, "t.company_id = ?")
		v = append(v, f.CompanyID)
	}
	if f.FromWarehouseID != 0 {
		w = append(w, "t.from_warehouse_id = ?")
		v = append(v, f.FromWarehouseID)
	}
	if f.ToWarehouseID != 0 {
		w = append(w, "t.to_warehouse_id = ?")
		v = append(v, f.ToWarehouseID)
	}
	if f.Status != "" {
		w = append(w, "t.status = ?")
		v = append(v, f.Status)
	}
	if f.DateFrom != nil {
		w = append(w, "t.transferred_at >= ?")
		v = append(v, f.DateFrom)
	}
	if f.DateTo != nil {
		w = append(w, "t.transferred_at < ?")
		v = append(v, f.DateTo.Add(24*time.Hour))
	}
	return
}

func writeoffListFilter(f structs.WriteoffOrderFilter) (w []string, v []interface{}) {
	if f.ID != 0 {
		w = append(w, "wo.id = ?")
		v = append(v, f.ID)
	}
	if f.CompanyID != 0 {
		w = append(w, "wo.company_id = ?")
		v = append(v, f.CompanyID)
	}
	if f.WarehouseID != 0 {
		w = append(w, "wo.warehouse_id = ?")
		v = append(v, f.WarehouseID)
	}
	if f.Status != "" {
		w = append(w, "wo.status = ?")
		v = append(v, f.Status)
	}
	if f.DateFrom != nil {
		w = append(w, "wo.written_off_at >= ?")
		v = append(v, f.DateFrom)
	}
	if f.DateTo != nil {
		w = append(w, "wo.written_off_at < ?")
		v = append(v, f.DateTo.Add(24*time.Hour))
	}
	return
}

// ---- Scan helpers ----

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanInventoryRow(s rowScanner) (structs.InventoryOrderResponse, error) {
	var r structs.InventoryOrderResponse
	var createdBy, updatedBy sql.NullInt64
	var createdByName, updatedByName string
	err := s.Scan(
		&r.ID, &r.CompanyID, &r.WarehouseID, &r.WarehouseName,
		&r.DocumentNo, &r.Status, &r.SurplusDeficit, &r.Note,
		&createdBy, &createdByName,
		&updatedBy, &updatedByName,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return structs.InventoryOrderResponse{}, err
	}
	if createdBy.Valid {
		r.CreatedBy = &createdBy.Int64
		r.CreatedByName = createdByName
	}
	if updatedBy.Valid {
		r.UpdatedBy = &updatedBy.Int64
		r.UpdatedByName = updatedByName
	}
	return r, nil
}

func scanTransferRow(s rowScanner) (structs.TransferOrderResponse, error) {
	var r structs.TransferOrderResponse
	var createdBy sql.NullInt64
	var receivedAt sql.NullTime
	err := s.Scan(
		&r.ID, &r.CompanyID, &r.DocumentNo,
		&r.FromWarehouseID, &r.FromWarehouseName,
		&r.ToWarehouseID, &r.ToWarehouseName,
		&r.Status, &r.Note,
		&r.TransferredAt, &receivedAt,
		&createdBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return structs.TransferOrderResponse{}, err
	}
	if createdBy.Valid {
		r.CreatedBy = &createdBy.Int64
	}
	if receivedAt.Valid {
		r.ReceivedAt = &receivedAt.Time
	}
	return r, nil
}

func scanWriteoffRow(s rowScanner) (structs.WriteoffOrderResponse, error) {
	var r structs.WriteoffOrderResponse
	var createdBy sql.NullInt64
	var createdByName string
	err := s.Scan(
		&r.ID, &r.CompanyID, &r.WarehouseID, &r.WarehouseName,
		&r.DocumentNo, &r.Status, &r.TotalAmount,
		&r.ObjectName, &r.CounterpartyName, &r.Note,
		&r.WrittenOffAt,
		&createdBy, &createdByName,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return structs.WriteoffOrderResponse{}, err
	}
	if createdBy.Valid {
		r.CreatedBy = &createdBy.Int64
		r.CreatedByName = createdByName
	}
	return r, nil
}

// ---- Utilities ----

func nullableStr(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

func nullableInt64Ptr(v *int64) interface{} {
	if v == nil || *v == 0 {
		return nil
	}
	return *v
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
