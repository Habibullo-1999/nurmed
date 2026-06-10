package accounting

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

func New(p Params) interfaces.AccountingRepo {
	return &repo{logger: p.Logger, db: p.Db}
}

// ─── Payment Documents ────────────────────────────────────────────────────────

func (r repo) ListPaymentDocuments(ctx context.Context, filter structs.PaymentDocumentFilter) ([]structs.PaymentDocument, error) {
	w, v := paymentDocFilter(filter)
	query := fmt.Sprintf(`SELECT %s FROM payment_documents %s ORDER BY doc_date DESC LIMIT ? OFFSET ?`, paymentDocColumns(), utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []structs.PaymentDocument
	for rows.Next() {
		d, scanErr := scanPaymentDocument(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (r repo) CreatePaymentDocument(ctx context.Context, doc structs.PaymentDocument) (structs.PaymentDocument, error) {
	now := time.Now().UTC()
	return scanPaymentDocument(r.db.QueryRow(ctx, `
		INSERT INTO payment_documents
			(company_id, document_no, doc_date, doc_type, debit_account, credit_account,
			 income, expense, category, note, organization, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13)
		RETURNING `+paymentDocColumns(),
		doc.CompanyID, doc.DocumentNo, doc.DocDate, doc.DocType,
		nullStr(doc.DebitAccount), nullStr(doc.CreditAccount),
		doc.Income, doc.Expense,
		nullStr(doc.Category), nullStr(doc.Note), nullStr(doc.Organization),
		nullInt64Ptr(doc.CreatedBy), now,
	))
}

// ─── Cash Registers / Account Statement ──────────────────────────────────────

func (r repo) ListCashRegisters(ctx context.Context, companyID int64) ([]structs.CashRegister, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, company_id, name, currency, created_at, updated_at FROM cash_registers WHERE company_id = $1 ORDER BY name`,
		companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []structs.CashRegister
	for rows.Next() {
		var cr structs.CashRegister
		if err := rows.Scan(&cr.ID, &cr.CompanyID, &cr.Name, &cr.Currency, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, cr)
	}
	return list, rows.Err()
}

func (r repo) GetCashRegisterBalances(ctx context.Context, filter structs.CashRegisterBalanceFilter) ([]structs.CashRegisterBalanceResponse, error) {
	dateCond := ""
	args := []interface{}{filter.CompanyID}
	if !filter.Date.IsZero() {
		dateCond = fmt.Sprintf(" AND ct.transacted_at <= $%d", len(args)+1)
		args = append(args, filter.Date)
	}

	query := fmt.Sprintf(`
		SELECT cr.id, cr.name, cr.currency,
		       COALESCE(SUM(ct.income), 0) - COALESCE(SUM(ct.expense), 0) AS balance
		FROM cash_registers cr
		LEFT JOIN cash_transactions ct ON ct.cash_register_id = cr.id%s
		WHERE cr.company_id = $1
		GROUP BY cr.id, cr.name, cr.currency
		ORDER BY cr.name`, dateCond)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []structs.CashRegisterBalanceResponse
	for rows.Next() {
		var b structs.CashRegisterBalanceResponse
		if err := rows.Scan(&b.ID, &b.Name, &b.Currency, &b.Balance); err != nil {
			return nil, err
		}
		b.BalanceInTJS = b.Balance // rate conversion handled at service level if needed
		list = append(list, b)
	}
	return list, rows.Err()
}

func (r repo) GetAccountStatement(ctx context.Context, filter structs.AccountStatementFilter) ([]structs.CashTransaction, error) {
	w, v := accountStatementFilter(filter)
	query := fmt.Sprintf(`
		SELECT id, company_id, cash_register_id, COALESCE(doc_type,''), COALESCE(doc_no,''),
		       income, expense, COALESCE(category,''), COALESCE(note,''), created_by,
		       transacted_at, created_at
		FROM cash_transactions %s ORDER BY transacted_at ASC LIMIT ? OFFSET ?`, utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []structs.CashTransaction
	for rows.Next() {
		var ct structs.CashTransaction
		var createdBy sql.NullInt64
		if err := rows.Scan(&ct.ID, &ct.CompanyID, &ct.CashRegisterID,
			&ct.DocType, &ct.DocNo, &ct.Income, &ct.Expense,
			&ct.Category, &ct.Note, &createdBy,
			&ct.TransactedAt, &ct.CreatedAt); err != nil {
			return nil, err
		}
		if createdBy.Valid {
			ct.CreatedBy = &createdBy.Int64
		}
		list = append(list, ct)
	}
	return list, rows.Err()
}

// ─── Counterparty Balance ─────────────────────────────────────────────────────

func (r repo) ListCounterpartyBalances(ctx context.Context, filter structs.CounterpartyBalanceFilter) ([]structs.CounterpartyBalanceResponse, error) {
	w, v := counterpartyFilter(filter)
	query := fmt.Sprintf(`
		SELECT c.id, c.name, COALESCE(c.group_type,''), COALESCE(c.phone,''),
		       COALESCE(c.region,''), c.currency,
		       COALESCE(SUM(ct.amount), 0) AS amount
		FROM counterparties c
		LEFT JOIN counterparty_transactions ct ON ct.counterparty_id = c.id
		%s
		GROUP BY c.id, c.name, c.group_type, c.phone, c.region, c.currency
		ORDER BY ABS(COALESCE(SUM(ct.amount),0)) DESC LIMIT ? OFFSET ?`,
		utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []structs.CounterpartyBalanceResponse
	no := filter.Offset + 1
	for rows.Next() {
		var b structs.CounterpartyBalanceResponse
		if err := rows.Scan(&b.ID, &b.Name, &b.GroupType, &b.Phone, &b.Region, &b.Currency, &b.Amount); err != nil {
			return nil, err
		}
		b.No = no
		b.AmountInTJS = b.Amount
		list = append(list, b)
		no++
	}
	return list, rows.Err()
}

// ─── Debt Records ─────────────────────────────────────────────────────────────

func (r repo) ListDebtRecords(ctx context.Context, filter structs.DebtRecordFilter) ([]structs.DebtRecord, error) {
	w, v := debtRecordFilter(filter)
	query := fmt.Sprintf(`SELECT %s FROM debt_records %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, debtColumns(), utils.Where(w))
	v = append(v, filter.Limit, filter.Offset)
	rows, err := r.db.Query(ctx, sqlx.Rebind(sqlx.DOLLAR, query), v...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []structs.DebtRecord
	for rows.Next() {
		d, scanErr := scanDebtRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r repo) CreateDebtRecord(ctx context.Context, record structs.DebtRecord) (structs.DebtRecord, error) {
	now := time.Now().UTC()
	return scanDebtRecord(r.db.QueryRow(ctx, `
		INSERT INTO debt_records
			(company_id, counterparty_id, client_name, phone, period, start_date,
			 next_payment_date, last_payment_date, balance, client_text, admin_text,
			 note, channels, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$15)
		RETURNING `+debtColumns(),
		record.CompanyID, nullInt64Ptr(record.CounterpartyID),
		nullStr(record.ClientName), nullStr(record.Phone), nullStr(record.Period),
		nullTimePtr(record.StartDate), nullTimePtr(record.NextPaymentDate), nullTimePtr(record.LastPaymentDate),
		record.Balance,
		nullStr(record.ClientText), nullStr(record.AdminText), nullStr(record.Note), nullStr(record.Channels),
		record.Status, now,
	))
}

// ─── Price Lists ──────────────────────────────────────────────────────────────

func (r repo) ListPriceLists(ctx context.Context, companyID int64) ([]structs.PriceList, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, company_id, name, COALESCE(note,''), price_list_type, created_at, updated_at
		 FROM price_lists WHERE company_id = $1 ORDER BY id`,
		companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []structs.PriceList
	for rows.Next() {
		var pl structs.PriceList
		if err := rows.Scan(&pl.ID, &pl.CompanyID, &pl.Name, &pl.Note, &pl.PriceListType, &pl.CreatedAt, &pl.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, pl)
	}
	return list, rows.Err()
}

func (r repo) CreatePriceList(ctx context.Context, pl structs.PriceList) (structs.PriceList, error) {
	now := time.Now().UTC()
	err := r.db.QueryRow(ctx, `
		INSERT INTO price_lists (company_id, name, note, price_list_type, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$5)
		RETURNING id, company_id, name, COALESCE(note,''), price_list_type, created_at, updated_at`,
		pl.CompanyID, pl.Name, nullStr(pl.Note), pl.PriceListType, now,
	).Scan(&pl.ID, &pl.CompanyID, &pl.Name, &pl.Note, &pl.PriceListType, &pl.CreatedAt, &pl.UpdatedAt)
	return pl, err
}

// ─── Currency Rates ───────────────────────────────────────────────────────────

func (r repo) ListCurrencyRates(ctx context.Context, companyID int64) ([]structs.CurrencyRate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, company_id, currency, rate, rate_date, created_by, created_at
		 FROM currency_rates WHERE company_id = $1 ORDER BY rate_date DESC`,
		companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []structs.CurrencyRate
	for rows.Next() {
		cr, scanErr := scanCurrencyRate(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		list = append(list, cr)
	}
	return list, rows.Err()
}

func (r repo) CreateCurrencyRate(ctx context.Context, rate structs.CurrencyRate) (structs.CurrencyRate, error) {
	now := time.Now().UTC()
	return scanCurrencyRate(r.db.QueryRow(ctx, `
		INSERT INTO currency_rates (company_id, currency, rate, rate_date, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, company_id, currency, rate, rate_date, created_by, created_at`,
		rate.CompanyID, rate.Currency, rate.Rate, rate.RateDate, nullInt64Ptr(rate.CreatedBy), now,
	))
}

// ─── Filter helpers ───────────────────────────────────────────────────────────

func paymentDocFilter(f structs.PaymentDocumentFilter) (w []string, v []interface{}) {
	if f.CompanyID != 0 {
		w = append(w, "company_id = ?")
		v = append(v, f.CompanyID)
	}
	if f.DocumentNo != "" {
		w = append(w, "document_no = ?")
		v = append(v, strings.TrimSpace(f.DocumentNo))
	}
	if f.DocType != "" {
		w = append(w, "doc_type = ?")
		v = append(v, strings.TrimSpace(f.DocType))
	}
	if !f.From.IsZero() {
		w = append(w, "doc_date >= ?")
		v = append(v, f.From)
	}
	if !f.To.IsZero() {
		w = append(w, "doc_date <= ?")
		v = append(v, f.To)
	}
	return
}

func accountStatementFilter(f structs.AccountStatementFilter) (w []string, v []interface{}) {
	if f.CompanyID != 0 {
		w = append(w, "company_id = ?")
		v = append(v, f.CompanyID)
	}
	if f.CashRegisterID != 0 {
		w = append(w, "cash_register_id = ?")
		v = append(v, f.CashRegisterID)
	}
	if !f.From.IsZero() {
		w = append(w, "transacted_at >= ?")
		v = append(v, f.From)
	}
	if !f.To.IsZero() {
		w = append(w, "transacted_at <= ?")
		v = append(v, f.To)
	}
	return
}

func counterpartyFilter(f structs.CounterpartyBalanceFilter) (w []string, v []interface{}) {
	w = append(w, "c.company_id = ?")
	v = append(v, f.CompanyID)
	if f.GroupType != "" {
		w = append(w, "c.group_type = ?")
		v = append(v, f.GroupType)
	}
	if f.Search != "" {
		w = append(w, `c.name ILIKE ? ESCAPE '\'`)
		v = append(v, "%"+escapeLike(f.Search)+"%")
	}
	return
}

func debtRecordFilter(f structs.DebtRecordFilter) (w []string, v []interface{}) {
	if f.CompanyID != 0 {
		w = append(w, "company_id = ?")
		v = append(v, f.CompanyID)
	}
	if f.Status != "" {
		w = append(w, "status = ?")
		v = append(v, f.Status)
	}
	if f.Search != "" {
		w = append(w, `client_name ILIKE ? ESCAPE '\'`)
		v = append(v, "%"+escapeLike(f.Search)+"%")
	}
	return
}

// ─── Scan helpers ─────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPaymentDocument(s rowScanner) (structs.PaymentDocument, error) {
	var (
		d             structs.PaymentDocument
		debitAccount  sql.NullString
		creditAccount sql.NullString
		category      sql.NullString
		note          sql.NullString
		organization  sql.NullString
		createdBy     sql.NullInt64
	)
	err := s.Scan(
		&d.ID, &d.CompanyID, &d.DocumentNo, &d.DocDate, &d.DocType,
		&debitAccount, &creditAccount, &d.Income, &d.Expense,
		&category, &note, &organization, &createdBy, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return structs.PaymentDocument{}, err
	}
	d.DebitAccount = debitAccount.String
	d.CreditAccount = creditAccount.String
	d.Category = category.String
	d.Note = note.String
	d.Organization = organization.String
	if createdBy.Valid {
		d.CreatedBy = &createdBy.Int64
	}
	return d, nil
}

func scanDebtRecord(s rowScanner) (structs.DebtRecord, error) {
	var (
		d               structs.DebtRecord
		counterpartyID  sql.NullInt64
		clientName      sql.NullString
		phone           sql.NullString
		period          sql.NullString
		startDate       sql.NullTime
		nextPaymentDate sql.NullTime
		lastPaymentDate sql.NullTime
		clientText      sql.NullString
		adminText       sql.NullString
		note            sql.NullString
		channels        sql.NullString
	)
	err := s.Scan(
		&d.ID, &d.CompanyID, &counterpartyID, &clientName, &phone, &period,
		&startDate, &nextPaymentDate, &lastPaymentDate, &d.Balance,
		&clientText, &adminText, &note, &channels, &d.Status,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return structs.DebtRecord{}, err
	}
	if counterpartyID.Valid {
		d.CounterpartyID = &counterpartyID.Int64
	}
	d.ClientName = clientName.String
	d.Phone = phone.String
	d.Period = period.String
	if startDate.Valid {
		d.StartDate = &startDate.Time
	}
	if nextPaymentDate.Valid {
		d.NextPaymentDate = &nextPaymentDate.Time
	}
	if lastPaymentDate.Valid {
		d.LastPaymentDate = &lastPaymentDate.Time
	}
	d.ClientText = clientText.String
	d.AdminText = adminText.String
	d.Note = note.String
	d.Channels = channels.String
	return d, nil
}

func scanCurrencyRate(s rowScanner) (structs.CurrencyRate, error) {
	var (
		cr        structs.CurrencyRate
		createdBy sql.NullInt64
	)
	err := s.Scan(&cr.ID, &cr.CompanyID, &cr.Currency, &cr.Rate, &cr.RateDate, &createdBy, &cr.CreatedAt)
	if err != nil {
		return structs.CurrencyRate{}, err
	}
	if createdBy.Valid {
		cr.CreatedBy = &createdBy.Int64
	}
	return cr, nil
}

// ─── Column lists ─────────────────────────────────────────────────────────────

func paymentDocColumns() string {
	return `id, company_id, document_no, doc_date, doc_type, debit_account, credit_account,
	        income, expense, category, note, organization, created_by, created_at, updated_at`
}

func debtColumns() string {
	return `id, company_id, counterparty_id, client_name, phone, period, start_date,
	        next_payment_date, last_payment_date, balance, client_text, admin_text,
	        note, channels, status, created_at, updated_at`
}

// ─── Null helpers ─────────────────────────────────────────────────────────────

func nullStr(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func nullInt64Ptr(v *int64) interface{} {
	if v == nil || *v == 0 {
		return nil
	}
	return *v
}

func nullTimePtr(t *time.Time) interface{} {
	if t == nil || t.IsZero() {
		return nil
	}
	return *t
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
