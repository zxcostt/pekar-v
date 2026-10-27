package sales

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProductOutOfStock = errors.New("выбранная позиция отсутствует на витрине в нужном количестве")
	ErrReceiptNotFound   = errors.New("чек с данным номером не зарегистрирован в системе")
	ErrAlreadyRefunded   = errors.New("по данному чеку уже был оформлен возврат")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateSaleTransaction — атомарная транзакция продажи (UC 8 + UC 9 + UC 10 + UC 11..13)
func (r *Repository) CreateSaleTransaction(
	ctx context.Context,
	req CreateSaleRequest,
	fiscalService *FiscalMockService,
	validatePayment func(grandTotal float64) (*PaymentRecord, error),
) (*Receipt, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	receipt := &Receipt{
		CashierID:    req.CashierID,
		DiscountRate: req.DiscountRate,
	}

	// 1. Проверяем остатки выпечки на витрине (с блокировкой FOR UPDATE) и собираем позиции чека
	for _, reqItem := range req.Items {
		var name string
		var price float64
		var availableQty int

		err := tx.QueryRow(ctx, `
			SELECT name, retail_price, current_quantity
			FROM products
			WHERE product_id = $1
			FOR UPDATE
		`, reqItem.ProductID).Scan(&name, &price, &availableQty)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("товар ID=%d не найден", reqItem.ProductID)
			}
			return nil, err
		}

		// Альтернативная ветка UC 8: Товар закончился на витрине
		if availableQty < reqItem.Quantity {
			return nil, fmt.Errorf("%w: «%s» (доступно: %d шт., запрошено: %d шт.)",
				ErrProductOutOfStock, name, availableQty, reqItem.Quantity)
		}

		// Списываем проданную выпечку с витрины
		_, err = tx.Exec(ctx, `
			UPDATE products
			SET current_quantity = current_quantity - $1
			WHERE product_id = $2
		`, reqItem.Quantity, reqItem.ProductID)
		if err != nil {
			return nil, err
		}

		receipt.AddPosition(reqItem.ProductID, name, price, reqItem.Quantity)
	}

	// 2. Рассчитываем итог чека с учетом скидки (UC 9)
	grandTotal := receipt.CalculateGrandTotal(req.DiscountRate)

	// 3. Проводим оплату (UC 11 -> UC 12 Наличные / UC 13 Карта)
	paymentRec, err := validatePayment(grandTotal)
	if err != nil {
		return nil, err
	}

	// 4. Печатаем чек на фискальном регистраторе (UC 10)
	fiscalDoc, err := fiscalService.PrintFiscalReceipt(grandTotal, req.PaymentType)
	if err != nil {
		return nil, err
	}
	receipt.RegisterFiscalSlip(fiscalDoc)

	// 5. Сохраняем шапку чека в таблицу receipts
	err = tx.QueryRow(ctx, `
		INSERT INTO receipts (cashier_id, discount_rate, is_completed, is_refunded, fiscal_doc_number)
		VALUES ($1, $2, TRUE, FALSE, $3)
		RETURNING receipt_id, receipt_date
	`, receipt.CashierID, receipt.DiscountRate, receipt.FiscalDocNumber).Scan(&receipt.ReceiptID, &receipt.ReceiptDate)
	if err != nil {
		return nil, err
	}

	// 6. Сохраняем строки чека в receipt_items
	for idx := range receipt.Items {
		item := &receipt.Items[idx]
		item.ReceiptID = receipt.ReceiptID
		err := tx.QueryRow(ctx, `
			INSERT INTO receipt_items (receipt_id, product_id, quantity, unit_price)
			VALUES ($1, $2, $3, $4)
			RETURNING receipt_item_id
		`, receipt.ReceiptID, item.ProductID, item.Quantity, item.UnitPrice).Scan(&item.ReceiptItemID)
		if err != nil {
			return nil, err
		}
	}

	// 7. Сохраняем платеж в таблицу payments
	paymentRec.ReceiptID = receipt.ReceiptID
	err = tx.QueryRow(ctx, `
		INSERT INTO payments (
			receipt_id, payment_type, amount,
			tendered_amount, change_amount, card_pan_masked, authorization_code
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING payment_id, payment_date
	`,
		paymentRec.ReceiptID, paymentRec.PaymentType, paymentRec.Amount,
		paymentRec.TenderedAmount, paymentRec.ChangeAmount,
		paymentRec.CardPanMasked, paymentRec.AuthorizationCode,
	).Scan(&paymentRec.PaymentID, &paymentRec.PaymentDate)
	if err != nil {
		return nil, err
	}

	receipt.PaymentDetails = paymentRec

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return receipt, nil
}

