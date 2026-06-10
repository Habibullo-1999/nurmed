package accounting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/fx"

	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger         logger.ILogger
	AccountingRepo interfaces.AccountingRepo
}

type service struct {
	logger         logger.ILogger
	accountingRepo interfaces.AccountingRepo
}

type Service interface {
	ListPaymentDocuments(ctx context.Context, filter structs.PaymentDocumentFilter) ([]structs.PaymentDocumentResponse, error)
	CreatePaymentDocument(ctx context.Context, req structs.CreatePaymentDocumentRequest) (structs.PaymentDocumentResponse, error)
	ExportPaymentDocuments(ctx context.Context, filter structs.PaymentDocumentFilter, format string) ([]byte, string, error)

	GetCashBalance(ctx context.Context, filter structs.CashRegisterBalanceFilter) ([]structs.CashRegisterBalanceResponse, error)
	ExportCashBalance(ctx context.Context, filter structs.CashRegisterBalanceFilter, format string) ([]byte, string, error)

	GetAccountStatement(ctx context.Context, filter structs.AccountStatementFilter) ([]structs.AccountStatementRow, error)
	ExportAccountStatement(ctx context.Context, filter structs.AccountStatementFilter, format string) ([]byte, string, error)

	ListCounterpartyBalances(ctx context.Context, filter structs.CounterpartyBalanceFilter) ([]structs.CounterpartyBalanceResponse, error)
	ExportCounterpartyBalances(ctx context.Context, filter structs.CounterpartyBalanceFilter, format string) ([]byte, string, error)

	ListDebtRecords(ctx context.Context, filter structs.DebtRecordFilter) ([]structs.DebtRecordResponse, error)
	CreateDebtRecord(ctx context.Context, req structs.CreateDebtRecordRequest) (structs.DebtRecordResponse, error)
	ExportDebtRecords(ctx context.Context, filter structs.DebtRecordFilter, format string) ([]byte, string, error)

	ListPriceLists(ctx context.Context, companyID int64) ([]structs.PriceListResponse, error)
	CreatePriceList(ctx context.Context, req structs.CreatePriceListRequest) (structs.PriceListResponse, error)

	ListCurrencyRates(ctx context.Context, companyID int64) ([]structs.CurrencyRateResponse, error)
	CreateCurrencyRate(ctx context.Context, req structs.CreateCurrencyRateRequest) (structs.CurrencyRateResponse, error)
}

func New(p Params) Service {
	return &service{logger: p.Logger, accountingRepo: p.AccountingRepo}
}

// ─── Payment Documents ────────────────────────────────────────────────────────

func (s *service) ListPaymentDocuments(ctx context.Context, filter structs.PaymentDocumentFilter) ([]structs.PaymentDocumentResponse, error) {
	filter.Pagination.Validate()
	docs, err := s.accountingRepo.ListPaymentDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}
	resp := make([]structs.PaymentDocumentResponse, 0, len(docs))
	for _, d := range docs {
		resp = append(resp, mapPaymentDocResponse(d))
	}
	return resp, nil
}

func (s *service) CreatePaymentDocument(ctx context.Context, req structs.CreatePaymentDocumentRequest) (structs.PaymentDocumentResponse, error) {
	req.DocType = strings.TrimSpace(req.DocType)
	if req.CompanyID <= 0 || req.DocType == "" {
		return structs.PaymentDocumentResponse{}, fmt.Errorf("invalid payment document payload")
	}

	docDate := time.Now().UTC()
	if req.DocDate != nil && !req.DocDate.IsZero() {
		docDate = req.DocDate.UTC()
	}

	var createdBy *int64
	if req.CreatedBy > 0 {
		createdBy = &req.CreatedBy
	}

	doc := structs.PaymentDocument{
		CompanyID:     req.CompanyID,
		DocumentNo:    req.DocumentNo,
		DocDate:       docDate,
		DocType:       req.DocType,
		DebitAccount:  strings.TrimSpace(req.DebitAccount),
		CreditAccount: strings.TrimSpace(req.CreditAccount),
		Income:        req.Income,
		Expense:       req.Expense,
		Category:      strings.TrimSpace(req.Category),
		Note:          strings.TrimSpace(req.Note),
		Organization:  strings.TrimSpace(req.Organization),
		CreatedBy:     createdBy,
	}

	created, err := s.accountingRepo.CreatePaymentDocument(ctx, doc)
	if err != nil {
		return structs.PaymentDocumentResponse{}, err
	}
	return mapPaymentDocResponse(created), nil
}

