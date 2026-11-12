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
	repo := &Repository{db: db}
	repo.ensureTables(context.Background())
	return repo
}

func (r *Repository) ensureTables(ctx context.Context) {
	_, _ = r.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS cash_shifts (
			shift_id SERIAL PRIMARY KEY,
			cashier_id INT NOT NULL REFERENCES users(id),
			register_number INT NOT NULL DEFAULT 1,
			opened_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			closed_at TIMESTAMP,
			initial_cash NUMERIC(10, 2) NOT NULL DEFAULT 0 CHECK (initial_cash >= 0),
			final_cash NUMERIC(10, 2),
			cash_revenue NUMERIC(10, 2) NOT NULL DEFAULT 0,
			card_revenue NUMERIC(10, 2) NOT NULL DEFAULT 0,
			encashment_amount NUMERIC(10, 2) NOT NULL DEFAULT 0,
			z_report_doc VARCHAR(100),
			status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED'))
		);
		ALTER TABLE receipts ADD COLUMN IF NOT EXISTS shift_id INT REFERENCES cash_shifts(shift_id) ON DELETE SET NULL;
	`)
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
	var activeShiftID *int
	_ = tx.QueryRow(ctx, `SELECT shift_id FROM cash_shifts WHERE status = 'OPEN' ORDER BY shift_id DESC LIMIT 1`).Scan(&activeShiftID)
	receipt.ShiftID = activeShiftID

	err = tx.QueryRow(ctx, `
		INSERT INTO receipts (cashier_id, shift_id, discount_rate, is_completed, is_refunded, fiscal_doc_number)
		VALUES ($1, $2, $3, TRUE, FALSE, $4)
		RETURNING receipt_id, receipt_date
	`, receipt.CashierID, receipt.ShiftID, receipt.DiscountRate, receipt.FiscalDocNumber).Scan(&receipt.ReceiptID, &receipt.ReceiptDate)
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
		SELECT r.receipt_id, r.shift_id, r.cashier_id, u.full_name, r.receipt_date,
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
			&rec.ReceiptID, &rec.ShiftID, &rec.CashierID, &rec.CashierName, &rec.ReceiptDate,
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

// GetActiveShift — получение текущей открытой кассовой смены
func (r *Repository) GetActiveShift(ctx context.Context) (*CashShift, error) {
	r.ensureTables(ctx)

	var s CashShift
	err := r.db.QueryRow(ctx, `
		SELECT s.shift_id, s.cashier_id, u.full_name, s.register_number,
		       s.opened_at, s.closed_at, s.initial_cash, s.final_cash,
		       s.cash_revenue, s.card_revenue, s.encashment_amount, s.z_report_doc, s.status
		FROM cash_shifts s
		JOIN users u ON u.id = s.cashier_id
		WHERE s.status = 'OPEN'
		ORDER BY s.shift_id DESC
		LIMIT 1
	`).Scan(
		&s.ShiftID, &s.CashierID, &s.CashierName, &s.RegisterNumber,
		&s.OpenedAt, &s.ClosedAt, &s.InitialCash, &s.FinalCash,
		&s.CashRevenue, &s.CardRevenue, &s.EncashmentAmount, &s.ZReportDoc, &s.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// OpenShift — открытие новой кассовой смены с внесением начального размена
func (r *Repository) OpenShift(ctx context.Context, req OpenShiftRequest) (*CashShift, error) {
	r.ensureTables(ctx)

	active, err := r.GetActiveShift(ctx)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, fmt.Errorf("кассовая смена №%d уже открыта кассиром %s", active.ShiftID, active.CashierName)
	}

	regNum := req.RegisterNumber
	if regNum <= 0 {
		regNum = 1
	}

	var s CashShift
	err = r.db.QueryRow(ctx, `
		INSERT INTO cash_shifts (cashier_id, register_number, initial_cash, status, opened_at)
		VALUES ($1, $2, $3, 'OPEN', CURRENT_TIMESTAMP)
		RETURNING shift_id, cashier_id, register_number, opened_at, initial_cash, cash_revenue, card_revenue, encashment_amount, status
	`, req.CashierID, regNum, req.InitialCash).Scan(
		&s.ShiftID, &s.CashierID, &s.RegisterNumber, &s.OpenedAt, &s.InitialCash,
		&s.CashRevenue, &s.CardRevenue, &s.EncashmentAmount, &s.Status,
	)
	if err != nil {
		return nil, err
	}

	_ = r.db.QueryRow(ctx, `SELECT full_name FROM users WHERE id = $1`, s.CashierID).Scan(&s.CashierName)
	return &s, nil
}

// GetXReport — снятие промежуточного фискального отчёта без гашения
func (r *Repository) GetXReport(ctx context.Context, shiftID int, fiscal *FiscalMockService) (*XReportDTO, error) {
	r.ensureTables(ctx)

	var (
		s           CashShift
		cashierName string
	)
	err := r.db.QueryRow(ctx, `
		SELECT s.shift_id, s.cashier_id, u.full_name, s.register_number, s.opened_at, s.initial_cash
		FROM cash_shifts s
		JOIN users u ON u.id = s.cashier_id
		WHERE s.shift_id = $1
	`, shiftID).Scan(&s.ShiftID, &s.CashierID, &cashierName, &s.RegisterNumber, &s.OpenedAt, &s.InitialCash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("смена №%d не найдена", shiftID)
		}
		return nil, err
	}

	var (
		cashRevenue   float64
		cardRevenue   float64
		receiptCount  int
		refundedCount int
	)
	err = r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CASH' THEN p.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CARD' THEN p.amount ELSE 0 END), 0),
			COUNT(CASE WHEN NOT r.is_refunded THEN 1 END),
			COUNT(CASE WHEN r.is_refunded THEN 1 END)
		FROM receipts r
		JOIN payments p ON p.receipt_id = r.receipt_id
		WHERE r.shift_id = $1
	`, shiftID).Scan(&cashRevenue, &cardRevenue, &receiptCount, &refundedCount)
	if err != nil {
		return nil, err
	}

	var (
		writtenOffCount int
		writtenOffTotal float64
	)
	_ = r.db.QueryRow(ctx, `
		SELECT 
			COALESCE(SUM(w.quantity), 0),
			COALESCE(SUM(w.quantity * p.retail_price), 0)
		FROM product_write_offs w
		JOIN products p ON p.product_id = w.product_id
		WHERE w.write_off_date >= $1
	`, s.OpenedAt).Scan(&writtenOffCount, &writtenOffTotal)

	totalRevenue := cashRevenue + cardRevenue
	expectedInDrawer := s.InitialCash + cashRevenue

	slip, _ := fiscal.PrintXReport(shiftID, totalRevenue)

	return &XReportDTO{
		ShiftID:              s.ShiftID,
		RegisterNumber:       s.RegisterNumber,
		CashierID:            s.CashierID,
		CashierName:          cashierName,
		OpenedAt:             s.OpenedAt,
		ReportTime:           time.Now(),
		InitialCash:          s.InitialCash,
		CashRevenue:          cashRevenue,
		CardRevenue:          cardRevenue,
		TotalRevenue:         totalRevenue,
		ReceiptCount:         receiptCount,
		RefundedCount:        refundedCount,
		ExpectedCashInDrawer: expectedInDrawer,
		WrittenOffCount:      writtenOffCount,
		WrittenOffTotal:      writtenOffTotal,
		FiscalSlip:           slip,
	}, nil
}

