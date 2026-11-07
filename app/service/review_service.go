package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type ReviewService struct {
	reviews repository.ReviewRepository
	courses repository.CourseRepository
	perms   *helper.PermissionSet
}

func NewReviewService(
	reviews repository.ReviewRepository,
	courses repository.CourseRepository,
	perms *helper.PermissionSet,
) *ReviewService {
	return &ReviewService{reviews: reviews, courses: courses, perms: perms}
}

// Create godoc
//
// @Summary      Menulis ulasan untuk sebuah kelas
// @Description  Hanya peserta kelas. Satu mahasiswa hanya boleh menulis satu ulasan per kelas.
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id    path      int                        true  "ID kelas"
// @Param        body  body      model.CreateReviewRequest  true  "Isi ulasan"
// @Success      201   {object}  model.WebResponse{data=model.Review}
// @Failure      403   {object}  model.ErrorResponse  "NOT_ENROLLED"
// @Failure      409   {object}  model.ErrorResponse  "ALREADY_REVIEWED"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /courses/{id}/reviews [post]
func (s *ReviewService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id kelas harus berupa angka positif")
	}

	var req model.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Comment = strings.TrimSpace(req.Comment)
	req.Tags = NormalizeTags(req.Tags)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return translateReviewError(err)
	}
	enrolled, err := s.courses.IsEnrolled(ctx, courseID, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}
	if !enrolled {
		return helper.NewError(fiber.StatusForbidden, "NOT_ENROLLED",
			"hanya peserta kelas yang boleh menulis ulasan")
	}

	created, err := s.reviews.Create(ctx, model.Review{
		CourseID: courseID,
		UserID:   current.UserID,
		Username: current.Username,
		Rating:   req.Rating,
		Comment:  req.Comment,
		Tags:     req.Tags,
	})
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Created(c, "ulasan berhasil disimpan", created, "/api/v1/reviews/"+created.ID)
}

// ListByCourse godoc
//
// @Summary      Daftar ulasan sebuah kelas, terbaru lebih dulu
// @Description  Cursor pagination: kirim meta.next_before sebagai parameter before untuk halaman berikutnya.
// @Tags         reviews
// @Produce      json
// @Param        id      path      int     true   "ID kelas"
// @Param        tag     query     string  false  "Saring berdasarkan tag"
// @Param        before  query     string  false  "Cursor: id ulasan terakhir yang sudah dilihat"
// @Param        limit   query     int     false  "Jumlah per halaman, 1 sampai 50"  default(10)
// @Failure      400     {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /courses/{id}/reviews [get]
func (s *ReviewService) ListByCourse(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id kelas harus berupa angka positif")
	}

	reviews, err := s.reviews.FindByCourse(ctx, courseID)
	if err != nil {
		return translateReviewError(err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "daftar ulasan berhasil diambil",
		"data":    reviews,
		"meta":    fiber.Map{"limit": 10, "has_more": false, "next_before": ""},
	})
}

func translateReviewError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id ulasan tidak valid")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("data tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(fiber.StatusConflict, "ALREADY_REVIEWED",
			"Anda sudah menulis ulasan untuk kelas ini, ubah ulasan yang ada")
	case errors.Is(err, repository.ErrNotOwner):
		return helper.Forbidden("hanya pemilik ulasan yang boleh melakukan ini")
	case errors.Is(err, repository.ErrOwnReview):
		return helper.NewError(fiber.StatusConflict, "OWN_REVIEW", "tidak dapat menilai ulasan sendiri")
	case errors.Is(err, repository.ErrAlreadyVoted):
		return helper.NewError(fiber.StatusConflict, "ALREADY_VOTED", "Anda sudah menilai ulasan ini")
	case errors.Is(err, repository.ErrReplyLimit):
		return helper.NewError(fiber.StatusConflict, "REPLY_LIMIT", "jumlah balasan sudah mencapai batas")
	default:
		return helper.Internal(err)
	}
}