func (s *service) ExportPaymentDocuments(ctx context.Context, filter structs.PaymentDocumentFilter, format string) ([]byte, string, error) {
	filter.Pagination = structs.Pagination{Limit: 10000}
	filter.Pagination.Validate()
	docs, err := s.accountingRepo.ListPaymentDocuments(ctx, filter)
	if err != nil {
		return nil, "", err
	}

	headers := []string{"Дата", "Номер", "Тип документа", "Счет зачисления", "Счет списания", "Приход", "Расход", "Статья", "Примечание", "Организация"}
	rows := make([][]string, 0, len(docs))
	for _, d := range docs {
		rows = append(rows, []string{
			formatDate(d.DocDate), d.DocumentNo, d.DocType,
			d.DebitAccount, d.CreditAccount,
			formatFloat(d.Income), formatFloat(d.Expense),
			d.Category, d.Note, d.Organization,
		})
	}

	return buildOutput("Платежные документы", headers, rows, format)
}

// ─── Cash Balance ─────────────────────────────────────────────────────────────

func (s *service) GetCashBalance(ctx context.Context, filter structs.CashRegisterBalanceFilter) ([]structs.CashRegisterBalanceResponse, error) {
	return s.accountingRepo.GetCashRegisterBalances(ctx, filter)
}

func (s *service) ExportCashBalance(ctx context.Context, filter structs.CashRegisterBalanceFilter, format string) ([]byte, string, error) {
	balances, err := s.accountingRepo.GetCashRegisterBalances(ctx, filter)
	if err != nil {
		return nil, "", err
	}

	meta := "На дату: " + formatDate(filter.Date)
	headers := []string{"Касса", "Валюта", "Баланс", "Баланс в TJS"}
	rows := make([][]string, 0, len(balances))
	for _, b := range balances {
		rows = append(rows, []string{b.Name, b.Currency, formatFloat(b.Balance), formatFloat(b.BalanceInTJS)})
	}

	return buildOutput("Наличность в кассе — "+meta, headers, rows, format)
}

// ─── Account Statement ────────────────────────────────────────────────────────

func (s *service) GetAccountStatement(ctx context.Context, filter structs.AccountStatementFilter) ([]structs.AccountStatementRow, error) {
	filter.Pagination.Validate()
	txs, err := s.accountingRepo.GetAccountStatement(ctx, filter)
	if err != nil {
		return nil, err
	}

	var runningBalance float64
	rows := make([]structs.AccountStatementRow, 0, len(txs))
	for i, tx := range txs {
		opening := runningBalance
		runningBalance += tx.Income - tx.Expense
		rows = append(rows, structs.AccountStatementRow{
			ID:             int64(i + 1),
			DocDate:        tx.TransactedAt,
			DocNo:          tx.DocNo,
			DocType:        tx.DocType,
			OpeningBalance: opening,
			Income:         tx.Income,
			Expense:        tx.Expense,
			ClosingBalance: runningBalance,
			Category:       tx.Category,
			Note:           tx.Note,
			CreatedBy:      tx.CreatedBy,
		})
	}
	return rows, nil
}

func (s *service) ExportAccountStatement(ctx context.Context, filter structs.AccountStatementFilter, format string) ([]byte, string, error) {
	filter.Pagination = structs.Pagination{Limit: 10000}
	filter.Pagination.Validate()
	rows, err := s.GetAccountStatement(ctx, filter)
	if err != nil {
		return nil, "", err
	}

	headers := []string{"№", "Дата", "Номер", "Документ", "На начало", "Приход", "Расход", "На конец", "Статья", "Примечание"}
	data := make([][]string, 0, len(rows))
	for i, r := range rows {
		data = append(data, []string{
			fmt.Sprintf("%d", i+1), formatDate(r.DocDate), r.DocNo, r.DocType,
			formatFloat(r.OpeningBalance), formatFloat(r.Income), formatFloat(r.Expense),
			formatFloat(r.ClosingBalance), r.Category, r.Note,
		})
	}

	return buildOutput("Выписка по счёту", headers, data, format)
}

// ─── Counterparty Balance ─────────────────────────────────────────────────────

func (s *service) ListCounterpartyBalances(ctx context.Context, filter structs.CounterpartyBalanceFilter) ([]structs.CounterpartyBalanceResponse, error) {
	filter.Pagination.Validate()
	return s.accountingRepo.ListCounterpartyBalances(ctx, filter)
}

