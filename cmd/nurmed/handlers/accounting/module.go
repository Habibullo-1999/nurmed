package accounting

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"

	restmiddleware "nurmed/cmd/nurmed/handlers/middleware"
	"nurmed/internal/accounting"
	"nurmed/internal/responses"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger        logger.ILogger
	AccountingSvs accounting.Service
}

type handler struct {
	logger        logger.ILogger
	accountingSvs accounting.Service
}

type Handler interface {
	GetPaymentDocuments(c *gin.Context)
	CreatePaymentDocument(c *gin.Context)
	ExportPaymentDocuments(c *gin.Context)

	GetCashBalance(c *gin.Context)
	ExportCashBalance(c *gin.Context)

	GetAccountStatement(c *gin.Context)
	ExportAccountStatement(c *gin.Context)

	GetCounterpartyBalances(c *gin.Context)
	ExportCounterpartyBalances(c *gin.Context)

	GetDebtRecords(c *gin.Context)
	CreateDebtRecord(c *gin.Context)
	ExportDebtRecords(c *gin.Context)

	GetPriceLists(c *gin.Context)
	CreatePriceList(c *gin.Context)

	GetCurrencyRates(c *gin.Context)
	CreateCurrencyRate(c *gin.Context)
}

func New(p Params) Handler {
	return &handler{logger: p.Logger, accountingSvs: p.AccountingSvs}
}

// ─── Payment Documents ────────────────────────────────────────────────────────

func (h *handler) GetPaymentDocuments(c *gin.Context) {
	var filter structs.PaymentDocumentFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "accounting: ShouldBindQuery payment documents", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}

	docs, err := h.accountingSvs.ListPaymentDocuments(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: ListPaymentDocuments", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = docs
	c.JSON(resp.Code, &resp)
}

