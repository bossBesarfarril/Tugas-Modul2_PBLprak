package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type CourseService struct {
	courses repository.CourseRepository
	outbox  repository.OutboxRepository
	tx      *repository.TxManager
}

func NewCourseService(
	courses repository.CourseRepository,
	outbox repository.OutboxRepository,
	tx *repository.TxManager,
) *CourseService {
	return &CourseService{courses: courses, outbox: outbox, tx: tx}
}

// List godoc
//
// @Summary   Daftar seluruh kelas
// @Tags      courses
// @Produce   json
// @Success   200  {object}  model.WebResponse{data=[]model.Course}
// @Failure   401  {object}  model.ErrorResponse
// @Security  BearerAuth
// @Router    /courses [get]
func (s *CourseService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courses, err := s.courses.FindAll(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar kelas berhasil diambil", courses)
}

// Mine godoc
//
// @Summary   KRS milik mahasiswa yang sedang login
// @Tags      courses
// @Produce   json
// @Success   200  {object}  model.WebResponse{data=model.MyCourses}
// @Failure   401  {object}  model.ErrorResponse
// @Security  BearerAuth
// @Router    /courses/mine [get]
func (s *CourseService) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courses, err := s.courses.FindByStudent(ctx, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}

	var totalCredits int
	for _, course := range courses {
		totalCredits += course.Credits
	}

	return helper.Success(c, fiber.StatusOK, "KRS mahasiswa berhasil diambil", model.MyCourses{
		Courses:      courses,
		TotalCredits: totalCredits,
		MaxCredits:   MaxCreditsPerSemester,
	})
}

// Get godoc
//
// @Summary   Melihat detail satu kelas
// @Tags      courses
// @Produce   json
// @Param     id   path      int  true  "ID kelas"
// @Success   200  {object}  model.WebResponse{data=model.Course}
// @Failure   401  {object}  model.ErrorResponse
// @Failure   404  {object}  model.ErrorResponse
// @Security  BearerAuth
// @Router    /courses/{id} [get]
func (s *CourseService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	course, err := s.courses.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return helper.NotFound("kelas tidak ditemukan")
	}
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "data kelas berhasil diambil", course)
}

// Enroll godoc
//
// @Summary      Mendaftar ke sebuah kelas
// @Description  Mengambil satu kursi dan mencatat event enrollment.created ke outbox dalam SATU transaction.
// @Description  Kode error 409: COURSE_FULL, ALREADY_ENROLLED, CREDIT_LIMIT_EXCEEDED.
// @Tags         courses
// @Produce      json
// @Param        id   path      int  true  "ID kelas"
// @Success      201  {object}  model.WebResponse{data=model.Course}
// @Header       201  {string}  Location  "URL kelas yang baru didaftari"
// @Failure      401  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse  "Kuota penuh, sudah terdaftar, atau batas SKS terlampaui"
// @Security     BearerAuth
// @Router       /courses/{id}/enroll [post]
func (s *CourseService) Enroll(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var enrolled model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)
		outboxRepo := s.outbox.WithTx(tx)

		// 1. Kunci KRS milik mahasiswa ini (Pessimistic locking) agar pendaftaran serentak antri tertib
		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			return err
		}

		// 2. Periksa batas SKS mahasiswa
		targetCourse, err := repo.FindByID(ctx, courseID)
		if err != nil {
			return err
		}
		taken, err := repo.SumCredits(ctx, current.UserID)
		if err != nil {
			return err
		}
		if !WithinCreditLimit(taken, targetCourse.Credits) {
			return repository.ErrCreditLimitExceeded
		}

		// 3. Kunci baris kelas yang dituju
		if err := repo.LockCourses(ctx, courseID); err != nil {
			return err
		}

		// 4. Tambah peserta secara atomik
		var errInc error
		enrolled, errInc = repo.IncrementEnrolled(ctx, courseID)
		if errInc != nil {
			return errInc
		}

		// 5. Catat pendaftarannya. Bila sudah terdaftar, error membatalkan langkah sebelumnya.
		if err := repo.InsertEnrollment(ctx, courseID, current.UserID); err != nil {
			return err
		}

		// 6. Catat event ke outbox DI DALAM transaction yang sama.
		// Catatan D.1: Modul asli memanggil s.outbox.Insert(ctx, ...) tanpa WithTx(tx).
		// Perbaikan: outboxRepo.Insert(ctx, ...) agar terikat pada transaction yang sama.
		return outboxRepo.Insert(ctx, model.EventEnrollmentCreated, model.EnrollmentEvent{
			UserID:     current.UserID,
			CourseID:   enrolled.ID,
			CourseCode: enrolled.Code,
			CourseName: enrolled.Name,
		})
	})

	if errors.Is(err, repository.ErrNotFound) {
		return helper.NotFound("kelas tidak ditemukan")
	}
	if errors.Is(err, repository.ErrCreditLimitExceeded) {
		return helper.ConflictWithCode("CREDIT_LIMIT_EXCEEDED", "batas 24 SKS terlampaui")
	}
	if errors.Is(err, repository.ErrDuplicate) {
		return helper.ConflictWithCode("ALREADY_ENROLLED", "mahasiswa sudah terdaftar di kelas ini")
	}
	if errors.Is(err, repository.ErrCourseFull) {
		return helper.ConflictWithCode("COURSE_FULL", "kelas sudah penuh")
	}
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Created(c, "berhasil mendaftar kelas", enrolled, "/api/v1/courses/"+strconv.Itoa(courseID))
}

