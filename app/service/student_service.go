package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet // Menambahkan wadah hak akses ke dalam service
}

func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, perms: perms}
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseCursorQuery(c)

	students, hasMore, err := s.repo.FindAllCursor(ctx, q)
	if err != nil {
		return helper.Internal(errors.New("gagal mengambil data student"))
	}

	meta := &model.CursorMeta{
		HasMore: hasMore,
	}

	// Kalau masih ada halaman selanjutnya, buat NextCursor dari data terakhir
	if hasMore && len(students) > 0 {
		lastStudent := students[len(students)-1]
		meta.NextCursor = helper.EncodeCursor(lastStudent.CreatedAt, lastStudent.ID)
	}

	return helper.SuccessList(c, "daftar student berhasil diambil", students, meta)
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "student")
	}

	// Pengaman Level 2: Cek kepemilikan sebelum melihat detail
	authUser, _ := helper.CurrentUser(c)
	if !CanAccessStudent(s.perms, authUser.Role, authUser.UserID, student.OwnerID, "student:read:any") {
		return helper.Forbidden("Akses ditolak: Anda tidak berhak melihat data ini")
	}

	return helper.Success(c, fiber.StatusOK, "student ditemukan", student)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Ambil identitas pembuat data dari token JWT
	authUser, _ := helper.CurrentUser(c)

	newStudent, err := s.repo.Create(ctx, model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: true,
		OwnerID:  &authUser.UserID, // Set pemilik data secara otomatis
	})
	if err != nil {
		return translateError(err, "student")
	}

	return helper.Created(c, "student berhasil dibuat", newStudent,
		"/api/v1/students/"+strconv.Itoa(newStudent.ID))
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.NIM = strings.TrimSpace(req.NIM)
	req.Name = strings.TrimSpace(req.Name)

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "student")
	}

	// Pengaman Level 2: Cek kepemilikan sebelum mengedit
	authUser, _ := helper.CurrentUser(c)
	if !CanAccessStudent(s.perms, authUser.Role, authUser.UserID, current.OwnerID, "student:update:any") {
		return helper.Forbidden("Akses ditolak: Anda tidak berhak mengubah data ini")
	}

	result, err := s.repo.Update(ctx, model.Student{
		ID:       id,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	})
	if err != nil {
		return translateError(err, "student")
	}

	return helper.Success(c, fiber.StatusOK, "student berhasil diganti seluruhnya", result)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	if IsEmptyPatch(req) {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "student")
	}

	// Pengaman Level 2: Cek kepemilikan sebelum mengedit sebagian
	authUser, _ := helper.CurrentUser(c)
	if !CanAccessStudent(s.perms, authUser.Role, authUser.UserID, current.OwnerID, "student:update:any") {
		return helper.Forbidden("Akses ditolak: Anda tidak berhak mengubah data ini")
	}

	updated := ApplyPatch(current, req)

	result, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateError(err, "student")
	}

	return helper.Success(c, fiber.StatusOK, "student berhasil diperbarui sebagian", result)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "student")
	}

	// Pengaman Level 2: Cek kepemilikan sebelum menghapus
	authUser, _ := helper.CurrentUser(c)
	if !CanAccessStudent(s.perms, authUser.Role, authUser.UserID, current.OwnerID, "student:delete") {
		return helper.Forbidden("Akses ditolak: Anda tidak berhak menghapus data ini")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(err, "student")
	}

	return helper.NoContent(c)
}

func translateError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("NIM sudah dipakai")
	default:
		return helper.Internal(err)
	}
}

func (s *StudentService) GetPrestasi(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "student")
	}

	// Pengaman Level 2
	authUser, _ := helper.CurrentUser(c)
	if !CanAccessStudent(s.perms, authUser.Role, authUser.UserID, student.OwnerID, "student:read:any") {
		return helper.Forbidden("Akses ditolak: Anda tidak berhak melihat data ini")
	}

	prestasiList, err := s.repo.FindPrestasiByStudentID(ctx, id)
	if err != nil {
		return helper.Internal(errors.New("gagal mengambil prestasi"))
	}

	return helper.Success(c, fiber.StatusOK, "daftar prestasi mahasiswa berhasil diambil", prestasiList)
}