func (h *handler) CreatePaymentDocument(c *gin.Context) {
	var req structs.CreatePaymentDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warning(c.Request.Context(), "accounting: ShouldBindJSON payment document", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if req.CompanyID == 0 && scope.CompanyID > 0 {
		req.CompanyID = scope.CompanyID
	}
	if principal, ok := restmiddleware.GetPrincipal(c); ok {
		req.CreatedBy = principal.UserID
	}

	doc, err := h.accountingSvs.CreatePaymentDocument(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: CreatePaymentDocument", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = doc
	c.JSON(resp.Code, &resp)
}

func (h *handler) ExportPaymentDocuments(c *gin.Context) {
	var filter structs.PaymentDocumentFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	format := c.DefaultQuery("format", structs.ExportFormatExcel)
	h.sendExport(c, "payment-documents", format, func() ([]byte, string, error) {
		return h.accountingSvs.ExportPaymentDocuments(c.Request.Context(), filter, format)
	})
}

// ─── Cash Balance ─────────────────────────────────────────────────────────────

func (h *handler) GetCashBalance(c *gin.Context) {
	var filter structs.CashRegisterBalanceFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "accounting: ShouldBindQuery cash balance", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	if filter.Date.IsZero() {
		filter.Date = time.Now()
	}

	balances, err := h.accountingSvs.GetCashBalance(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: GetCashBalance", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = balances
	c.JSON(resp.Code, &resp)
}

func (h *handler) ExportCashBalance(c *gin.Context) {
	var filter structs.CashRegisterBalanceFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	if filter.Date.IsZero() {
		filter.Date = time.Now()
	}
	format := c.DefaultQuery("format", structs.ExportFormatExcel)
	h.sendExport(c, "cash-balance", format, func() ([]byte, string, error) {
		return h.accountingSvs.ExportCashBalance(c.Request.Context(), filter, format)
	})
}

// ─── Account Statement ────────────────────────────────────────────────────────

func (h *handler) GetAccountStatement(c *gin.Context) {
	var filter structs.AccountStatementFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "accounting: ShouldBindQuery account statement", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}

	rows, err := h.accountingSvs.GetAccountStatement(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: GetAccountStatement", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = rows
	c.JSON(resp.Code, &resp)
}

func (h *handler) ExportAccountStatement(c *gin.Context) {
	var filter structs.AccountStatementFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	format := c.DefaultQuery("format", structs.ExportFormatExcel)
	h.sendExport(c, "account-statement", format, func() ([]byte, string, error) {
		return h.accountingSvs.ExportAccountStatement(c.Request.Context(), filter, format)
	})
}

// ─── Counterparty Balance ─────────────────────────────────────────────────────

func (h *handler) GetCounterpartyBalances(c *gin.Context) {
	var filter structs.CounterpartyBalanceFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "accounting: ShouldBindQuery counterparty balance", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}

	balances, err := h.accountingSvs.ListCounterpartyBalances(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: ListCounterpartyBalances", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = balances
	c.JSON(resp.Code, &resp)
}

func (h *handler) ExportCounterpartyBalances(c *gin.Context) {
	var filter structs.CounterpartyBalanceFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	format := c.DefaultQuery("format", structs.ExportFormatExcel)
	h.sendExport(c, "counterparty-balance", format, func() ([]byte, string, error) {
		return h.accountingSvs.ExportCounterpartyBalances(c.Request.Context(), filter, format)
	})
}

// ─── Debt Records ─────────────────────────────────────────────────────────────

func (h *handler) GetDebtRecords(c *gin.Context) {
	var filter structs.DebtRecordFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "accounting: ShouldBindQuery debt records", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}

	records, err := h.accountingSvs.ListDebtRecords(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: ListDebtRecords", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = records
	c.JSON(resp.Code, &resp)
}

func (h *handler) CreateDebtRecord(c *gin.Context) {
	var req structs.CreateDebtRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warning(c.Request.Context(), "accounting: ShouldBindJSON debt record", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if req.CompanyID == 0 && scope.CompanyID > 0 {
		req.CompanyID = scope.CompanyID
	}

	record, err := h.accountingSvs.CreateDebtRecord(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: CreateDebtRecord", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = record
	c.JSON(resp.Code, &resp)
}

func (h *handler) ExportDebtRecords(c *gin.Context) {
	var filter structs.DebtRecordFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	format := c.DefaultQuery("format", structs.ExportFormatExcel)
	h.sendExport(c, "debt-records", format, func() ([]byte, string, error) {
		return h.accountingSvs.ExportDebtRecords(c.Request.Context(), filter, format)
	})
}

// ─── Price Lists ──────────────────────────────────────────────────────────────

func (h *handler) GetPriceLists(c *gin.Context) {
	scope := restmiddleware.GetRequestScope(c)
	companyID := scope.CompanyID
	if idStr := c.Query("company_id"); idStr != "" {
		fmt.Sscanf(idStr, "%d", &companyID)
	}
	if companyID == 0 {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}

	pls, err := h.accountingSvs.ListPriceLists(c.Request.Context(), companyID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: ListPriceLists", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = pls
	c.JSON(resp.Code, &resp)
}

func (h *handler) CreatePriceList(c *gin.Context) {
	var req structs.CreatePriceListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warning(c.Request.Context(), "accounting: ShouldBindJSON price list", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if req.CompanyID == 0 && scope.CompanyID > 0 {
		req.CompanyID = scope.CompanyID
	}

	pl, err := h.accountingSvs.CreatePriceList(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: CreatePriceList", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = pl
	c.JSON(resp.Code, &resp)
}

// ─── Currency Rates ───────────────────────────────────────────────────────────

func (h *handler) GetCurrencyRates(c *gin.Context) {
	scope := restmiddleware.GetRequestScope(c)
	companyID := scope.CompanyID
	if idStr := c.Query("company_id"); idStr != "" {
		fmt.Sscanf(idStr, "%d", &companyID)
	}
	if companyID == 0 {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}

	rates, err := h.accountingSvs.ListCurrencyRates(c.Request.Context(), companyID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: ListCurrencyRates", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = rates
	c.JSON(resp.Code, &resp)
}

func (h *handler) CreateCurrencyRate(c *gin.Context) {
	var req structs.CreateCurrencyRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warning(c.Request.Context(), "accounting: ShouldBindJSON currency rate", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if req.CompanyID == 0 && scope.CompanyID > 0 {
		req.CompanyID = scope.CompanyID
	}
	if principal, ok := restmiddleware.GetPrincipal(c); ok {
		req.CreatedBy = principal.UserID
	}

	rate, err := h.accountingSvs.CreateCurrencyRate(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: CreateCurrencyRate", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = rate
	c.JSON(resp.Code, &resp)
}

// ─── Export helper ────────────────────────────────────────────────────────────

func (h *handler) sendExport(c *gin.Context, name string, format string, fn func() ([]byte, string, error)) {
	data, ct, err := fn()
	if err != nil {
		h.logger.Error(c.Request.Context(), "accounting: export error", zap.String("name", name), zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}

	ext := "xlsx"
	switch format {
	case structs.ExportFormatHTML:
		ext = "html"
	case structs.ExportFormatTSV:
		ext = "tsv"
	}

	filename := fmt.Sprintf("%s-%s.%s", name, time.Now().Format("2006-01-02"), ext)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, ct, data)
}
