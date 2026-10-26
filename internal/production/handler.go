package production

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
	// Склад сырья (UC 4, UC 5)
	r.Get("/api/ingredients", h.ListIngredients)
	r.Post("/api/ingredients/supply", h.SupplyIngredient)
	r.Post("/api/ingredients/writeoff", h.WriteOffIngredient)

	// Техкарты / Рецептуры (UC 2)
	r.Get("/api/recipes", h.ListRecipes)
	r.Post("/api/recipes", h.CreateRecipe)

	// Готовая продукция и Выпуск выпечки (UC 6 + UC 7)
	r.Get("/api/products", h.ListProducts)
	r.Get("/api/production/orders", h.ListOrders)
	r.Post("/api/production/orders", h.Produce)
}

func (h *Handler) ListIngredients(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.GetIngredients(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, items, http.StatusOK)
}

func (h *Handler) SupplyIngredient(w http.ResponseWriter, r *http.Request) {
	var req StockAdjustmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}
	ing, err := h.service.SupplyIngredient(r.Context(), req)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	writeJSON(w, ing, http.StatusOK)
}

func (h *Handler) WriteOffIngredient(w http.ResponseWriter, r *http.Request) {
	var req StockAdjustmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}
	ing, err := h.service.WriteOffIngredient(r.Context(), req)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	writeJSON(w, ing, http.StatusOK)
}

func (h *Handler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	recipes, err := h.service.GetRecipes(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, recipes, http.StatusOK)
}

func (h *Handler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}
	rec, err := h.service.CreateRecipe(r.Context(), req)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	writeJSON(w, rec, http.StatusCreated)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.GetProducts(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, products, http.StatusOK)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.GetProductionOrders(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orders, http.StatusOK)
}

func (h *Handler) Produce(w http.ResponseWriter, r *http.Request) {
	var req ProduceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}
	order, err := h.service.ProduceProduct(r.Context(), req)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	writeJSON(w, map[string]any{
		"message": "Выпуск зафиксирован. Сырьё списано по рецептуре, готовая продукция поступила на витрину",
		"order":   order,
	}, http.StatusCreated)
}

func handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidQuantity), errors.Is(err, ErrInsufficientStock):
		writeJSONError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrIngredientNotFound), errors.Is(err, ErrProductNotFound):
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
