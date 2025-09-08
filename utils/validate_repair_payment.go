package utils

import (
	"fmt"
	"strconv"
	"strings"

	"CommandHandler/types"
)

// Payload seperti di TS: field payment berisi KEY (DBCA, KQRIS, ...)
type PayloadRepairPayment struct {
	SenderNIK       string `json:"senderNik"`
	IDTRSalesHeader string `json:"ID_TR_SALES_HEADER"`
	GrandTotal      string `json:"grandTotal"`
	FromPaymentType string `json:"fromPaymentType"`
	ToPaymentType   string `json:"toPaymentType"`
	DirectSelling   bool   `json:"directSelling"`
}

func ValidateRepairPayload(p PayloadRepairPayment) error {
	if strings.TrimSpace(p.IDTRSalesHeader) == "" || strings.TrimSpace(p.GrandTotal) == "" {
		return fmt.Errorf("missing ID_TR_SALES_HEADER or grandTotal")
	}
	if _, err := strconv.Atoi(p.GrandTotal); err != nil {
		return fmt.Errorf("grandTotal must be an integer string")
	}
	if _, ok := types.PaymentKeyToValue[strings.ToUpper(p.FromPaymentType)]; !ok {
		return fmt.Errorf("invalid fromPaymentType: %s", p.FromPaymentType)
	}
	if _, ok := types.PaymentKeyToValue[strings.ToUpper(p.ToPaymentType)]; !ok {
		return fmt.Errorf("invalid toPaymentType: %s", p.ToPaymentType)
	}
	return nil
}
