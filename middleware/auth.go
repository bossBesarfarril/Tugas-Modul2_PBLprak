package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"api-students/helper"
)

// RequireAuth memeriksa access token pada header Authorization.
// Endpoint yang dipasangi middleware ini WAJIB membawa token yang sah.
func RequireAuth(jwtManager *helper.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, err := bearerToken(c)
		if err != nil {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)
			return helper.Unauthorized("header Authorization tidak ada atau salah bentuk")
		}

		authUser, err := jwtManager.Parse(token)
		if err != nil {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)

			// Membedakan "kedaluwarsa" dari "tidak valid" aman dilakukan:
			// client memang perlu tahu kapan harus memanggil /auth/refresh.
			if errors.Is(err, helper.ErrExpiredToken) {
				return helper.Unauthorized("access token kedaluwarsa")
			}
			return helper.Unauthorized("access token tidak valid")
		}

		// Simpan identitas user ke context locals
		c.Locals(helper.LocalsAuthUser, authUser)
		return c.Next()
	}
}

func bearerToken(c *fiber.Ctx) (string, error) {
	header := c.Get(fiber.HeaderAuthorization)
	if header == "" {
		return "", errors.New("header kosong")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("format bukan Bearer")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("token kosong")
	}

	return token, nil
}

// LoginRateLimiter membatasi jumlah percobaan login dari satu alamat IP (maks 5x per menit).
// Menjawab 429 Too Many Requests disertai header Retry-After bila batas terlampaui.
func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", "60")
			return helper.TooManyRequests("terlalu banyak percobaan login, coba lagi dalam satu menit")
		},
	})
}

// RequirePermission adalah pengaman Level 1 untuk mencegat user yang tidak punya hak akses
func RequirePermission(perms *helper.PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Ambil data user dari token JWT yang sedang login
		authUser, ok := helper.CurrentUser(c)
		if !ok {
			return helper.Unauthorized("Anda harus login terlebih dahulu")
		}

		// Cek apakah role dari user tersebut punya izin (permission) yang diminta
		if !perms.Can(authUser.Role, permission) {
			return helper.Forbidden("Akses ditolak: role tidak memiliki hak akses ini")
		}

		// Kalau punya izin, silakan lanjut ke proses berikutnya
		return c.Next()
	}
}
