package sales

import (
	"math"
	"time"
)

// ReceiptItem — товарная позиция в чеке (композиция от Receipt)
type ReceiptItem struct {
	ReceiptItemID int     `json:"receiptItemId"`
	ReceiptID     int     `json:"receiptId"`
	ProductID     int     `json:"productId"`
	ProductName   string  `json:"productName,omitempty"`
	Quantity      int     `json:"quantity"`
	UnitPrice     float64 `json:"unitPrice"`
	LineTotal     float64 `json:"lineTotal"`
}

func (ri *ReceiptItem) CalculateLineTotal() float64 {
	ri.LineTotal = math.Round(float64(ri.Quantity)*ri.UnitPrice*100) / 100
	return ri.LineTotal
}

// Receipt — фискальный чек продажи
type Receipt struct {
	ReceiptID       int            `json:"receiptId"`
	CashierID       int            `json:"cashierId"`
	CashierName     string         `json:"cashierName,omitempty"`
	ReceiptDate     time.Time      `json:"receiptDate"`
	DiscountRate    float64        `json:"discountRate"`
	IsCompleted     bool           `json:"isCompleted"`
	IsRefunded      bool           `json:"isRefunded"`
	FiscalDocNumber string         `json:"fiscalDocNumber"`
	Items           []ReceiptItem  `json:"items"`
	GrandTotal      float64        `json:"grandTotal"`
	PaymentDetails  *PaymentRecord `json:"payment,omitempty"`
}

func (r *Receipt) AddPosition(productID int, name string, price float64, quantity int) {
	item := ReceiptItem{
		ProductID:   productID,
		ProductName: name,
		Quantity:    quantity,
		UnitPrice:   price,
	}
	item.CalculateLineTotal()
	r.Items = append(r.Items, item)
}

func (r *Receipt) CalculateGrandTotal(appliedDiscount float64) float64 {
	var subtotal float64
	for i := range r.Items {
		subtotal += r.Items[i].CalculateLineTotal()
	}
	r.DiscountRate = appliedDiscount
	discountFactor := 1.0 - (appliedDiscount / 100.0)
	r.GrandTotal = math.Round(subtotal*discountFactor*100) / 100
	return r.GrandTotal
}

func (r *Receipt) RegisterFiscalSlip(docNumber string) bool {
	if docNumber == "" {
		return false
	}
	r.FiscalDocNumber = docNumber
	r.IsCompleted = true
	return true
}

// PaymentProcessor — интерфейс абстрактного класса Payment (UC 11)
type PaymentProcessor interface {
	ExecuteTransaction(targetAmount float64) bool
}

// BasePayment — поля абстрактного класса Payment
type BasePayment struct {
	PaymentID   int       `json:"paymentId"`
	Amount      float64   `json:"amount"`
	PaymentDate time.Time `json:"paymentDate"`
}

// CashPayment — наличный расчет (UC 12, наследует Payment)
type CashPayment struct {
	BasePayment
	TenderedAmount float64 `json:"tenderedAmount"`
	ChangeAmount   float64 `json:"changeAmount"`
}

func (cp *CashPayment) CalculateChange(cashReceived float64) float64 {
	cp.TenderedAmount = cashReceived
	cp.ChangeAmount = math.Round((cashReceived-cp.Amount)*100) / 100
	return cp.ChangeAmount
}

func (cp *CashPayment) ExecuteTransaction(targetAmount float64) bool {
	cp.Amount = targetAmount
	if cp.TenderedAmount < targetAmount {
		return false
	}
	cp.CalculateChange(cp.TenderedAmount)
	cp.PaymentDate = time.Now()
	return true
}

// CardPayment — безналичный расчет по карте (UC 13, наследует Payment)
type CardPayment struct {
	BasePayment
	CardPanMasked     string `json:"cardPanMasked"`
	AuthorizationCode string `json:"authorizationCode"`
}

func (cp *CardPayment) VerifyCardAcquiring(authCode string) bool {
	if authCode == "" {
		return false
	}
	cp.AuthorizationCode = authCode
	return true
}

func (cp *CardPayment) ExecuteTransaction(targetAmount float64) bool {
	cp.Amount = targetAmount
	cp.PaymentDate = time.Now()
	return cp.VerifyCardAcquiring(cp.AuthorizationCode)
}

// PaymentRecord — универсальная структура платежа для чтения/записи в БД
type PaymentRecord struct {
	PaymentID         int       `json:"paymentId"`
	ReceiptID         int       `json:"receiptId"`
	PaymentType       string    `json:"paymentType"` // CASH | CARD
	Amount            float64   `json:"amount"`
	PaymentDate       time.Time `json:"paymentDate"`
	TenderedAmount    *float64  `json:"tenderedAmount,omitempty"`
	ChangeAmount      *float64  `json:"changeAmount,omitempty"`
	CardPanMasked     *string   `json:"cardPanMasked,omitempty"`
	AuthorizationCode *string   `json:"authorizationCode,omitempty"`
}

// ShiftReportDTO — модель представления сменного отчета (UC 3)
type ShiftReportDTO struct {
	ShiftID       int       `json:"shiftId"`
	OpenedAt      time.Time `json:"openedAt"`
	TotalRevenue  float64   `json:"totalRevenue"`
	CashSubtotal  float64   `json:"cashSubtotal"`
	CardSubtotal  float64   `json:"cardSubtotal"`
	ReceiptCount  int       `json:"receiptCount"`
	RefundedCount int       `json:"refundedCount"`
}

func (s *ShiftReportDTO) ExportReportToPdf(outputPath string) bool {
	return outputPath != ""
}

// DTO для HTTP-запросов оформления продажи и возврата
type CreateSaleRequest struct {
	CashierID    int     `json:"cashierId"`
	DiscountRate float64 `json:"discountRate"` // UC 9: Применить скидку (0..100%)
	PaymentType  string  `json:"paymentType"`  // "CASH" или "CARD"
	TenderedCash float64 `json:"tenderedCash,omitempty"`
	CardPan      string  `json:"cardPan,omitempty"`
	Items        []struct {
		ProductID int `json:"productId"`
		Quantity  int `json:"quantity"`
	} `json:"items"`
}

type RefundRequest struct {
	FiscalDocNumber string `json:"fiscalDocNumber"`
	Reason          string `json:"reason"`
}
