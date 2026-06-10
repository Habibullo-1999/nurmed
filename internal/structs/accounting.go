package structs

import "time"

type ExportFormat = string

const (
	ExportFormatExcel ExportFormat = "excel"
	ExportFormatHTML  ExportFormat = "html"
	ExportFormatTSV   ExportFormat = "tsv"
)

// ─── Cash Registers ──────────────────────────────────────────────────────────

type CashRegister struct {
	ID        int64
	CompanyID int64
	Name      string
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CashRegisterBalanceFilter struct {
	CompanyID int64     `form:"company_id"`
	Date      time.Time `form:"date" time_format:"2006-01-02"`
}

type CashRegisterBalanceResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Currency    string  `json:"currency"`
	Balance     float64 `json:"balance"`
	BalanceInTJS float64 `json:"balanceInTJS"`
}

// ─── Cash Transactions / Account Statement ───────────────────────────────────

type CashTransaction struct {
	ID             int64
	CompanyID      int64
	CashRegisterID int64
	DocType        string
	DocNo          string
	Income         float64
	Expense        float64
	Category       string
	Note           string
	CreatedBy      *int64
	TransactedAt   time.Time
	CreatedAt      time.Time
}

type AccountStatementFilter struct {
	CompanyID      int64     `form:"company_id"`
	CashRegisterID int64     `form:"cash_register_id"`
	From           time.Time `form:"from" time_format:"2006-01-02"`
	To             time.Time `form:"to" time_format:"2006-01-02"`
	Pagination
}

type AccountStatementRow struct {
	ID             int64     `json:"id"`
	DocDate        time.Time `json:"docDate"`
	DocNo          string    `json:"docNo"`
	DocType        string    `json:"docType"`
	OpeningBalance float64   `json:"openingBalance"`
	Income         float64   `json:"income"`
	Expense        float64   `json:"expense"`
	ClosingBalance float64   `json:"closingBalance"`
	Category       string    `json:"category"`
	Note           string    `json:"note"`
	CreatedBy      *int64    `json:"createdBy,omitempty"`
}

// ─── Payment Documents ────────────────────────────────────────────────────────

