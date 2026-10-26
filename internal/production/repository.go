package production

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrIngredientNotFound = errors.New("сырьё не найдено")
	ErrProductNotFound    = errors.New("готовое изделие не найдено")
	ErrInsufficientStock  = errors.New("недостаточно остатка сырья на складе")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Список всего сырья на складе
func (r *Repository) ListIngredients(ctx context.Context) ([]Ingredient, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ingredient_id, name, current_stock, unit, purchase_price
		FROM ingredients ORDER BY ingredient_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Ingredient
	for rows.Next() {
		var i Ingredient
		if err := rows.Scan(&i.IngredientID, &i.Name, &i.CurrentStock, &i.Unit, &i.PurchasePrice); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, nil
}

// UC 4: Поступление сырья на склад
func (r *Repository) AddIngredientStock(ctx context.Context, ingredientID int, amount float64) (*Ingredient, error) {
	var i Ingredient
	err := r.db.QueryRow(ctx, `
		UPDATE ingredients
		SET current_stock = current_stock + $1
		WHERE ingredient_id = $2
		RETURNING ingredient_id, name, current_stock, unit, purchase_price
	`, amount, ingredientID).Scan(&i.IngredientID, &i.Name, &i.CurrentStock, &i.Unit, &i.PurchasePrice)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIngredientNotFound
		}
		return nil, err
	}
	return &i, nil
}

// UC 5: Списание сырья со склада (с проверкой остатка)
func (r *Repository) WriteOffIngredientStock(ctx context.Context, ingredientID int, amount float64) (*Ingredient, error) {
	var current float64
	err := r.db.QueryRow(ctx, `SELECT current_stock FROM ingredients WHERE ingredient_id = $1`, ingredientID).Scan(&current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIngredientNotFound
		}
		return nil, err
	}

	if current < amount {
		return nil, fmt.Errorf("%w: учётный остаток %.3f, попытка списания %.3f", ErrInsufficientStock, current, amount)
	}

	var i Ingredient
	err = r.db.QueryRow(ctx, `
		UPDATE ingredients
		SET current_stock = current_stock - $1
		WHERE ingredient_id = $2
		RETURNING ingredient_id, name, current_stock, unit, purchase_price
	`, amount, ingredientID).Scan(&i.IngredientID, &i.Name, &i.CurrentStock, &i.Unit, &i.PurchasePrice)
	return &i, err
}

// Список готовой продукции на витрине
func (r *Repository) ListProducts(ctx context.Context) ([]Product, error) {
	rows, err := r.db.Query(ctx, `
		SELECT product_id, recipe_id, name, barcode, retail_price, current_quantity
		FROM products ORDER BY product_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ProductID, &p.RecipeID, &p.Name, &p.Barcode, &p.RetailPrice, &p.CurrentQuantity); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

// Список всех рецептур с составом ингредиентов и расчётом себестоимости
func (r *Repository) ListRecipes(ctx context.Context) ([]Recipe, error) {
	rows, err := r.db.Query(ctx, `SELECT recipe_id, recipe_title, yield_quantity FROM recipes ORDER BY recipe_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var rec Recipe
		if err := rows.Scan(&rec.RecipeID, &rec.RecipeTitle, &rec.YieldQuantity); err != nil {
			return nil, err
		}
		recipes = append(recipes, rec)
	}

	for idx := range recipes {
		itemRows, err := r.db.Query(ctx, `
			SELECT ri.recipe_item_id, ri.recipe_id, ri.ingredient_id,
			       i.name, i.unit, i.purchase_price, ri.required_quantity
			FROM recipe_items ri
			JOIN ingredients i ON i.ingredient_id = ri.ingredient_id
			WHERE ri.recipe_id = $1
		`, recipes[idx].RecipeID)
		if err != nil {
			return nil, err
		}

		for itemRows.Next() {
			var item RecipeItem
			if err := itemRows.Scan(
				&item.RecipeItemID, &item.RecipeID, &item.IngredientID,
				&item.IngredientName, &item.Unit, &item.PurchasePrice, &item.RequiredQuantity,
			); err != nil {
				itemRows.Close()
				return nil, err
			}
			recipes[idx].Items = append(recipes[idx].Items, item)
		}
		itemRows.Close()
		recipes[idx].CalculateTotalCost()
	}

	return recipes, nil
}

