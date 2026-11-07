package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

const refreshTokenBytes = 32

type AuthService struct {
	users      repository.UserRepository
	tokens     repository.TokenRepository
	jwt        *helper.JWTManager
	refreshTTL time.Duration
}

func NewAuthService(
	users repository.UserRepository,
	tokens repository.TokenRepository,
	jwtManager *helper.JWTManager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:      users,
		tokens:     tokens,
		jwt:        jwtManager,
		refreshTTL: refreshTTL,
	}
}

// Register godoc
//
// @Summary      Mendaftarkan akun baru
// @Description  Role akun baru selalu "user". Password minimal 8 karakter dan memuat huruf serta angka.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.RegisterRequest  true  "Data pendaftaran"
// @Success      201   {object}  model.WebResponse{data=model.AuthUser}
// @Failure      400   {object}  model.ErrorResponse  "Format body atau email tidak valid"
// @Failure      409   {object}  model.ErrorResponse  "Username atau email sudah digunakan"
// @Failure      422   {object}  model.ErrorResponse  "Validasi gagal: password tidak memenuhi syarat"
// @Router       /auth/register [post]
func (s *AuthService) Register(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Password di-hash sebelum ke database
	hashed, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(errors.New("gagal memproses password"))
	}

	// Role selalu ditentukan server ("user"), kebal dari mass assignment
	created, err := s.users.Create(ctx, model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashed,
		Role:     "user",
		IsActive: true,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Conflict("username sudah dipakai")
		}
		return helper.Internal(errors.New("gagal mendaftarkan user"))
	}

	return helper.Created(c, "pendaftaran berhasil", created,
		"/api/v1/users/"+strconv.Itoa(created.ID))
}

// Login godoc
//
// @Summary      Masuk dan mendapatkan pasangan token
// @Description  Rate limit: maksimal 5 percobaan per menit per IP. Mengembalikan access token dan refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.LoginRequest  true  "Username dan password"
// @Success      200   {object}  model.WebResponse{data=model.TokenPair}
// @Failure      401   {object}  model.ErrorResponse  "Username atau password salah"
// @Failure      429   {object}  model.ErrorResponse  "Terlalu banyak percobaan login"
// @Router       /auth/login [post]
func (s *AuthService) Login(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	user, err := s.users.FindByUsername(ctx, strings.TrimSpace(req.Username))
	if err != nil {
		// Proteksi user enumeration: jalankan hash palsu lalu beri pesan yang sama persis
		helper.VerifyDummyPassword(req.Password)
		return helper.Unauthorized("username atau password salah")
	}

	if !helper.VerifyPassword(user.Password, req.Password) {
		return helper.Unauthorized("username atau password salah")
	}

	if !user.IsActive {
		return helper.Forbidden("akun dinonaktifkan")
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return helper.Internal(errors.New("gagal membuat token"))
	}

	return helper.Success(c, fiber.StatusOK, "login berhasil", pair)
}

// Refresh godoc
//
// @Summary      Menukar refresh token dengan pasangan token baru
// @Description  Refresh token lama langsung dicabut (rotation).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.RefreshRequest  true  "Refresh token"
// @Success      200   {object}  model.WebResponse{data=model.TokenPair}
// @Failure      401   {object}  model.ErrorResponse  "Refresh token tidak valid, kedaluwarsa, atau sudah dicabut"
// @Router       /auth/refresh [post]
func (s *AuthService) Refresh(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		return helper.BadRequest("refresh_token wajib diisi")
	}

	hash := helper.SHA256Hex(req.RefreshToken)

	stored, err := s.tokens.FindActive(ctx, hash)
	if err != nil {
		return helper.Unauthorized("refresh token tidak valid atau sudah kedaluwarsa")
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil || !user.IsActive {
		return helper.Unauthorized("akun tidak dapat dipakai")
	}

	// ROTASI TOKEN: cabut token lama, ganti dengan pasangan token baru
	if err := s.tokens.Revoke(ctx, hash); err != nil {
		return helper.Internal(errors.New("gagal memperbarui token"))
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return helper.Internal(errors.New("gagal membuat token"))
	}

	return helper.Success(c, fiber.StatusOK, "token berhasil diperbarui", pair)
}

// Logout godoc
//
// @Summary      Keluar dan mencabut refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.RefreshRequest  true  "Refresh token yang ingin dicabut"
// @Success      204   "Berhasil keluar; tanpa body"
// @Failure      400   {object}  model.ErrorResponse
// @Router       /auth/logout [post]
func (s *AuthService) Logout(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if strings.TrimSpace(req.RefreshToken) != "" {
		_ = s.tokens.Revoke(ctx, helper.SHA256Hex(req.RefreshToken))
	}

	return helper.Success(c, fiber.StatusOK, "logout berhasil", nil)
}

// Me godoc
//
// @Summary      Informasi user yang sedang login
// @Tags         auth
// @Produce      json
// @Success      200  {object}  model.WebResponse{data=model.AuthUser}
// @Failure      401  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /auth/me [get]
func (s *AuthService) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	user, err := s.users.FindByID(ctx, authUser.UserID)
	if err != nil {
		return helper.Unauthorized("user tidak ditemukan")
	}

	return helper.Success(c, fiber.StatusOK, "profil berhasil diambil", user)
}

func (s *AuthService) issueTokenPair(
	ctx context.Context, user model.User,
) (model.TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccess(user)
	if err != nil {
		return model.TokenPair{}, err
	}

	rawRefreshToken, err := helper.GenerateRandomToken(refreshTokenBytes)
	if err != nil {
		return model.TokenPair{}, err
	}

	err = s.tokens.Save(ctx, model.RefreshToken{
		UserID:    user.ID,
		TokenHash: helper.SHA256Hex(rawRefreshToken),
		ExpiresAt: time.Now().Add(s.refreshTTL),
	})
	if err != nil {
		return model.TokenPair{}, err
	}

	return model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    15 * 60, // 15 menit dalam detik
	}, nil
}
