package territory

import (
	"fmt"
	"sync"

	"github.com/caturseptian/go-indonesia-territory/internal/dataset"
)

func buildEmbedded() (*snapshot, error) {
	fsys, err := dataset.Embedded()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDataCorrupt, err)
	}
	return buildSnapshot(fsys)
}

// loadSnapshot memuat data tertanam tepat sekali; hasil dan galatnya dipakai bersama oleh semua pemanggil.
var loadSnapshot = sync.OnceValues(buildEmbedded)

// Load memuat dan memverifikasi data tertanam sekarang (bila belum). Berguna untuk memuat saat aplikasi mulai dan
// mendapatkan galat lebih awal; fungsi lain memanggilnya otomatis.
func Load() error {
	_, err := loadSnapshot()
	return err
}

func expect(code string, want Level) error {
	c, err := Parse(code)
	if err != nil {
		return err
	}
	if c.Level != want {
		return fmt.Errorf("%w: %q adalah %s, bukan %s", ErrWrongLevel, code, c.Level, want)
	}
	return nil
}

func notFound(code string) error { return fmt.Errorf("%w: %q", ErrNotFound, code) }

// lookup dan children menerima fungsi clone secara eksplisit (method expression) agar salinan terlihat jelas di
// setiap pemanggil.
func lookup[T any](code string, level Level, idx func(*snapshot) map[string]int, items func(*snapshot) []T, clone func(T) T) (T, error) {
	var zero T
	if err := expect(code, level); err != nil {
		return zero, err
	}
	s, err := loadSnapshot()
	if err != nil {
		return zero, err
	}
	i, ok := idx(s)[code]
	if !ok {
		return zero, notFound(code)
	}
	return clone(items(s)[i]), nil
}

// ProvinceByCode mengembalikan provinsi berkode NN.
func ProvinceByCode(code string) (Province, error) {
	return lookup(code, LevelProvince, func(s *snapshot) map[string]int { return s.provinceIdx }, func(s *snapshot) []Province { return s.provinces }, Province.clone)
}

// RegencyByCode mengembalikan kabupaten/kota berkode NN.NN.
func RegencyByCode(code string) (Regency, error) {
	return lookup(code, LevelRegency, func(s *snapshot) map[string]int { return s.regencyIdx }, func(s *snapshot) []Regency { return s.regencies }, Regency.clone)
}

// DistrictByCode mengembalikan kecamatan berkode NN.NN.NN.
func DistrictByCode(code string) (District, error) {
	return lookup(code, LevelDistrict, func(s *snapshot) map[string]int { return s.districtIdx }, func(s *snapshot) []District { return s.districts }, District.clone)
}

// VillageByCode mengembalikan desa/kelurahan berkode NN.NN.NN.NNNN.
func VillageByCode(code string) (Village, error) {
	return lookup(code, LevelVillage, func(s *snapshot) map[string]int { return s.villageIdx }, func(s *snapshot) []Village { return s.villages }, Village.clone)
}

// IslandByCode mengembalikan pulau berkode NN.NN.4NNNN.
func IslandByCode(code string) (Island, error) {
	if !matchPattern(code, islandPattern) {
		return Island{}, fmt.Errorf("%w: %q bukan kode pulau", ErrInvalidCode, code)
	}
	s, err := loadSnapshot()
	if err != nil {
		return Island{}, err
	}
	i, ok := s.islandIdx[code]
	if !ok {
		return Island{}, notFound(code)
	}
	return s.islands[i].clone(), nil
}

// Provinces mengembalikan semua provinsi, urut kode.
func Provinces() ([]Province, error) {
	s, err := loadSnapshot()
	if err != nil {
		return nil, err
	}
	return cloneAll(s.provinces, Province.clone), nil
}

func children[T any](parent string, parentLevel Level, parentIdx func(*snapshot) map[string]int, items func(*snapshot) []T, code func(T) string, clone func(T) T) ([]T, error) {
	if err := expect(parent, parentLevel); err != nil {
		return nil, err
	}
	s, err := loadSnapshot()
	if err != nil {
		return nil, err
	}
	if _, ok := parentIdx(s)[parent]; !ok {
		return nil, notFound(parent)
	}
	all := items(s)
	lo, hi := childRange(func(i int) string { return code(all[i]) }, len(all), parent)
	return cloneAll(all[lo:hi], clone), nil
}

