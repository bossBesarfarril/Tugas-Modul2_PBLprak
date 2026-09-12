package helper

import (
	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// LocalsAuthUser adalah kunci penyimpanan identitas pemakai di context Fiber.
const LocalsAuthUser = "authUser"

// CurrentUser mengambil data user login yang disimpan oleh middleware RequireAuth.
func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(LocalsAuthUser).(model.AuthUser)
	return user, ok
}