// CloseShift — закрытие смены, снятие Z-отчёта и инкассация
func (r *Repository) CloseShift(ctx context.Context, req CloseShiftRequest, fiscal *FiscalMockService) (*CloseShiftResponse, error) {
	r.ensureTables(ctx)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var s CashShift
	err = tx.QueryRow(ctx, `
		SELECT s.shift_id, s.cashier_id, u.full_name, s.register_number, s.opened_at, s.initial_cash
		FROM cash_shifts s
		JOIN users u ON u.id = s.cashier_id
		WHERE s.status = 'OPEN'
		FOR UPDATE OF s
	`).Scan(&s.ShiftID, &s.CashierID, &s.CashierName, &s.RegisterNumber, &s.OpenedAt, &s.InitialCash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("нет открытой кассовой смены для закрытия")
		}
		return nil, err
	}

	var (
		cashRevenue float64
		cardRevenue float64
	)
	err = tx.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CASH' THEN p.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CARD' THEN p.amount ELSE 0 END), 0)
		FROM receipts r
		JOIN payments p ON p.receipt_id = r.receipt_id
		WHERE r.shift_id = $1
	`, s.ShiftID).Scan(&cashRevenue, &cardRevenue)
	if err != nil {
		return nil, err
	}

	totalRevenue := cashRevenue + cardRevenue
	expectedCash := s.InitialCash + cashRevenue
	discrepancy := req.ActualCash - expectedCash

	// Печатаем фискальный Z-отчёт с гашением
	zReportDoc, err := fiscal.PrintZReport(s.ShiftID, totalRevenue)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE cash_shifts
		SET status = 'CLOSED',
		    closed_at = $1,
		    final_cash = $2,
		    cash_revenue = $3,
		    card_revenue = $4,
		    encashment_amount = $5,
		    z_report_doc = $6
		WHERE shift_id = $7
	`, now, req.ActualCash, cashRevenue, cardRevenue, req.EncashmentAmount, zReportDoc, s.ShiftID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.ClosedAt = &now
	s.FinalCash = &req.ActualCash
	s.CashRevenue = cashRevenue
	s.CardRevenue = cardRevenue
	s.EncashmentAmount = req.EncashmentAmount
	s.ZReportDoc = &zReportDoc
	s.Status = "CLOSED"

	msg := "Кассовая смена успешно закрыта. Z-отчёт распечатан."
	if discrepancy > 0 {
		msg += fmt.Sprintf(" Зафиксирован излишек в кассе: +%.2f ₽.", discrepancy)
	} else if discrepancy < 0 {
		msg += fmt.Sprintf(" ВНИМАНИЕ: Зафиксирована недостача в кассе: %.2f ₽.", discrepancy)
	} else {
		msg += " Расхождений в кассе нет (идеальный баланс)."
	}

	return &CloseShiftResponse{
		Shift:       &s,
		ZReportSlip: zReportDoc,
		Discrepancy: discrepancy,
		Message:     msg,
	}, nil
}

