package sales

import (
	"fmt"
	"math/rand"
	"time"
)

// FiscalMockService — программная эмуляция онлайн-кассы и POS-терминала
type FiscalMockService struct{}

func NewFiscalMockService() *FiscalMockService {
	return &FiscalMockService{}
}

// PrintFiscalReceipt — UC 10: Печатать фискальный чек
func (f *FiscalMockService) PrintFiscalReceipt(total float64, paymentType string) (string, error) {
	// Эмуляция задержки печати чека на кассовой ленте
	time.Sleep(50 * time.Millisecond)

	fiscalNumber := fmt.Sprintf("ФД-%s-%04d",
		time.Now().Format("20060102-150405"),
		rand.Intn(9000)+1000,
	)
	return fiscalNumber, nil
}

// PrintRefundSlip — UC 14: Печать фискального чека «Возврат прихода»
func (f *FiscalMockService) PrintRefundSlip(originalFiscalDoc string) (string, error) {
	time.Sleep(50 * time.Millisecond)
	return fmt.Sprintf("ВОЗВРАТ-%s", originalFiscalDoc), nil
}

// AuthorizeCardPayment — UC 13: Эмуляция связи с банком по POS-терминалу
func (f *FiscalMockService) AuthorizeCardPayment(amount float64) string {
	return fmt.Sprintf("AUTH-%06d", rand.Intn(900000)+100000)
}
