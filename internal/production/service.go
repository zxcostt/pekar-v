package production

import (
	"context"
	"errors"
)

var ErrInvalidQuantity = errors.New("количество и нормы расхода должны быть строго больше нуля")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetIngredients(ctx context.Context) ([]Ingredient, error) {
	return s.repo.ListIngredients(ctx)
}

func (s *Service) SupplyIngredient(ctx context.Context, req StockAdjustmentRequest) (*Ingredient, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidQuantity
	}
	return s.repo.AddIngredientStock(ctx, req.IngredientID, req.Amount)
}

func (s *Service) WriteOffIngredient(ctx context.Context, req StockAdjustmentRequest) (*Ingredient, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidQuantity
	}
	return s.repo.WriteOffIngredientStock(ctx, req.IngredientID, req.Amount)
}

func (s *Service) GetProducts(ctx context.Context) ([]Product, error) {
	return s.repo.ListProducts(ctx)
}

func (s *Service) GetRecipes(ctx context.Context) ([]Recipe, error) {
	return s.repo.ListRecipes(ctx)
}

func (s *Service) CreateRecipe(ctx context.Context, req CreateRecipeRequest) (*Recipe, error) {
	if req.RecipeTitle == "" || req.YieldQuantity <= 0 || len(req.Items) == 0 {
		return nil, ErrInvalidQuantity
	}
	for _, item := range req.Items {
		if item.RequiredQuantity <= 0 {
			return nil, ErrInvalidQuantity
		}
	}
	return s.repo.CreateRecipe(ctx, req)
}

func (s *Service) ProduceProduct(ctx context.Context, req ProduceRequest) (*ProductionOrder, error) {
	if req.ProducedUnits <= 0 {
		return nil, ErrInvalidQuantity
	}
	return s.repo.ExecuteProductionOrder(ctx, req)
}

func (s *Service) GetProductionOrders(ctx context.Context) ([]ProductionOrder, error) {
	return s.repo.ListProductionOrders(ctx)
}