func (s *service) ExportCounterpartyBalances(ctx context.Context, filter structs.CounterpartyBalanceFilter, format string) ([]byte, string, error) {
	filter.Pagination = structs.Pagination{Limit: 10000}
	filter.Pagination.Validate()
	balances, err := s.accountingRepo.ListCounterpartyBalances(ctx, filter)
	if err != nil {
		return nil, "", err
	}

	headers := []string{"№", "Группа", "Контрагент", "Телефон", "Регион", "Валюта", "Сумма", "Сумма (TJS)"}
	rows := make([][]string, 0, len(balances))
	for _, b := range balances {
		rows = append(rows, []string{
			fmt.Sprintf("%d", b.No), b.GroupType, b.Name, b.Phone, b.Region,
			b.Currency, formatFloat(b.Amount), formatFloat(b.AmountInTJS),
		})
	}

	return buildOutput("Баланс контрагентов", headers, rows, format)
}

// ─── Debt Records ─────────────────────────────────────────────────────────────

func (s *service) ListDebtRecords(ctx context.Context, filter structs.DebtRecordFilter) ([]structs.DebtRecordResponse, error) {
	filter.Pagination.Validate()
	records, err := s.accountingRepo.ListDebtRecords(ctx, filter)
	if err != nil {
		return nil, err
	}
	resp := make([]structs.DebtRecordResponse, 0, len(records))
	for _, r := range records {
		resp = append(resp, mapDebtRecordResponse(r))
	}
	return resp, nil
}

func (s *service) CreateDebtRecord(ctx context.Context, req structs.CreateDebtRecordRequest) (structs.DebtRecordResponse, error) {
	if req.CompanyID <= 0 || req.Balance <= 0 {
		return structs.DebtRecordResponse{}, fmt.Errorf("invalid debt record payload")
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}

	record := structs.DebtRecord{
		CompanyID:       req.CompanyID,
		CounterpartyID:  req.CounterpartyID,
		ClientName:      strings.TrimSpace(req.ClientName),
		Phone:           strings.TrimSpace(req.Phone),
		Period:          strings.TrimSpace(req.Period),
		StartDate:       req.StartDate,
		NextPaymentDate: req.NextPaymentDate,
		Balance:         req.Balance,
		ClientText:      strings.TrimSpace(req.ClientText),
		AdminText:       strings.TrimSpace(req.AdminText),
		Note:            strings.TrimSpace(req.Note),
		Channels:        strings.TrimSpace(req.Channels),
		Status:          status,
	}

	created, err := s.accountingRepo.CreateDebtRecord(ctx, record)
	if err != nil {
		return structs.DebtRecordResponse{}, err
	}
	return mapDebtRecordResponse(created), nil
}

func (s *service) ExportDebtRecords(ctx context.Context, filter structs.DebtRecordFilter, format string) ([]byte, string, error) {
	filter.Pagination = structs.Pagination{Limit: 10000}
	filter.Pagination.Validate()
	records, err := s.accountingRepo.ListDebtRecords(ctx, filter)
	if err != nil {
		return nil, "", err
	}

	headers := []string{"Клиент", "Телефон", "Период", "Дата", "След. платёж", "Пос. платёж", "Баланс", "Текст клиенту", "Текст администратору", "Примечание", "Каналы", "Статус"}
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		rows = append(rows, []string{
			r.ClientName, r.Phone, r.Period,
			formatTimePtr(r.StartDate), formatTimePtr(r.NextPaymentDate), formatTimePtr(r.LastPaymentDate),
			formatFloat(r.Balance), r.ClientText, r.AdminText, r.Note, r.Channels, r.Status,
		})
	}

	return buildOutput("Управление долгами", headers, rows, format)
}

// ─── Price Lists ──────────────────────────────────────────────────────────────

func (s *service) ListPriceLists(ctx context.Context, companyID int64) ([]structs.PriceListResponse, error) {
	pls, err := s.accountingRepo.ListPriceLists(ctx, companyID)
	if err != nil {
		return nil, err
	}
	resp := make([]structs.PriceListResponse, 0, len(pls))
	for i, pl := range pls {
		resp = append(resp, structs.PriceListResponse{
			ID:            pl.ID,
			CompanyID:     pl.CompanyID,
			No:            i + 1,
			Name:          pl.Name,
			Note:          pl.Note,
			PriceListType: pl.PriceListType,
			CreatedAt:     pl.CreatedAt,
			UpdatedAt:     pl.UpdatedAt,
		})
	}
	return resp, nil
}

