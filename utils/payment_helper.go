package utils

import (
	"fmt"
	"strings"

	"CommandHandler/types"
)

func IsDirectSelling(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func CashDrawerLogDescription(value string) string {
	if strings.HasPrefix(value, "D.QRIS ") {
		bank := strings.TrimPrefix(value, "D.QRIS ")
		return "QRIS " + bank
	}
	if value == "K.QRIS" {
		return "QRIS"
	}
	if value == "K.INDODANA" {
		return "INDODANA"
	}
	if strings.HasPrefix(value, "D.") {
		return "Debet Card"
	}
	if strings.HasPrefix(value, "K.") {
		return "Credit Card"
	}
	return "Cash"
}

// GetPaymentValue menerima "key" (DBCA, KQRIS, ...) ATAU "value" (D.BCA, K.QRIS, ...)
// dan mengembalikan value DB final.
func GetPaymentValue(in string) (string, error) {
	s := strings.TrimSpace(in)
	if s == "" {
		return "", fmt.Errorf("empty payment type")
	}

	// Sudah format nilai DB yang valid? kembalikan apa adanya (trim)
	if strings.HasPrefix(s, "D.") || strings.HasPrefix(s, "K.") ||
		s == "Cash" || strings.HasPrefix(s, "D.QRIS ") || s == "K.QRIS" {
		return strings.TrimSpace(s), nil
	}

	// Coba mapping KEY → VALUE (case/space-insensitive)
	key := strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	if v, ok := types.PaymentKeyToValue[key]; ok {
		return v, nil
	}

	// Fallback generik: Dxxxx → D.xxxx, Kxxxx → K.xxxx, DQRISXXX → D.QRIS XXX
	if len(key) > 1 && (key[0] == 'D' || key[0] == 'K') {
		// DQRISBCA → D.QRIS BCA
		if strings.HasPrefix(key, "DQRIS") && len(key) > 5 {
			bank := strings.TrimSpace(key[5:])
			if bank != "" {
				return "D.QRIS " + bank, nil
			}
		}
		// DBRI → D.BRI, KBRI → K.BRI
		return string(key[0]) + "." + key[1:], nil
	}

	if key == "CASH" {
		return "Cash", nil
	}
	if key == "KQRIS" {
		return "K.QRIS", nil
	}

	return "", fmt.Errorf("invalid payment type: %s", in)
}