// RegenciesOf mengembalikan kabupaten/kota dalam provinsi, urut kode.
func RegenciesOf(provinceCode string) ([]Regency, error) {
	return children(provinceCode, LevelProvince, func(s *snapshot) map[string]int { return s.provinceIdx },
		func(s *snapshot) []Regency { return s.regencies }, func(r Regency) string { return r.Code }, Regency.clone)
}

// DistrictsOf mengembalikan kecamatan dalam kabupaten/kota, urut kode.
func DistrictsOf(regencyCode string) ([]District, error) {
	return children(regencyCode, LevelRegency, func(s *snapshot) map[string]int { return s.regencyIdx },
		func(s *snapshot) []District { return s.districts }, func(d District) string { return d.Code }, District.clone)
}

// VillagesOf mengembalikan desa/kelurahan dalam kecamatan, urut kode.
func VillagesOf(districtCode string) ([]Village, error) {
	return children(districtCode, LevelDistrict, func(s *snapshot) map[string]int { return s.districtIdx },
		func(s *snapshot) []Village { return s.villages }, func(v Village) string { return v.Code }, Village.clone)
}

// IslandsOf mengembalikan pulau dalam provinsi (termasuk yang tanpa kabupaten/kota) atau dalam kabupaten/kota, urut kode.
func IslandsOf(code string) ([]Island, error) {
	c, err := Parse(code)
	if err != nil {
		return nil, err
	}
	if c.Level != LevelProvince && c.Level != LevelRegency {
		return nil, fmt.Errorf("%w: %q adalah %s, bukan provinsi atau kabupaten/kota", ErrWrongLevel, code, c.Level)
	}
	s, err := loadSnapshot()
	if err != nil {
		return nil, err
	}
	idx := s.provinceIdx
	if c.Level == LevelRegency {
		idx = s.regencyIdx
	}
	if _, ok := idx[code]; !ok {
		return nil, notFound(code)
	}
	out := []Island{}
	for _, isl := range s.islands {
		if (c.Level == LevelProvince && isl.ProvinceCode == code) || (c.Level == LevelRegency && isl.RegencyCode == code) {
			out = append(out, isl.clone())
		}
	}
	return out, nil
}

// Path adalah rantai wilayah dari provinsi sampai kode yang diminta; tingkat di bawah kode itu nil.
type Path struct {
	Province Province
	Regency  *Regency
	District *District
	Village  *Village
}

// PathOf mengembalikan rantai provinsi → kabupaten/kota → kecamatan → desa untuk kode di tingkat mana pun.
func PathOf(code string) (Path, error) {
	c, err := Parse(code)
	if err != nil {
		return Path{}, err
	}
	var p Path
	if p.Province, err = ProvinceByCode(code[:2]); err != nil {
		return Path{}, err
	}
	if c.Level == LevelProvince {
		return p, nil
	}
	r, err := RegencyByCode(code[:5])
	if err != nil {
		return Path{}, err
	}
	p.Regency = &r
	if c.Level == LevelRegency {
		return p, nil
	}
	d, err := DistrictByCode(code[:8])
	if err != nil {
		return Path{}, err
	}
	p.District = &d
	if c.Level == LevelDistrict {
		return p, nil
	}
	v, err := VillageByCode(code)
	if err != nil {
		return Path{}, err
	}
	p.Village = &v
	return p, nil
}

// CodeChanges mengembalikan riwayat perubahan untuk kode (boleh kode yang sudah dihapus), urut seperti di data.
func CodeChanges(code string) ([]CodeChange, error) {
	if _, err := Parse(code); err != nil {
		return nil, err
	}
	s, err := loadSnapshot()
	if err != nil {
		return nil, err
	}
	idx := s.changesByCode[code]
	out := make([]CodeChange, len(idx))
	for i, j := range idx {
		out[i] = s.changes[j]
	}
	return out, nil
}

// Version menjelaskan dataset tertanam.
func Version() (VersionInfo, error) {
	s, err := loadSnapshot()
	if err != nil {
		return VersionInfo{}, err
	}
	return s.version, nil
}
