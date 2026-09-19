package repository

import (
	"api-students/helper"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleRepository interface {
	LoadPermissions(ctx context.Context) (*helper.PermissionSet, error)
}

type roleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) LoadPermissions(ctx context.Context) (*helper.PermissionSet, error) {
	query := `
		SELECT r.name, rp.permission_name 
		FROM roles r
		LEFT JOIN role_permissions rp ON r.name = rp.role_name
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := helper.NewPermissionSet()
	for rows.Next() {
		var roleName string
		var permName *string // Pakai pointer karena hasilnya bisa bernilai NULL

		if err := rows.Scan(&roleName, &permName); err != nil {
			return nil, err
		}

		// Kalau permission-nya ada (tidak NULL), masukkan ke dalam wadah memori
		if permName != nil {
			perms.Add(roleName, *permName)
		}
	}

	return perms, rows.Err()
}
