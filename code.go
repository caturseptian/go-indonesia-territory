package territory

import "fmt"

// Level adalah tingkat wilayah administrasi.
type Level string

// Tingkat wilayah, dari yang tertinggi.
const (
	LevelProvince Level = "province" // kode 2 digit, mis. "31"
	LevelRegency  Level = "regency"  // kode 5 karakter, mis. "31.71"
	LevelDistrict Level = "district" // kode 8 karakter, mis. "31.71.01"
	LevelVillage  Level = "village"  // kode 13 karakter, mis. "31.71.01.1001"
)

// Code adalah kode wilayah yang formatnya sah. Parent kosong untuk provinsi.
type Code struct {
	Value  string
	Level  Level
	Parent string
}

type levelRule struct {
	level   Level
	pattern string // N = digit ASCII, karakter lain harus sama persis
}

// levelRules: aturan format per tingkat (Permendagri 58/2021), dari induk ke anak.
var levelRules = [...]levelRule{
	{level: LevelProvince, pattern: "NN"},
	{level: LevelRegency, pattern: "NN.NN"},
	{level: LevelDistrict, pattern: "NN.NN.NN"},
	{level: LevelVillage, pattern: "NN.NN.NN.NNNN"},
}

const islandPattern = "NN.NN.4NNNN"

// Parse memeriksa format kode dan menentukan tingkat serta kode induknya, tanpa memeriksa keberadaan di dataset.
// Parse hanya mengenal empat tingkat administratif (provinsi, kabupaten/kota, kecamatan, desa/kelurahan). Kode
// pulau (NN.NN.4NNNN, mis. "11.01.40001") bukan tingkat administratif sehingga ditolak dengan ErrInvalidCode;
// pakai IslandByCode untuk kode pulau.
func Parse(code string) (Code, error) {
	for i, r := range levelRules {
		if !matchPattern(code, r.pattern) {
			continue
		}
		c := Code{Value: code, Level: r.level}
		if i > 0 {
			c.Parent = code[:len(levelRules[i-1].pattern)]
		}
		return c, nil
	}
	return Code{}, fmt.Errorf("%w: %q", ErrInvalidCode, code)
}

func levelPattern(l Level) string {
	for _, r := range levelRules {
		if r.level == l {
			return r.pattern
		}
	}
	return ""
}

func matchPattern(s, pattern string) bool {
	if len(s) != len(pattern) {
		return false
	}
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == 'N' {
			if s[i] < '0' || s[i] > '9' {
				return false
			}
		} else if s[i] != pattern[i] {
			return false
		}
	}
	return true
}