// Drop godoc
//
// @Summary   Keluar dari sebuah kelas
// @Tags      courses
// @Param     id   path  int  true  "ID kelas"
// @Success   204  "Berhasil keluar; tanpa body"
// @Failure   401  {object}  model.ErrorResponse
// @Failure   409  {object}  model.ErrorResponse  "NOT_ENROLLED"
// @Security  BearerAuth
// @Router    /courses/{id}/enroll [delete]
func (s *CourseService) Drop(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)
		outboxRepo := s.outbox.WithTx(tx)

		if err := repo.LockCourses(ctx, courseID); err != nil {
			return err
		}

		if err := repo.DeleteEnrollment(ctx, courseID, current.UserID); err != nil {
			return err
		}

		course, err := repo.DecrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}

		// Catatan D.1: Modul asli keliru memakai model.EventEnrollmentCreated di method Drop.
		// Perbaikan: gunakan model.EventEnrollmentDropped.
		return outboxRepo.Insert(ctx, model.EventEnrollmentDropped, model.EnrollmentEvent{
			UserID:     current.UserID,
			CourseID:   course.ID,
			CourseCode: course.Code,
			CourseName: course.Name,
		})
	})

	if errors.Is(err, repository.ErrNotFound) {
		return helper.NotFound("pendaftaran tidak ditemukan")
	}
	if err != nil {
		return helper.Internal(err)
	}

	return helper.NoContent(c)
}

