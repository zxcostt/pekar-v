package production

import (
	"context"
	"errors"
	"testing"
)

func TestServiceValidation(t *testing.T) {
	svc := NewService(nil)
	ctx := context.Background()

	t.Run("SupplyIngredient invalid amount", func(t *testing.T) {
		_, err := svc.SupplyIngredient(ctx, StockAdjustmentRequest{IngredientID: 1, Amount: 0})
		if !errors.Is(err, ErrInvalidQuantity) {
			t.Errorf("expected ErrInvalidQuantity, got %v", err)
		}
	})

	t.Run("WriteOffIngredient invalid amount", func(t *testing.T) {
		_, err := svc.WriteOffIngredient(ctx, StockAdjustmentRequest{IngredientID: 1, Amount: -5})
		if !errors.Is(err, ErrInvalidQuantity) {
			t.Errorf("expected ErrInvalidQuantity, got %v", err)
		}
	})

	t.Run("ProduceProduct invalid units", func(t *testing.T) {
		_, err := svc.ProduceProduct(ctx, ProduceRequest{ProductID: 1, BakerID: 1, ProducedUnits: 0})
		if !errors.Is(err, ErrInvalidQuantity) {
			t.Errorf("expected ErrInvalidQuantity, got %v", err)
		}
	})

	t.Run("CreateRecipe invalid title or yield", func(t *testing.T) {
		_, err := svc.CreateRecipe(ctx, CreateRecipeRequest{RecipeTitle: "", YieldQuantity: 10})
		if !errors.Is(err, ErrInvalidQuantity) {
			t.Errorf("expected ErrInvalidQuantity, got %v", err)
		}
	})

	t.Run("WriteOffProduct missing product", func(t *testing.T) {
		_, err := svc.WriteOffProduct(ctx, ProductWriteOffRequest{ProductID: 0, Quantity: 5})
		if err == nil || err.Error() != "не выбрано изделие для списания" {
			t.Errorf("expected 'не выбрано изделие для списания', got %v", err)
		}
	})

	t.Run("WriteOffProduct zero or negative quantity", func(t *testing.T) {
		_, err := svc.WriteOffProduct(ctx, ProductWriteOffRequest{ProductID: 1, Quantity: 0})
		if !errors.Is(err, ErrInvalidQuantity) {
			t.Errorf("expected ErrInvalidQuantity, got %v", err)
		}
	})
}
