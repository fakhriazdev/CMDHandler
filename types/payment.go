package types

type Payment string

const (
	CASH      Payment = "Cash"
	DBCA      Payment = "D.BCA"
	DBRI      Payment = "D.BRI"
	DMandiri  Payment = "D.DMANDIRI"
	DBNI      Payment = "D.DBNI"
	CBCA      Payment = "K.BCA"
	CBRI      Payment = "K.BRI"
	CMandiri  Payment = "K.MANDIRI"
	CBNI      Payment = "K.DBNI"
	QRISBCA   Payment = "D.QRIS BCA"
	QRISBNI   Payment = "D.QRIS BNI"
	QRISMDR   Payment = "D.QRIS MDR"
	KQRIS     Payment = "K.QRIS"
	KINDODANA Payment = "K.INDODANA"
)

var PaymentKeyToValue = map[string]string{
	"CASH":      string(CASH),
	"DBCA":      string(DBCA),
	"DBRI":      string(DBRI),
	"DMANDIRI":  string(DMandiri),
	"DBNI":      string(DBNI),
	"KBCA":      string(CBCA),
	"KBRI":      string(CBRI),
	"KMANDIRI":  string(CMandiri),
	"KBNI":      string(CBNI),
	"QRISBCA":   string(QRISBCA),
	"QRISBNI":   string(QRISBNI),
	"QRISMDR":   string(QRISMDR),
	"KQRIS":     string(KQRIS),
	"KINDODANA": string(KINDODANA),
}