// RefundReceiptByFiscalDoc — UC 14: Оформить возврат товара по фискальному номеру чека
func (r *Repository) RefundReceiptByFiscalDoc(ctx context.Context, fiscalDoc string) (*Receipt, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var rec Receipt
	err = tx.QueryRow(ctx, `
		SELECT receipt_id, cashier_id, receipt_date, discount_rate, is_completed, is_refunded, fiscal_doc_number
		FROM receipts
		WHERE fiscal_doc_number = $1
		FOR UPDATE
	`, fiscalDoc).Scan(
		&rec.ReceiptID, &rec.CashierID, &rec.ReceiptDate,
		&rec.DiscountRate, &rec.IsCompleted, &rec.IsRefunded, &rec.FiscalDocNumber,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReceiptNotFound
		}
		return nil, err
	}

	if rec.IsRefunded {
		return nil, ErrAlreadyRefunded
	}

	// Считываем позиции возвращаемого чека
	rows, err := tx.Query(ctx, `
		SELECT ri.receipt_item_id, ri.product_id, p.name, ri.quantity, ri.unit_price
		FROM receipt_items ri
		JOIN products p ON p.product_id = ri.product_id
		WHERE ri.receipt_id = $1
	`, rec.ReceiptID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var item ReceiptItem
		if err := rows.Scan(&item.ReceiptItemID, &item.ProductID, &item.ProductName, &item.Quantity, &item.UnitPrice); err != nil {
			rows.Close()
			return nil, err
		}
		item.ReceiptID = rec.ReceiptID
		item.CalculateLineTotal()
		rec.Items = append(rec.Items, item)
	}
	rows.Close()

	// Возвращаем продукцию на витрину
	for _, item := range rec.Items {
		_, err := tx.Exec(ctx, `
			UPDATE products
			SET current_quantity = current_quantity + $1
			WHERE product_id = $2
		`, item.Quantity, item.ProductID)
		if err != nil {
			return nil, err
		}
	}

	// Отмечаем чек как возвращённый
	_, err = tx.Exec(ctx, `UPDATE receipts SET is_refunded = TRUE WHERE receipt_id = $1`, rec.ReceiptID)
	if err != nil {
		return nil, err
	}
	rec.IsRefunded = true
	rec.CalculateGrandTotal(rec.DiscountRate)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &rec, nil
}

// ListReceipts — список всех чеков с позициями и платежами
func (r *Repository) ListReceipts(ctx context.Context) ([]Receipt, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.receipt_id, r.cashier_id, u.full_name, r.receipt_date,
		       r.discount_rate, r.is_completed, r.is_refunded, COALESCE(r.fiscal_doc_number, '')
		FROM receipts r
		JOIN users u ON u.id = r.cashier_id
		ORDER BY r.receipt_date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []Receipt
	for rows.Next() {
		var rec Receipt
		if err := rows.Scan(
			&rec.ReceiptID, &rec.CashierID, &rec.CashierName, &rec.ReceiptDate,
			&rec.DiscountRate, &rec.IsCompleted, &rec.IsRefunded, &rec.FiscalDocNumber,
		); err != nil {
			return nil, err
		}
		receipts = append(receipts, rec)
	}

	for i := range receipts {
		rec := &receipts[i]

		// Загружаем строки чека
		itemRows, err := r.db.Query(ctx, `
			SELECT ri.receipt_item_id, ri.receipt_id, ri.product_id, p.name, ri.quantity, ri.unit_price
			FROM receipt_items ri
			JOIN products p ON p.product_id = ri.product_id
			WHERE ri.receipt_id = $1
		`, rec.ReceiptID)
		if err != nil {
			return nil, err
		}
		for itemRows.Next() {
			var item ReceiptItem
			if err := itemRows.Scan(&item.ReceiptItemID, &item.ReceiptID, &item.ProductID, &item.ProductName, &item.Quantity, &item.UnitPrice); err != nil {
				itemRows.Close()
				return nil, err
			}
			item.CalculateLineTotal()
			rec.Items = append(rec.Items, item)
		}
		itemRows.Close()
		rec.CalculateGrandTotal(rec.DiscountRate)

		// Загружаем информацию об оплате
		var pay PaymentRecord
		err = r.db.QueryRow(ctx, `
			SELECT payment_id, receipt_id, payment_type, amount, payment_date,
			       tendered_amount, change_amount, card_pan_masked, authorization_code
			FROM payments
			WHERE receipt_id = $1
		`, rec.ReceiptID).Scan(
			&pay.PaymentID, &pay.ReceiptID, &pay.PaymentType, &pay.Amount, &pay.PaymentDate,
			&pay.TenderedAmount, &pay.ChangeAmount, &pay.CardPanMasked, &pay.AuthorizationCode,
		)
		if err == nil {
			rec.PaymentDetails = &pay
		}
	}

	return receipts, nil
}

// GetShiftReport — UC 3: Формировать сменный отчет
func (r *Repository) GetShiftReport(ctx context.Context) (*ShiftReportDTO, error) {
	report := &ShiftReportDTO{
		ShiftID:  1,
		OpenedAt: time.Now().Truncate(24 * time.Hour),
	}

	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN NOT r.is_refunded THEN p.amount ELSE 0 END), 0) AS total_revenue,
			COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CASH' THEN p.amount ELSE 0 END), 0) AS cash_subtotal,
			COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CARD' THEN p.amount ELSE 0 END), 0) AS card_subtotal,
			COUNT(CASE WHEN NOT r.is_refunded THEN 1 END) AS active_receipts,
			COUNT(CASE WHEN r.is_refunded THEN 1 END) AS refunded_receipts
		FROM receipts r
		JOIN payments p ON p.receipt_id = r.receipt_id
	`).Scan(
		&report.TotalRevenue,
		&report.CashSubtotal,
		&report.CardSubtotal,
		&report.ReceiptCount,
		&report.RefundedCount,
	)
	if err != nil {
		return nil, err
	}

	return report, nil
}
