package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type PortfolioService struct {
	portfolioRepo repository.PortfolioRepository
	studentRepo   repository.StudentRepository
}

func NewPortfolioService(portfolioRepo repository.PortfolioRepository, studentRepo repository.StudentRepository) *PortfolioService {
	return &PortfolioService{
		portfolioRepo: portfolioRepo,
		studentRepo:   studentRepo,
	}
}

// GET /students/:id/portfolio — membaca portofolio mahasiswa
func (s *PortfolioService) GetPortfolio(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.BadRequest("id mahasiswa harus berupa angka")
	}

	// 1. Validasi ke PostgreSQL: pastikan mahasiswa ada
	if _, err := s.studentRepo.FindByID(ctx, studentID); err != nil {
		return helper.NotFound("mahasiswa tidak ditemukan di sistem")
	}

	// 2. Ambil portofolio dari MongoDB
	portfolio, err := s.portfolioRepo.FindByStudentID(ctx, studentID)
	if err != nil {
		if errors.Is(err, repository.ErrPortfolioNotFound) {
			return helper.NotFound("portofolio mahasiswa belum dibuat")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "berhasil mengambil portofolio", portfolio)
}

// PUT /students/:id/portfolio — membuat atau mengganti portofolio mahasiswa
func (s *PortfolioService) UpsertPortfolio(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.BadRequest("id mahasiswa harus berupa angka")
	}

	// 1. Validasi ke PostgreSQL: tolak bila mahasiswa tidak ada di PostgreSQL
	if _, err := s.studentRepo.FindByID(ctx, studentID); err != nil {
		return helper.NotFound("mahasiswa tidak ditemukan di sistem")
	}

	// 2. Parse request body dan validasi
	var req model.UpsertPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// 3. Simpan / perbarui ke MongoDB (Upsert)
	portfolio, err := s.portfolioRepo.UpsertByStudentID(ctx, studentID, req)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "portofolio berhasil disimpan", portfolio)
}

// POST /students/:id/portfolio/projects — menambah satu proyek (batas 20 proyek atomik)
func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.BadRequest("id mahasiswa harus berupa angka")
	}

	// 1. Validasi ke PostgreSQL
	if _, err := s.studentRepo.FindByID(ctx, studentID); err != nil {
		return helper.NotFound("mahasiswa tidak ditemukan di sistem")
	}

	// 2. Parse request body dan validasi
	var req model.AddProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// 3. Tambahkan proyek ke MongoDB
	portfolio, err := s.portfolioRepo.AddProject(ctx, studentID, req)
	if err != nil {
		if errors.Is(err, repository.ErrProjectLimit) {
			return helper.BadRequest("jumlah proyek sudah mencapai batas maksimal 20")
		}
		if errors.Is(err, repository.ErrPortfolioNotFound) {
			return helper.NotFound("portofolio belum dibuat, silakan buat portofolio terlebih dahulu")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusCreated, "proyek berhasil ditambahkan", portfolio)
}

// DELETE /students/:id/portfolio/projects/:projectId — menghapus satu proyek
func (s *PortfolioService) DeleteProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return helper.BadRequest("id mahasiswa harus berupa angka")
	}

	projectID := c.Params("projectId")
	if projectID == "" {
		return helper.BadRequest("id proyek tidak boleh kosong")
	}

	// 1. Validasi ke PostgreSQL
	if _, err := s.studentRepo.FindByID(ctx, studentID); err != nil {
		return helper.NotFound("mahasiswa tidak ditemukan di sistem")
	}

	// 2. Hapus proyek di MongoDB
	portfolio, err := s.portfolioRepo.DeleteProject(ctx, studentID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidProjectID) {
			return helper.BadRequest("format id proyek tidak valid")
		}
		if errors.Is(err, repository.ErrPortfolioNotFound) {
			return helper.NotFound("portofolio atau proyek tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "proyek berhasil dihapus", portfolio)
}

// GET /portfolios/stats/technologies — aggregation pipeline 10 teknologi terpopuler
func (s *PortfolioService) GetTopTechnologies(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := s.portfolioRepo.GetTopTechnologies(ctx)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "berhasil mengambil 10 teknologi terpopuler", stats)
}
