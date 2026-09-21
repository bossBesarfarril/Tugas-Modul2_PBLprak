package helper

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// EncodeCursor menggabungkan waktu dan ID menjadi string base64 yang tidak bisa
// diubah sembarangan oleh client.
func EncodeCursor(t time.Time, id int) string {
	raw := fmt.Sprintf("%d,%d", t.UnixMilli(), id)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor mengembalikan waktu dan ID dari string base64.
func DecodeCursor(encoded string) (time.Time, int, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return time.Time{}, 0, err
	}

	parts := strings.SplitN(string(decoded), ",", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("format cursor tidak valid")
	}

	milli, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, 0, err
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, 0, err
	}

	return time.UnixMilli(milli), id, nil
}