// Swap godoc
//
// @Summary      Pindah kelas
// @Description  Keluar dari kelas lama dan masuk ke kelas baru sebagai satu tindakan. Bila kelas baru penuh, mahasiswa tetap di kelas lama.
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        id    path      int                      true  "ID kelas asal"
// @Param        body  body      model.SwapCourseRequest  true  "Kelas tujuan"
// @Success      200   {object}  model.WebResponse{data=model.Course}
// @Failure      400   {object}  model.ErrorResponse
// @Failure      401   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "COURSE_FULL atau ALREADY_ENROLLED"
// @Security     BearerAuth
// @Router       /courses/{id}/swap [post]
func (s *CourseService) Swap(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	fromCourseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.SwapCourseRequest
	if err := c.BodyParser(&req); err != nil || req.ToCourseID < 1 {
		return helper.BadRequest("to_course_id harus berupa angka positif")
	}

	if fromCourseID == req.ToCourseID {
		return helper.BadRequest("kelas asal dan kelas tujuan tidak boleh sama")
	}

	var joined model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)
		outboxRepo := s.outbox.WithTx(tx)

		// Cegah deadlock dengan selalu mengunci dalam urutan ID menaik
		firstID, secondID := fromCourseID, req.ToCourseID
		if firstID > secondID {
			firstID, secondID = secondID, firstID
		}
		if err := repo.LockCourses(ctx, firstID, secondID); err != nil {
			return err
		}

		from, err := repo.FindByID(ctx, fromCourseID)
		if err != nil {
			return err
		}
		to, err := repo.FindByID(ctx, req.ToCourseID)
		if err != nil {
			return err
		}

		// Hapus dari kelas lama
		if err := repo.DeleteEnrollment(ctx, fromCourseID, current.UserID); err != nil {
			return err
		}
		if _, err := repo.DecrementEnrolled(ctx, fromCourseID); err != nil {
			return err
		}

		// Tambah ke kelas baru
		joined, err = repo.IncrementEnrolled(ctx, req.ToCourseID)
		if err != nil {
			return err
		}
		if err := repo.InsertEnrollment(ctx, req.ToCourseID, current.UserID); err != nil {
			return err
		}

		// Catat satu event swapped ke outbox
		return outboxRepo.Insert(ctx, model.EventEnrollmentSwapped, model.EnrollmentEvent{
			UserID:         current.UserID,
			CourseID:       to.ID,
			CourseCode:     to.Code,
			CourseName:     to.Name,
			FromCourseCode: from.Code,
			FromCourseName: from.Name,
		})
	})

	if errors.Is(err, repository.ErrNotFound) {
		return helper.NotFound("kelas tidak ditemukan atau belum terdaftar di kelas asal")
	}
	if errors.Is(err, repository.ErrDuplicate) {
		return helper.ConflictWithCode("ALREADY_ENROLLED", "sudah terdaftar di kelas tujuan")
	}
	if errors.Is(err, repository.ErrCourseFull) {
		return helper.ConflictWithCode("COURSE_FULL", "kelas tujuan sudah penuh")
	}
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "berhasil pindah kelas", joined)
}

// UpdateQuota godoc
//
// @Summary      Mengubah kuota kelas (admin)
// @Description  Optimistic locking menggunakan version.
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        id    path      int                       true  "ID kelas"
// @Param        body  body      model.UpdateQuotaRequest  true  "Kuota baru dan version"
// @Success      200   {object}  model.WebResponse{data=model.Course}
// @Failure      400   {object}  model.ErrorResponse
// @Failure      403   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "VERSION_CONFLICT atau QUOTA_BELOW_FILLED"
// @Security     BearerAuth
// @Router       /courses/{id} [patch]
func (s *CourseService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.UpdateQuotaRequest
	if err := c.BodyParser(&req); err != nil || req.Quota < 1 || req.Version < 1 {
		return helper.BadRequest("quota dan version harus berupa angka positif")
	}

	var updated model.Course
	// D.2: Melakukan perubahan kuota dan pencatatan event course.quota_changed
	// di dalam satu transaksi database yang sama (WithTx).
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)
		outboxRepo := s.outbox.WithTx(tx)

		var err error
		updated, err = repo.UpdateQuota(ctx, id, req.Quota, req.Version)
		if err != nil {
			return err
		}

		// Catat event course.quota_changed ke outbox
		return outboxRepo.Insert(ctx, model.EventCourseQuotaChanged, model.QuotaChangedEvent{
			CourseID:      updated.ID,
			CourseCode:    updated.Code,
			CourseName:    updated.Name,
			NewQuota:      updated.Quota,
			EnrolledCount: updated.EnrolledCount,
		})
	})

	if errors.Is(err, repository.ErrNotFound) {
		return helper.NotFound("kelas tidak ditemukan")
	}
	if errors.Is(err, repository.ErrVersionConflict) {
		return helper.ConflictWithCode("VERSION_CONFLICT", "data kelas telah diubah pihak lain, muat ulang")
	}
	if errors.Is(err, repository.ErrQuotaBelowFilled) {
		return helper.ConflictWithCode("QUOTA_TOO_SMALL", "kuota baru tidak boleh lebih kecil dari jumlah mahasiswa yang sudah terdaftar")
	}
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "kuota kelas berhasil diperbarui", updated)
}
