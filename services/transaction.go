package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"CommandHandler/types"
	"CommandHandler/utils"
)

// Service menyimpan koneksi DB (pool).
type Service struct{ DB *sql.DB }

func New(db *sql.DB) *Service { return &Service{DB: db} }

// RepairPaymentMethod: strict; jika langkah penting gagal → return error (dispatcher mark FAILED)
func (s *Service) RepairPaymentMethod(ctx context.Context, p types.PayloadRepairPayment) (types.ResponseRepairPayment, error) {
	// --- Validasi dasar ---
	id := strings.TrimSpace(p.IDTRSalesHeader)
	gt := strings.TrimSpace(p.GrandTotal)
	if id == "" || gt == "" {
		return types.ResponseRepairPayment{}, fmt.Errorf("missing ID_TR_SALES_HEADER or grandTotal")
	}
	if len(id) < 12 {
		return types.ResponseRepairPayment{}, fmt.Errorf("ID_TR_SALES_HEADER must be >= 12 chars")
	}
	grandInt, convErr := strconv.Atoi(gt)
	if convErr != nil {
		return types.ResponseRepairPayment{}, fmt.Errorf("grandTotal must be an integer string: %v", convErr)
	}

	// Normalisasi tipe bayar (key → value); kalau sudah value biarkan
	fromType := string(p.FromPaymentType)
	toType := string(p.ToPaymentType)
	if v, err := utils.GetPaymentValue(fromType); err == nil && v != "" {
		fromType = v
	}
	if v, err := utils.GetPaymentValue(toType); err == nil && v != "" {
		toType = v
	}
	fromType = strings.TrimSpace(fromType)
	toType = strings.TrimSpace(toType)

	left6 := id[:6]
	right6 := id[len(id)-6:]

	// default return fields
	logCashdrawer := ""

	// --- Transaksi ---
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return types.ResponseRepairPayment{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // aman walau sudah commit

	// (1) Cari billcode di header (cast Grand_Total → INT supaya aman jika DECIMAL)
	var billcode string
	var orderOnline sql.NullString
	const q1 = `
SELECT TOP 1 ID_TR_SALES_HEADER, order_online
FROM TR_SALES_HEADER
WHERE LEFT(LTRIM(RTRIM(ID_TR_SALES_HEADER)), 6) = @left
  AND RIGHT(LTRIM(RTRIM(ID_TR_SALES_HEADER)), 6) = @right
  AND CAST(Grand_Total AS INT) = @grandTotal
`
	if scanErr := tx.QueryRowContext(
		ctx, q1,
		sql.Named("left", left6),
		sql.Named("right", right6),
		sql.Named("grandTotal", grandInt),
	).Scan(&billcode, &orderOnline); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return types.ResponseRepairPayment{}, fmt.Errorf("billcode not found: left=%s right=%s grandTotal=%d", left6, right6, grandInt)
		}
		return types.ResponseRepairPayment{}, fmt.Errorf("query header failed: %w", scanErr)
	}

	// (2) Update order_online
	const q2 = `
UPDATE TR_SALES_HEADER
SET order_online = @directSelling 
WHERE ID_TR_SALES_HEADER = @billcode
`
	if _, err = tx.ExecContext(
		ctx, q2,
		sql.Named("billcode", billcode),
		sql.Named("directSelling", utils.IsDirectSelling(p.DirectSelling)),
	); err != nil {
		return types.ResponseRepairPayment{}, fmt.Errorf("update order_online failed: %w", err)
	}

	// (3) Ambil detail pembayaran (paksa CI collation) + CAST BAYAR ke INT
	var waktuUrut time.Time
	var bayar int
	const q3 = `
SELECT TOP 1 
  WAKTU_URUT, 
  CAST(BAYAR AS INT) AS BAYAR
FROM TR_SALES_PAYMENT_DETAIL
WHERE ID_TR_SALES_HEADER = @billcode
  AND LTRIM(RTRIM(TIPE_BAYAR)) COLLATE SQL_Latin1_General_CP1_CI_AS
      = LTRIM(RTRIM(@fromType))   COLLATE SQL_Latin1_General_CP1_CI_AS
`
	if scanErr := tx.QueryRowContext(
		ctx, q3,
		sql.Named("billcode", billcode),
		sql.Named("fromType", fromType),
	).Scan(&waktuUrut, &bayar); scanErr != nil {

		if errors.Is(scanErr, sql.ErrNoRows) {
			// Diagnostik: list tipe bayar yang tersedia pada billcode ini
			rows, _ := tx.QueryContext(ctx, `
				SELECT DISTINCT LTRIM(RTRIM(TIPE_BAYAR)) AS tipe, COUNT(*) cnt
				FROM TR_SALES_PAYMENT_DETAIL
				WHERE ID_TR_SALES_HEADER = @billcode
				GROUP BY LTRIM(RTRIM(TIPE_BAYAR))
			`, sql.Named("billcode", billcode))
			var got []string
			for rows != nil && rows.Next() {
				var tipe string
				var cnt int
				_ = rows.Scan(&tipe, &cnt)
				got = append(got, fmt.Sprintf("%s(%d)", strings.ToUpper(strings.TrimSpace(tipe)), cnt))
			}
			if rows != nil {
				rows.Close()
			}
			return types.ResponseRepairPayment{}, fmt.Errorf(
				"payment detail not found: billcode=%s fromType=%s available=%v",
				billcode, fromType, got,
			)
		}
		return types.ResponseRepairPayment{}, fmt.Errorf("query payment detail failed: %w", scanErr)
	}

	// set date-only agar cocok DATEDIFF(day, ...)
	dateOnly := time.Date(waktuUrut.Year(), waktuUrut.Month(), waktuUrut.Day(), 0, 0, 0, 0, waktuUrut.Location())

	// (4) Update payment type (pakai collation CI yang sama)
	const q4 = `
UPDATE TR_SALES_PAYMENT_DETAIL
SET TIPE_BAYAR = @toType
WHERE ID_TR_SALES_HEADER = @billcode
  AND LTRIM(RTRIM(TIPE_BAYAR)) COLLATE SQL_Latin1_General_CP1_CI_AS
      = LTRIM(RTRIM(@fromType))   COLLATE SQL_Latin1_General_CP1_CI_AS
`
	if _, err = tx.ExecContext(
		ctx, q4,
		sql.Named("billcode", billcode),
		sql.Named("fromType", fromType),
		sql.Named("toType", toType),
	); err != nil {
		return types.ResponseRepairPayment{}, fmt.Errorf("update payment type failed: %w", err)
	}

	// (5) Update LOG_CASHDRAWER
	oldIsOnline := orderOnline.Valid && strings.TrimSpace(orderOnline.String) == "1"
	keteranganLama := "Online"
	if !oldIsOnline {
		keteranganLama = utils.CashDrawerLogDescription(fromType)
	}
	keteranganBaru := "Online"
	if !p.DirectSelling {
		keteranganBaru = utils.CashDrawerLogDescription(toType)
	}

	const q5sel = `
SELECT TOP 1 Keterangan
FROM LOG_CASHDRAWER
WHERE DATEDIFF(day, Tanggal, @date) = 0
  AND CashIn = @bayar
  AND LTRIM(RTRIM(Keterangan)) = @keteranganLama
`
	var exists string
	if scanErr := tx.QueryRowContext(
		ctx, q5sel,
		sql.Named("date", dateOnly),
		sql.Named("bayar", bayar),
		sql.Named("keteranganLama", strings.TrimSpace(keteranganLama)),
	).Scan(&exists); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return types.ResponseRepairPayment{}, fmt.Errorf(
				"log cashdrawer not found: date=%s bayar=%d keteranganLama=%s (oldIsOnline=%v)",
				dateOnly.Format("2006-01-02"), bayar, keteranganLama, oldIsOnline,
			)
		}
		return types.ResponseRepairPayment{}, fmt.Errorf("query log cashdrawer failed: %w", scanErr)
	}

	const q5upd = `
WITH TargetRow AS (
  SELECT TOP 1 *
  FROM LOG_CASHDRAWER
  WHERE DATEDIFF(day, Tanggal, @date) = 0
    AND CashIn = @bayar
    AND LTRIM(RTRIM(Keterangan)) = @keteranganLama
)
UPDATE TargetRow
SET Keterangan = @keteranganBaru;
`
	if _, err = tx.ExecContext(
		ctx, q5upd,
		sql.Named("date", dateOnly),
		sql.Named("bayar", bayar),
		sql.Named("keteranganLama", strings.TrimSpace(keteranganLama)),
		sql.Named("keteranganBaru", strings.TrimSpace(keteranganBaru)),
	); err != nil {
		return types.ResponseRepairPayment{}, fmt.Errorf("update LOG_CASHDRAWER failed: %w", err)
	}
	logCashdrawer = keteranganBaru

	// (6) Reset status_kirim
	const q6 = `
UPDATE TR_SALES_HEADER
SET status_kirim = '0'
WHERE ID_TR_SALES_HEADER = @billcode
`
	if _, err = tx.ExecContext(ctx, q6, sql.Named("billcode", billcode)); err != nil {
		return types.ResponseRepairPayment{}, fmt.Errorf("reset status_kirim failed: %w", err)
	}

	// Commit
	if err = tx.Commit(); err != nil {
		return types.ResponseRepairPayment{}, fmt.Errorf("commit failed: %w", err)
	}

	return types.ResponseRepairPayment{
		TipeBayar:     toType,
		LogCashdrawer: logCashdrawer,
	}, nil
}