type PaymentDocument struct {
	ID            int64
	CompanyID     int64
	DocumentNo    string
	DocDate       time.Time
	DocType       string
	DebitAccount  string
	CreditAccount string
	Income        float64
	Expense       float64
	Category      string
	Note          string
	Organization  string
	CreatedBy     *int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PaymentDocumentFilter struct {
	CompanyID  int64     `form:"company_id"`
	DocumentNo string    `form:"document_no"`
	DocType    string    `form:"doc_type"`
	From       time.Time `form:"from" time_format:"2006-01-02"`
	To         time.Time `form:"to" time_format:"2006-01-02"`
	Pagination
}

type CreatePaymentDocumentRequest struct {
	CompanyID     int64      `json:"companyId"`
	DocumentNo    string     `json:"documentNo"`
	DocDate       *time.Time `json:"docDate"`
	DocType       string     `json:"docType" binding:"required"`
	DebitAccount  string     `json:"debitAccount"`
	CreditAccount string     `json:"creditAccount"`
	Income        float64    `json:"income"`
	Expense       float64    `json:"expense"`
	Category      string     `json:"category"`
	Note          string     `json:"note"`
	Organization  string     `json:"organization"`
	CreatedBy     int64      `json:"-"`
}

type PaymentDocumentResponse struct {
	ID            int64     `json:"id"`
	CompanyID     int64     `json:"companyId"`
	DocumentNo    string    `json:"documentNo"`
	DocDate       time.Time `json:"docDate"`
	DocType       string    `json:"docType"`
	DebitAccount  string    `json:"debitAccount,omitempty"`
	CreditAccount string    `json:"creditAccount,omitempty"`
	Income        float64   `json:"income"`
	Expense       float64   `json:"expense"`
	Category      string    `json:"category,omitempty"`
	Note          string    `json:"note,omitempty"`
	Organization  string    `json:"organization,omitempty"`
	CreatedBy     *int64    `json:"createdBy,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ─── Counterparties / Counterparty Balance ────────────────────────────────────

type Counterparty struct {
	ID        int64
	CompanyID int64
	Name      string
	GroupType string
	Phone     string
	Region    string
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CounterpartyBalanceFilter struct {
	CompanyID int64  `form:"company_id"`
	GroupType string `form:"group_type"` // "debtor" | "creditor" | ""
	Search    string `form:"search"`
	Pagination
}

type CounterpartyBalanceResponse struct {
	ID          int64   `json:"id"`
	No          int     `json:"no"`
	GroupType   string  `json:"groupType"`
	Name        string  `json:"name"`
	Phone       string  `json:"phone,omitempty"`
	Region      string  `json:"region,omitempty"`
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount"`
	AmountInTJS float64 `json:"amountInTJS"`
}

// ─── Debt Records ─────────────────────────────────────────────────────────────

type DebtRecord struct {
	ID              int64
	CompanyID       int64
	CounterpartyID  *int64
	ClientName      string
	Phone           string
	Period          string
	StartDate       *time.Time
	NextPaymentDate *time.Time
	LastPaymentDate *time.Time
	Balance         float64
	ClientText      string
	AdminText       string
	Note            string
	Channels        string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type DebtRecordFilter struct {
	CompanyID int64  `form:"company_id"`
	Status    string `form:"status"`
	Search    string `form:"search"`
	Pagination
}

type CreateDebtRecordRequest struct {
	CompanyID       int64      `json:"companyId"`
	CounterpartyID  *int64     `json:"counterpartyId,omitempty"`
	ClientName      string     `json:"clientName"`
	Phone           string     `json:"phone"`
	Period          string     `json:"period"`
	StartDate       *time.Time `json:"startDate"`
	NextPaymentDate *time.Time `json:"nextPaymentDate"`
	Balance         float64    `json:"balance" binding:"required"`
	ClientText      string     `json:"clientText"`
	AdminText       string     `json:"adminText"`
	Note            string     `json:"note"`
	Channels        string     `json:"channels"`
	Status          string     `json:"status"`
}

type DebtRecordResponse struct {
	ID              int64      `json:"id"`
	CompanyID       int64      `json:"companyId"`
	CounterpartyID  *int64     `json:"counterpartyId,omitempty"`
	ClientName      string     `json:"clientName"`
	Phone           string     `json:"phone,omitempty"`
	Period          string     `json:"period,omitempty"`
	StartDate       *time.Time `json:"startDate,omitempty"`
	NextPaymentDate *time.Time `json:"nextPaymentDate,omitempty"`
	LastPaymentDate *time.Time `json:"lastPaymentDate,omitempty"`
	Balance         float64    `json:"balance"`
	ClientText      string     `json:"clientText,omitempty"`
	AdminText       string     `json:"adminText,omitempty"`
	Note            string     `json:"note,omitempty"`
	Channels        string     `json:"channels,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// ─── Price Lists ──────────────────────────────────────────────────────────────

type PriceList struct {
	ID            int64
	CompanyID     int64
	Name          string
	Note          string
	PriceListType string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreatePriceListRequest struct {
	CompanyID     int64  `json:"companyId"`
	Name          string `json:"name" binding:"required"`
	Note          string `json:"note"`
	PriceListType string `json:"priceListType"`
}

type PriceListResponse struct {
	ID            int64     `json:"id"`
	CompanyID     int64     `json:"companyId"`
	No            int       `json:"no"`
	Name          string    `json:"name"`
	Note          string    `json:"note,omitempty"`
	PriceListType string    `json:"priceListType"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ─── Currency Rates ───────────────────────────────────────────────────────────

type CurrencyRate struct {
	ID        int64
	CompanyID int64
	Currency  string
	Rate      float64
	RateDate  time.Time
	CreatedBy *int64
	CreatedAt time.Time
}

type CreateCurrencyRateRequest struct {
	CompanyID int64      `json:"companyId"`
	Currency  string     `json:"currency" binding:"required"`
	Rate      float64    `json:"rate" binding:"required"`
	RateDate  *time.Time `json:"rateDate"`
	CreatedBy int64      `json:"-"`
}

type CurrencyRateResponse struct {
	ID        int64     `json:"id"`
	CompanyID int64     `json:"companyId"`
	Currency  string    `json:"currency"`
	Rate      float64   `json:"rate"`
	RateDate  time.Time `json:"rateDate"`
	CreatedBy *int64    `json:"createdBy,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