// ListShifts — получение истории всех кассовых смен
func (r *Repository) ListShifts(ctx context.Context) ([]CashShift, error) {
	r.ensureTables(ctx)

	rows, err := r.db.Query(ctx, `
		SELECT s.shift_id, s.cashier_id, u.full_name, s.register_number,
		       s.opened_at, s.closed_at, s.initial_cash, s.final_cash,
		       s.cash_revenue, s.card_revenue, s.encashment_amount, s.z_report_doc, s.status
		FROM cash_shifts s
		JOIN users u ON u.id = s.cashier_id
		ORDER BY s.shift_id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shifts []CashShift
	for rows.Next() {
		var s CashShift
		if err := rows.Scan(
			&s.ShiftID, &s.CashierID, &s.CashierName, &s.RegisterNumber,
			&s.OpenedAt, &s.ClosedAt, &s.InitialCash, &s.FinalCash,
			&s.CashRevenue, &s.CardRevenue, &s.EncashmentAmount, &s.ZReportDoc, &s.Status,
		); err != nil {
			return nil, err
		}
		shifts = append(shifts, s)
	}
	return shifts, nil
}

// GetShiftReport — UC 3: Формировать сменный отчет
func (r *Repository) GetShiftReport(ctx context.Context) (*ShiftReportDTO, error) {
	r.ensureTables(ctx)

	active, err := r.GetActiveShift(ctx)
	if err != nil {
		return nil, err
	}

	if active != nil {
		report := &ShiftReportDTO{
			ShiftID:  active.ShiftID,
			OpenedAt: active.OpenedAt,
		}

		err = r.db.QueryRow(ctx, `
			SELECT
				COALESCE(SUM(CASE WHEN NOT r.is_refunded THEN p.amount ELSE 0 END), 0) AS total_revenue,
				COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CASH' THEN p.amount ELSE 0 END), 0) AS cash_subtotal,
				COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CARD' THEN p.amount ELSE 0 END), 0) AS card_subtotal,
				COUNT(CASE WHEN NOT r.is_refunded THEN 1 END) AS active_receipts,
				COUNT(CASE WHEN r.is_refunded THEN 1 END) AS refunded_receipts
			FROM receipts r
			JOIN payments p ON p.receipt_id = r.receipt_id
			WHERE r.shift_id = $1
		`, active.ShiftID).Scan(
			&report.TotalRevenue,
			&report.CashSubtotal,
			&report.CardSubtotal,
			&report.ReceiptCount,
			&report.RefundedCount,
		)
		if err != nil {
			return nil, err
		}

		var (
			writtenOffCount int
			writtenOffTotal float64
		)
		_ = r.db.QueryRow(ctx, `
			SELECT 
				COALESCE(SUM(w.quantity), 0),
				COALESCE(SUM(w.quantity * p.retail_price), 0)
			FROM product_write_offs w
			JOIN products p ON p.product_id = w.product_id
			WHERE w.write_off_date >= $1
		`, active.OpenedAt).Scan(&writtenOffCount, &writtenOffTotal)

		report.WrittenOffCount = writtenOffCount
		report.WrittenOffTotal = writtenOffTotal
		return report, nil
	}

	// Если активной смены нет, ищем последнюю закрытую смену
	var lastShift CashShift
	err = r.db.QueryRow(ctx, `
		SELECT shift_id, opened_at, initial_cash, cash_revenue, card_revenue
		FROM cash_shifts
		ORDER BY shift_id DESC
		LIMIT 1
	`).Scan(&lastShift.ShiftID, &lastShift.OpenedAt, &lastShift.InitialCash, &lastShift.CashRevenue, &lastShift.CardRevenue)

	if err == nil {
		report := &ShiftReportDTO{
			ShiftID:      lastShift.ShiftID,
			OpenedAt:     lastShift.OpenedAt,
			TotalRevenue: lastShift.CashRevenue + lastShift.CardRevenue,
			CashSubtotal: lastShift.CashRevenue,
			CardSubtotal: lastShift.CardRevenue,
		}
		_ = r.db.QueryRow(ctx, `
			SELECT COUNT(CASE WHEN NOT r.is_refunded THEN 1 END), COUNT(CASE WHEN r.is_refunded THEN 1 END)
			FROM receipts r WHERE r.shift_id = $1
		`, lastShift.ShiftID).Scan(&report.ReceiptCount, &report.RefundedCount)

		return report, nil
	}

	// Резервный fallback для пустой таблицы
	report := &ShiftReportDTO{
		ShiftID:  0,
		OpenedAt: time.Now().Truncate(24 * time.Hour),
	}
	_ = r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN NOT r.is_refunded THEN p.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CASH' THEN p.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN NOT r.is_refunded AND p.payment_type = 'CARD' THEN p.amount ELSE 0 END), 0),
			COUNT(CASE WHEN NOT r.is_refunded THEN 1 END),
			COUNT(CASE WHEN r.is_refunded THEN 1 END)
		FROM receipts r
		JOIN payments p ON p.receipt_id = r.receipt_id
	`).Scan(&report.TotalRevenue, &report.CashSubtotal, &report.CardSubtotal, &report.ReceiptCount, &report.RefundedCount)

	return report, nil
}
