package sales

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrEmptyCart        = errors.New("корзина чека пуста")
	ErrInvalidDiscount  = errors.New("скидка должна быть в диапазоне от 0 до 100%")
	ErrInsufficientCash = errors.New("внесено недостаточно наличных средств")
	ErrInvalidPayType   = errors.New("неподдерживаемый способ оплаты (допустимо CASH или CARD)")
)

type Service struct {
	repo   *Repository
	fiscal *FiscalMockService
}

func NewService(repo *Repository, fiscal *FiscalMockService) *Service {
	return &Service{
		repo:   repo,
		fiscal: fiscal,
	}
}

func (s *Service) ProcessSale(ctx context.Context, req CreateSaleRequest) (*Receipt, error) {
	if len(req.Items) == 0 {
		return nil, ErrEmptyCart
	}
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return nil, errors.New("количество товара в чеке должно быть больше нуля")
		}
	}
	if req.DiscountRate < 0 || req.DiscountRate > 100 {
		return nil, ErrInvalidDiscount
	}

	validatePayment := func(grandTotal float64) (*PaymentRecord, error) {
		switch req.PaymentType {
		case "CASH":
			// UC 12: Оплатить наличными
			cashPay := &CashPayment{TenderedAmount: req.TenderedCash}
			if !cashPay.ExecuteTransaction(grandTotal) {
				shortage := grandTotal - req.TenderedCash
				return nil, fmt.Errorf("%w: сумма чека %.2f руб., внесено %.2f руб. (не хватает %.2f руб.)",
					ErrInsufficientCash, grandTotal, req.TenderedCash, shortage)
			}
			return &PaymentRecord{
				PaymentType:    "CASH",
				Amount:         grandTotal,
				TenderedAmount: &cashPay.TenderedAmount,
				ChangeAmount:   &cashPay.ChangeAmount,
			}, nil

		case "CARD":
			// UC 13: Оплатить банковской картой
			maskedPan := req.CardPan
			if maskedPan == "" {
				maskedPan = "**** **** **** 4242"
			}
			authCode := s.fiscal.AuthorizeCardPayment(grandTotal)
			cardPay := &CardPayment{
				CardPanMasked:     maskedPan,
				AuthorizationCode: authCode,
			}
			if !cardPay.ExecuteTransaction(grandTotal) {
				return nil, errors.New("платеж отклонен банком")
			}
			return &PaymentRecord{
				PaymentType:       "CARD",
				Amount:            grandTotal,
				CardPanMasked:     &cardPay.CardPanMasked,
				AuthorizationCode: &cardPay.AuthorizationCode,
			}, nil

		default:
			return nil, ErrInvalidPayType
		}
	}

	return s.repo.CreateSaleTransaction(ctx, req, s.fiscal, validatePayment)
}

func (s *Service) ProcessRefund(ctx context.Context, req RefundRequest) (*Receipt, string, error) {
	if req.FiscalDocNumber == "" {
		return nil, "", errors.New("укажите фискальный номер чека для возврата")
	}

	rec, err := s.repo.RefundReceiptByFiscalDoc(ctx, req.FiscalDocNumber)
	if err != nil {
		return nil, "", err
	}

	refundSlip, _ := s.fiscal.PrintRefundSlip(req.FiscalDocNumber)
	return rec, refundSlip, nil
}

func (s *Service) GetReceipts(ctx context.Context) ([]Receipt, error) {
	return s.repo.ListReceipts(ctx)
}

func (s *Service) GenerateShiftReport(ctx context.Context) (*ShiftReportDTO, error) {
	return s.repo.GetShiftReport(ctx)
}
