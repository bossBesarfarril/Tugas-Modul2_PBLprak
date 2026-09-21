package helper

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// RequestContext memberi batas waktu 5 detik untuk setiap operasi database.
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

// ParamID membaca parameter :id dari URL dan memastikan berupa angka positif.
func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// allowedSort adalah daftar putih kolom yang boleh dipakai untuk mengurutkan.
var allowedSort = map[string]bool{
	"id": true, "nim": true, "name": true, "grade": true,
}

// ParseCursorQuery mengambil parameter untuk Cursor Pagination.
func ParseCursorQuery(c *fiber.Ctx) model.CursorQuery {
	q := model.CursorQuery{
		Cursor: c.Query("cursor", ""),
		Limit:  c.QueryInt("limit", 10),
		Search: c.Query("search", ""),
	}

	if q.Limit > 100 {
		q.Limit = 100
	} else if q.Limit <= 0 {
		q.Limit = 10
	}

	if active := c.Query("is_active"); active != "" {
		val := (active == "true" || active == "1")
		q.IsActive = &val
	}

	return q
}
