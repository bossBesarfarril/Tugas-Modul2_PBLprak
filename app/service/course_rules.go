package service

const (
	MaxCredits            = 24
	MaxCreditsPerSemester = 24
)

// WithinCreditLimit memeriksa apakah penambahan SKS masih dalam batas maksimal 24 SKS.
func WithinCreditLimit(taken, adding int) bool {
	return taken+adding <= MaxCredits
}

// CreditsAfterSwap menghitung total SKS setelah pertukaran kelas (meninggalkan satu kelas dan mengambil kelas lain).
func CreditsAfterSwap(taken, leaving, joining int) int {
	return taken - leaving + joining
}
