package interfaces

import (
	"context"

	"nurmed/internal/structs"
)

type AccountingRepo interface {
	ListPaymentDocuments(ctx context.Context, filter structs.PaymentDocumentFilter) ([]structs.PaymentDocument, error)
	CreatePaymentDocument(ctx context.Context, doc structs.PaymentDocument) (structs.PaymentDocument, error)

	ListCashRegisters(ctx context.Context, companyID int64) ([]structs.CashRegister, error)
	GetCashRegisterBalances(ctx context.Context, filter structs.CashRegisterBalanceFilter) ([]structs.CashRegisterBalanceResponse, error)
	GetAccountStatement(ctx context.Context, filter structs.AccountStatementFilter) ([]structs.CashTransaction, error)

	ListCounterpartyBalances(ctx context.Context, filter structs.CounterpartyBalanceFilter) ([]structs.CounterpartyBalanceResponse, error)

	ListDebtRecords(ctx context.Context, filter structs.DebtRecordFilter) ([]structs.DebtRecord, error)
	CreateDebtRecord(ctx context.Context, record structs.DebtRecord) (structs.DebtRecord, error)

	ListPriceLists(ctx context.Context, companyID int64) ([]structs.PriceList, error)
	CreatePriceList(ctx context.Context, pl structs.PriceList) (structs.PriceList, error)

	ListCurrencyRates(ctx context.Context, companyID int64) ([]structs.CurrencyRate, error)
	CreateCurrencyRate(ctx context.Context, rate structs.CurrencyRate) (structs.CurrencyRate, error)
}
