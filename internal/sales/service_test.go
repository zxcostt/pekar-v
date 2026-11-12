package sales

import (
	"context"
	"errors"
	"testing"
)

func TestSalesShiftValidation(t *testing.T) {
	svc := NewService(nil, NewFiscalMockService())
	ctx := context.Background()

	t.Run("OpenShift missing cashier", func(t *testing.T) {
		_, err := svc.OpenShift(ctx, OpenShiftRequest{CashierID: 0, InitialCash: 1000})
		if err == nil || err.Error() != "не указан ID кассира для открытия смены" {
			t.Errorf("expected missing cashier error, got %v", err)
		}
	})

	t.Run("OpenShift negative cash", func(t *testing.T) {
		_, err := svc.OpenShift(ctx, OpenShiftRequest{CashierID: 1, InitialCash: -500})
		if !errors.Is(err, ErrInvalidCashValue) {
			t.Errorf("expected ErrInvalidCashValue, got %v", err)
		}
	})

	t.Run("CloseShift negative cash", func(t *testing.T) {
		_, err := svc.CloseShift(ctx, CloseShiftRequest{CashierID: 1, ActualCash: -100, EncashmentAmount: 0})
		if !errors.Is(err, ErrInvalidCashValue) {
			t.Errorf("expected ErrInvalidCashValue, got %v", err)
		}
	})

	t.Run("CloseShift negative encashment", func(t *testing.T) {
		_, err := svc.CloseShift(ctx, CloseShiftRequest{CashierID: 1, ActualCash: 1000, EncashmentAmount: -50})
		if !errors.Is(err, ErrInvalidCashValue) {
			t.Errorf("expected ErrInvalidCashValue, got %v", err)
		}
	})

	t.Run("ProcessRefund missing fiscal doc", func(t *testing.T) {
		_, _, err := svc.ProcessRefund(ctx, RefundRequest{FiscalDocNumber: ""})
		if err == nil {
			t.Errorf("expected error for empty fiscal doc number")
		}
	})

	t.Run("FiscalMock PrintXReport and PrintZReport", func(t *testing.T) {
		fiscal := NewFiscalMockService()
		xDoc, err := fiscal.PrintXReport(1, 1500.50)
		if err != nil || len(xDoc) == 0 {
			t.Errorf("failed PrintXReport: %v", err)
		}

		zDoc, err := fiscal.PrintZReport(1, 1500.50)
		if err != nil || len(zDoc) == 0 {
			t.Errorf("failed PrintZReport: %v", err)
		}
	})
}