// UC 2: Создание новой рецептуры (техкарты)
func (r *Repository) CreateRecipe(ctx context.Context, req CreateRecipeRequest) (*Recipe, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var recipe Recipe
	err = tx.QueryRow(ctx, `
		INSERT INTO recipes (recipe_title, yield_quantity)
		VALUES ($1, $2)
		RETURNING recipe_id, recipe_title, yield_quantity
	`, req.RecipeTitle, req.YieldQuantity).Scan(&recipe.RecipeID, &recipe.RecipeTitle, &recipe.YieldQuantity)
	if err != nil {
		return nil, err
	}

	for _, item := range req.Items {
		var ri RecipeItem
		err := tx.QueryRow(ctx, `
			INSERT INTO recipe_items (recipe_id, ingredient_id, required_quantity)
			VALUES ($1, $2, $3)
			RETURNING recipe_item_id, recipe_id, ingredient_id, required_quantity
		`, recipe.RecipeID, item.IngredientID, item.RequiredQuantity).Scan(
			&ri.RecipeItemID, &ri.RecipeID, &ri.IngredientID, &ri.RequiredQuantity,
		)
		if err != nil {
			return nil, err
		}
		recipe.Items = append(recipe.Items, ri)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &recipe, nil
}

// UC 6 + UC 7: Зафиксировать выпуск продукции со списанием сырья по рецептуре в одной транзакции
func (r *Repository) ExecuteProductionOrder(ctx context.Context, req ProduceRequest) (*ProductionOrder, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Находим продукт и привязанную к нему техкарту (рецептуру)
	var recipeID int
	var yieldQty int
	err = tx.QueryRow(ctx, `
		SELECT p.recipe_id, r.yield_quantity
		FROM products p
		JOIN recipes r ON r.recipe_id = p.recipe_id
		WHERE p.product_id = $1
	`, req.ProductID).Scan(&recipeID, &yieldQty)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	// Коэффициент пересчёта: сколько партий/единиц относительно выхода по техкарте
	multiplier := float64(req.ProducedUnits) / float64(yieldQty)

	// 2. Считываем нормы расхода и текущие остатки сырья (с блокировкой строк FOR UPDATE)
	rows, err := tx.Query(ctx, `
		SELECT i.ingredient_id, i.name, i.current_stock, i.unit, ri.required_quantity
		FROM recipe_items ri
		JOIN ingredients i ON i.ingredient_id = ri.ingredient_id
		WHERE ri.recipe_id = $1
		FOR UPDATE OF i
	`, recipeID)
	if err != nil {
		return nil, err
	}

	type deduction struct {
		ingredientID int
		name         string
		unit         string
		currentStock float64
		required     float64
	}
	var deductions []deduction

	for rows.Next() {
		var d deduction
		var baseReq float64
		if err := rows.Scan(&d.ingredientID, &d.name, &d.currentStock, &d.unit, &baseReq); err != nil {
			rows.Close()
			return nil, err
		}
		d.required = baseReq * multiplier
		deductions = append(deductions, d)
	}
	rows.Close()

	// 3. Проверяем достаточность каждого ингредиента (альтернативный сценарий UC 6 / UC 7)
	for _, d := range deductions {
		if d.currentStock < d.required {
			shortage := d.required - d.currentStock
			return nil, fmt.Errorf("%w: %s — не хватает %.3f %s (нужно %.3f, на складе %.3f)",
				ErrInsufficientStock, d.name, shortage, d.unit, d.required, d.currentStock)
		}
	}

	// 4. Списываем сырьё со склада (UC 7)
	for _, d := range deductions {
		_, err := tx.Exec(ctx, `
			UPDATE ingredients
			SET current_stock = current_stock - $1
			WHERE ingredient_id = $2
		`, d.required, d.ingredientID)
		if err != nil {
			return nil, err
		}
	}

	// 5. Увеличиваем остаток готовой выпечки на витрине (UC 6)
	_, err = tx.Exec(ctx, `
		UPDATE products
		SET current_quantity = current_quantity + $1
		WHERE product_id = $2
	`, req.ProducedUnits, req.ProductID)
	if err != nil {
		return nil, err
	}

	// 6. Создаём запись акта выпуска в production_orders
	var order ProductionOrder
	err = tx.QueryRow(ctx, `
		INSERT INTO production_orders (recipe_id, baker_id, produced_units, order_status)
		VALUES ($1, $2, $3, 'COMPLETED')
		RETURNING order_id, recipe_id, baker_id, order_date, produced_units, order_status
	`, recipeID, req.BakerID, req.ProducedUnits).Scan(
		&order.OrderID, &order.RecipeID, &order.BakerID,
		&order.OrderDate, &order.ProducedUnits, &order.OrderStatus,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &order, nil
}

// История актов выпуска продукции
func (r *Repository) ListProductionOrders(ctx context.Context) ([]ProductionOrder, error) {
	rows, err := r.db.Query(ctx, `
		SELECT po.order_id, po.recipe_id, r.recipe_title, po.baker_id, u.full_name,
		       po.order_date, po.produced_units, po.order_status, po.cancel_reason
		FROM production_orders po
		JOIN recipes r ON r.recipe_id = po.recipe_id
		JOIN users u ON u.id = po.baker_id
		ORDER BY po.order_date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []ProductionOrder
	for rows.Next() {
		var o ProductionOrder
		if err := rows.Scan(
			&o.OrderID, &o.RecipeID, &o.RecipeTitle, &o.BakerID, &o.BakerName,
			&o.OrderDate, &o.ProducedUnits, &o.OrderStatus, &o.CancelReason,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}
