package types

type PayloadRepairPayment struct {
	SenderNIK       string `json:"senderNik"` // wajib
	IDTRSalesHeader string `json:"ID_TR_SALES_HEADER"`
	GrandTotal      string `json:"grandTotal"`
	FromPaymentType string `json:"fromPaymentType"`
	ToPaymentType   string `json:"toPaymentType"`
	DirectSelling   bool   `json:"directSelling"`
}

type PayloadDeletePayment struct {
	SenderNIK       string `json:"senderNik"` // kalau di DELETE juga wajib, samakan
	IDTRSalesHeader string `json:"ID_TR_SALES_HEADER"`
}
