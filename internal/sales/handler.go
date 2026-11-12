package sales

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	// Чеки и продажи (UC 8, UC 9, UC 10, UC 11, UC 12, UC 13)
	r.Get("/api/sales/receipts", h.ListReceipts)
	r.Post("/api/sales/checkout", h.Checkout)

	// Возврат товара по чеку (UC 14)
	r.Post("/api/sales/refund", h.Refund)

	// Сменный отчет (UC 3)
	r.Get("/api/sales/shift-report", h.ShiftReport)

	// Кассовые смены: жизненный цикл, X-отчет, Z-отчет и инкассация
	r.Get("/api/sales/shifts/current", h.GetCurrentShift)
	r.Post("/api/sales/shifts/open", h.OpenShift)
	r.Get("/api/sales/shifts/current/x-report", h.GetXReport)
	r.Post("/api/sales/shifts/close", h.CloseShift)
	r.Get("/api/sales/shifts", h.ListShifts)
}

func (h *Handler) ListReceipts(w http.ResponseWriter, r *http.Request) {
	receipts, err := h.service.GetReceipts(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, receipts, http.StatusOK)
}

func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	var req CreateSaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	receipt, err := h.service.ProcessSale(r.Context(), req)
	if err != nil {
		handleSalesError(w, err)
		return
	}

	writeJSON(w, map[string]any{
		"message": "Продажа успешно оформлена, фискальный чек распечатан",
		"receipt": receipt,
	}, http.StatusCreated)
}

func (h *Handler) Refund(w http.ResponseWriter, r *http.Request) {
	var req RefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	rec, refundSlip, err := h.service.ProcessRefund(r.Context(), req)
	if err != nil {
		handleSalesError(w, err)
		return
	}

	writeJSON(w, map[string]any{
		"message":          "Возврат прихода успешно выполнен, товары возвращены на витрину",
		"refundFiscalSlip": refundSlip,
		"receipt":          rec,
	}, http.StatusOK)
}

func (h *Handler) ShiftReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.GenerateShiftReport(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, report, http.StatusOK)
}

func (h *Handler) GetCurrentShift(w http.ResponseWriter, r *http.Request) {
	shift, err := h.service.GetActiveShift(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"hasActiveShift": shift != nil,
		"shift":          shift,
	}, http.StatusOK)
}

func (h *Handler) OpenShift(w http.ResponseWriter, r *http.Request) {
	var req OpenShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}
	shift, err := h.service.OpenShift(r.Context(), req)
	if err != nil {
		handleSalesError(w, err)
		return
	}
	writeJSON(w, map[string]any{
		"message": "Кассовая смена успешно открыта, размен внесён",
		"shift":   shift,
	}, http.StatusCreated)
}

func (h *Handler) GetXReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.GetXReport(r.Context())
	if err != nil {
		handleSalesError(w, err)
		return
	}
	writeJSON(w, report, http.StatusOK)
}

func (h *Handler) CloseShift(w http.ResponseWriter, r *http.Request) {
	var req CloseShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}
	res, err := h.service.CloseShift(r.Context(), req)
	if err != nil {
		handleSalesError(w, err)
		return
	}
	writeJSON(w, res, http.StatusOK)
}

func (h *Handler) ListShifts(w http.ResponseWriter, r *http.Request) {
	shifts, err := h.service.GetShiftsHistory(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, shifts, http.StatusOK)
}

func handleSalesError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmptyCart),
		errors.Is(err, ErrInvalidDiscount),
		errors.Is(err, ErrInsufficientCash),
		errors.Is(err, ErrInvalidPayType),
		errors.Is(err, ErrProductOutOfStock),
		errors.Is(err, ErrAlreadyRefunded),
		errors.Is(err, ErrNoActiveShift),
		errors.Is(err, ErrInvalidCashValue):
		writeJSONError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrShiftAlreadyOpen):
		writeJSONError(w, err.Error(), http.StatusConflict)
	case errors.Is(err, ErrReceiptNotFound):
		writeJSONError(w, err.Error(), http.StatusNotFound)
	default:
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, message string, code int) {
	writeJSON(w, map[string]string{"error": message}, code)
}