func (s *service) CreatePriceList(ctx context.Context, req structs.CreatePriceListRequest) (structs.PriceListResponse, error) {
	if req.CompanyID <= 0 || strings.TrimSpace(req.Name) == "" {
		return structs.PriceListResponse{}, fmt.Errorf("invalid price list payload")
	}
	plType := strings.TrimSpace(req.PriceListType)
	if plType == "" {
		plType = "nomenclature"
	}

	created, err := s.accountingRepo.CreatePriceList(ctx, structs.PriceList{
		CompanyID:     req.CompanyID,
		Name:          strings.TrimSpace(req.Name),
		Note:          strings.TrimSpace(req.Note),
		PriceListType: plType,
	})
	if err != nil {
		return structs.PriceListResponse{}, err
	}
	return structs.PriceListResponse{
		ID:            created.ID,
		CompanyID:     created.CompanyID,
		No:            1,
		Name:          created.Name,
		Note:          created.Note,
		PriceListType: created.PriceListType,
		CreatedAt:     created.CreatedAt,
		UpdatedAt:     created.UpdatedAt,
	}, nil
}

// ─── Currency Rates ───────────────────────────────────────────────────────────

func (s *service) ListCurrencyRates(ctx context.Context, companyID int64) ([]structs.CurrencyRateResponse, error) {
	rates, err := s.accountingRepo.ListCurrencyRates(ctx, companyID)
	if err != nil {
		return nil, err
	}
	resp := make([]structs.CurrencyRateResponse, 0, len(rates))
	for _, r := range rates {
		resp = append(resp, mapCurrencyRateResponse(r))
	}
	return resp, nil
}

func (s *service) CreateCurrencyRate(ctx context.Context, req structs.CreateCurrencyRateRequest) (structs.CurrencyRateResponse, error) {
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.CompanyID <= 0 || req.Currency == "" || req.Rate <= 0 {
		return structs.CurrencyRateResponse{}, fmt.Errorf("invalid currency rate payload")
	}

	rateDate := time.Now().UTC()
	if req.RateDate != nil && !req.RateDate.IsZero() {
		rateDate = req.RateDate.UTC()
	}

	var createdBy *int64
	if req.CreatedBy > 0 {
		createdBy = &req.CreatedBy
	}

	created, err := s.accountingRepo.CreateCurrencyRate(ctx, structs.CurrencyRate{
		CompanyID: req.CompanyID,
		Currency:  req.Currency,
		Rate:      req.Rate,
		RateDate:  rateDate,
		CreatedBy: createdBy,
	})
	if err != nil {
		return structs.CurrencyRateResponse{}, err
	}
	return mapCurrencyRateResponse(created), nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func buildOutput(title string, headers []string, rows [][]string, format string) ([]byte, string, error) {
	switch format {
	case structs.ExportFormatExcel:
		data, err := buildExcel(title, headers, rows)
		return data, contentType(format), err
	case structs.ExportFormatHTML:
		data, err := buildHTML(exportData{
			Title:   title,
			Meta:    "Экспортировано: " + formatDateTime(time.Now()),
			Headers: headers,
			Rows:    rows,
		})
		return data, contentType(format), err
	default: // tsv
		return buildTSV(headers, rows), contentType(format), nil
	}
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatDate(*t)
}

func mapPaymentDocResponse(d structs.PaymentDocument) structs.PaymentDocumentResponse {
	return structs.PaymentDocumentResponse{
		ID:            d.ID,
		CompanyID:     d.CompanyID,
		DocumentNo:    d.DocumentNo,
		DocDate:       d.DocDate,
		DocType:       d.DocType,
		DebitAccount:  d.DebitAccount,
		CreditAccount: d.CreditAccount,
		Income:        d.Income,
		Expense:       d.Expense,
		Category:      d.Category,
		Note:          d.Note,
		Organization:  d.Organization,
		CreatedBy:     d.CreatedBy,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

func mapDebtRecordResponse(r structs.DebtRecord) structs.DebtRecordResponse {
	return structs.DebtRecordResponse{
		ID:              r.ID,
		CompanyID:       r.CompanyID,
		CounterpartyID:  r.CounterpartyID,
		ClientName:      r.ClientName,
		Phone:           r.Phone,
		Period:          r.Period,
		StartDate:       r.StartDate,
		NextPaymentDate: r.NextPaymentDate,
		LastPaymentDate: r.LastPaymentDate,
		Balance:         r.Balance,
		ClientText:      r.ClientText,
		AdminText:       r.AdminText,
		Note:            r.Note,
		Channels:        r.Channels,
		Status:          r.Status,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

func mapCurrencyRateResponse(r structs.CurrencyRate) structs.CurrencyRateResponse {
	return structs.CurrencyRateResponse{
		ID:        r.ID,
		CompanyID: r.CompanyID,
		Currency:  r.Currency,
		Rate:      r.Rate,
		RateDate:  r.RateDate,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
	}
}
