package production

import "time"

const ExpirationThresholdDays = 5

// Ingredient — сырьё на складе (мука, масло, сахар и т.д.)
type Ingredient struct {
	IngredientID  int     `json:"ingredientId"`
	Name          string  `json:"name"`
	CurrentStock  float64 `json:"currentStock"`
	Unit          string  `json:"unit"`
	PurchasePrice float64 `json:"purchasePrice"`
}

func (i *Ingredient) AddStock(amount float64) {
	i.CurrentStock += amount
}

func (i *Ingredient) IsStockSufficient(requiredAmount float64) bool {
	return i.CurrentStock >= requiredAmount
}

func (i *Ingredient) DeductStock(amount float64) bool {
	if !i.IsStockSufficient(amount) {
		return false
	}
	i.CurrentStock -= amount
	return true
}

// Product — готовое хлебобулочное изделие на витрине
type Product struct {
	ProductID       int     `json:"productId"`
	RecipeID        *int    `json:"recipeId,omitempty"`
	Name            string  `json:"name"`
	Barcode         string  `json:"barcode"`
	RetailPrice     float64 `json:"retailPrice"`
	CurrentQuantity int     `json:"currentQuantity"`
}

func (p *Product) UpdatePrice(newPrice float64) {
	p.RetailPrice = newPrice
}

func (p *Product) HasSufficientQuantity(quantity int) bool {
	return p.CurrentQuantity >= quantity
}

// RecipeItem — строка технологической карты (ингредиент и норма расхода)
type RecipeItem struct {
	RecipeItemID     int     `json:"recipeItemId"`
	RecipeID         int     `json:"recipeId"`
	IngredientID     int     `json:"ingredientId"`
	IngredientName   string  `json:"ingredientName,omitempty"`
	Unit             string  `json:"unit,omitempty"`
	PurchasePrice    float64 `json:"purchasePrice,omitempty"`
	RequiredQuantity float64 `json:"requiredQuantity"`
}

func (ri *RecipeItem) CalculateSubCost() float64 {
	return ri.RequiredQuantity * ri.PurchasePrice
}

// Recipe — технологическая карта (рецептура)
type Recipe struct {
	RecipeID      int          `json:"recipeId"`
	RecipeTitle   string       `json:"recipeTitle"`
	YieldQuantity int          `json:"yieldQuantity"`
	Items         []RecipeItem `json:"items"`
	TotalCost     float64      `json:"totalCost"`
}

func (r *Recipe) CalculateTotalCost() float64 {
	var total float64
	for _, item := range r.Items {
		total += item.CalculateSubCost()
	}
	r.TotalCost = total
	return total
}

// ProductionOrder — наряд/акт выпуска готовой продукции
type ProductionOrder struct {
	OrderID       int       `json:"orderId"`
	RecipeID      int       `json:"recipeId"`
	RecipeTitle   string    `json:"recipeTitle,omitempty"`
	BakerID       int       `json:"bakerId"`
	BakerName     string    `json:"bakerName,omitempty"`
	OrderDate     time.Time `json:"orderDate"`
	ProducedUnits int       `json:"producedUnits"`
	OrderStatus   string    `json:"orderStatus"`
	CancelReason  *string   `json:"cancelReason,omitempty"`
}

// DTO для HTTP-запросов
type StockAdjustmentRequest struct {
	IngredientID int     `json:"ingredientId"`
	Amount       float64 `json:"amount"`
	Reason       string  `json:"reason,omitempty"`
}

type CreateRecipeRequest struct {
	RecipeTitle   string `json:"recipeTitle"`
	YieldQuantity int    `json:"yieldQuantity"`
	Items         []struct {
		IngredientID     int     `json:"ingredientId"`
		RequiredQuantity float64 `json:"requiredQuantity"`
	} `json:"items"`
}

type ProduceRequest struct {
	ProductID     int `json:"productId"`
	BakerID       int `json:"bakerId"`
	ProducedUnits int `json:"producedUnits"`
}
