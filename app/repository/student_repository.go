package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/model"
	"api-students/helper"
)

// Sentinel error milik repository
var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
)

// StudentRepository adalah KONTRAK penyimpanan data mahasiswa.
type StudentRepository interface {
	FindAllCursor(ctx context.Context, q model.CursorQuery) ([]model.Student, bool, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	Create(ctx context.Context, s model.Student) (model.Student, error)
	Update(ctx context.Context, s model.Student) (model.Student, error)
	Delete(ctx context.Context, id int) error
	FindPrestasiByStudentID(ctx context.Context, studentID int) ([]model.Prestasi, error)
}


type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

// NewStudentRepository mengembalikan interface, bukan struct konkret
func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

func (r *studentPostgresRepository) FindAllCursor(ctx context.Context, q model.CursorQuery) ([]model.Student, bool, error) {
	fetchSize := q.Limit + 1
	args := []any{fetchSize}
	where := []string{"1=1"}

	if q.Cursor != "" {
		lastTime, lastID, err := helper.DecodeCursor(q.Cursor)
		if err != nil {
			return nil, false, err
		}
		where = append(where, "(created_at, id) < ($2, $3)")
		args = append(args, lastTime, lastID)
	}

	if q.Search != "" {
		where = append(where, "name ILIKE $"+strconv.Itoa(len(args)+1))
		args = append(args, "%"+q.Search+"%")
	}

	if q.IsActive != nil {
		where = append(where, "is_active = $"+strconv.Itoa(len(args)+1))
		args = append(args, *q.IsActive)
	}

	query := `
		SELECT id, nim, name, grade, is_active, created_at, owner_id 
		FROM students 
		WHERE ` + strings.Join(where, " AND ") + ` 
		ORDER BY created_at DESC, id DESC 
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var students []model.Student
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.CreatedAt, &s.OwnerID); err != nil {
			return nil, false, err
		}
		students = append(students, s)
	}

	hasMore := len(students) > q.Limit
	if hasMore {
		students = students[:q.Limit]
	}

	return students, hasMore, nil
}

func (r *studentPostgresRepository) FindByID(
	ctx context.Context, id int,
) (model.Student, error) {
	var s model.Student

	err := r.pool.QueryRow(ctx,
		`SELECT id, nim, name, grade, is_active, created_at, owner_id
         FROM students WHERE id = $1`, id,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.CreatedAt, &s.OwnerID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, fmt.Errorf("mengambil student: %w", err)
	}

	return s, nil
}

func (r *studentPostgresRepository) Create(
	ctx context.Context, s model.Student,
) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO students (nim, name, grade, is_active, owner_id) 
         VALUES ($1, $2, $3, $4, $5) 
         RETURNING id, created_at`,
		s.NIM, s.Name, s.Grade, s.IsActive, s.OwnerID,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("menyimpan student: %w", err)
	}

	return s, nil
}

func (r *studentPostgresRepository) Update(
	ctx context.Context, s model.Student,
) (model.Student, error) {
	err := r.pool.QueryRow(ctx,
		`UPDATE students SET nim = $1, name = $2, grade = $3, is_active = $4 
         WHERE id = $5 
         RETURNING id, nim, name, grade, is_active, created_at`,
		s.NIM, s.Name, s.Grade, s.IsActive, s.ID,
	).Scan(&s.ID, &s.NIM, &s.Name, &s.Grade, &s.IsActive, &s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, fmt.Errorf("memperbarui student: %w", err)
	}

	return s, nil
}

func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM students WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus student: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// Cek error kode UNIQUE dari Postgres (23505)
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (r *studentPostgresRepository) FindPrestasiByStudentID(
	ctx context.Context, studentID int,
) ([]model.Prestasi, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, id_mahasiswa, nama_prestasi, juara 
         FROM prestasi WHERE id_mahasiswa = $1`, studentID,
	)
	if err != nil {
		return nil, fmt.Errorf("mengambil prestasi: %w", err)
	}
	defer rows.Close()

	hasil := []model.Prestasi{}
	for rows.Next() {
		var p model.Prestasi
		if err := rows.Scan(&p.ID, &p.IDMahasiswa, &p.NamaPrestasi, &p.Juara); err != nil {
			return nil, fmt.Errorf("membaca baris prestasi: %w", err)
		}
		hasil = append(hasil, p)
	}

	return hasil, nil
}
