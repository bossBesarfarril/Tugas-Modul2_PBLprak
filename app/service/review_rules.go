package service

import "strings"

// NormalizeTags membersihkan dan menstandarkan daftar tag ulasan:
// 1. Menghilangkan whitespace di tepi.
// 2. Mengubah semua huruf menjadi huruf kecil.
// 3. Membuang string kosong.
// 4. Menghapus duplikat sambil mempertahankan urutan kemunculan pertama.
// 5. Mengembalikan slice kosong []string{} (bukan nil) jika input kosong/nil.
func NormalizeTags(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(in))
	seen := make(map[string]bool, len(in))

	for _, tag := range in {
		cleaned := strings.ToLower(strings.TrimSpace(tag))
		if cleaned == "" {
			continue
		}
		if !seen[cleaned] {
			seen[cleaned] = true
			result = append(result, cleaned)
		}
	}

	return result
}
