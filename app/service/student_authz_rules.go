package service

import "api-students/helper"

// CanAccessStudent adalah pengaman Level 2 (Kepemilikan Data)
func CanAccessStudent(perms *helper.PermissionSet, role string, userID int, ownerID *int, permissionAny string) bool {
	// 1. Kalau punya izin ":any" (admin/staff), langsung diizinkan buka punya siapa saja
	if perms.Can(role, permissionAny) {
		return true
	}

	// 2. Kalau data ini buatan dia sendiri (owner), diizinkan
	if ownerID != nil && *ownerID == userID {
		return true
	}

	// 3. Sisanya otomatis ditolak
	return false
}
