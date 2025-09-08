package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// GetStoreID
func GetStoreID(ctx context.Context, db *sql.DB) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var id sql.NullString
	err := db.QueryRowContext(ctx, `SELECT TOP 1 Store_ID FROM DT_STORE ORDER BY Store_ID`).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("store id not found")
		}
		return "", err
	}

	if !id.Valid || strings.TrimSpace(id.String) == "" {
		return "", errors.New("store id is empty")
	}

	return strings.TrimSpace(id.String), nil
}
