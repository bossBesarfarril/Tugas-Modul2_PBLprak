package helper

// PermissionSet menyimpan data hak akses tiap role
type PermissionSet struct {
	permissions map[string]map[string]struct{}
}

// NewPermissionSet membuat wadah hak akses, opsional menerima peta izin awal
func NewPermissionSet(initial ...map[string][]string) *PermissionSet {
	ps := &PermissionSet{
		permissions: make(map[string]map[string]struct{}),
	}
	for _, m := range initial {
		for role, perms := range m {
			for _, perm := range perms {
				ps.Add(role, perm)
			}
		}
	}
	return ps
}

// Add memasukkan hak akses baru ke dalam wadah role tertentu
func (p *PermissionSet) Add(role, permission string) {
	if p.permissions[role] == nil {
		p.permissions[role] = make(map[string]struct{})
	}
	p.permissions[role][permission] = struct{}{}
}

// Can bertugas mengecek apakah suatu role diizinkan melakukan suatu tindakan
func (p *PermissionSet) Can(role, permission string) bool {
	if p == nil || p.permissions == nil {
		return false
	}
	_, allowed := p.permissions[role][permission]
	return allowed
}
