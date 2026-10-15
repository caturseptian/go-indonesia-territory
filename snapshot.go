package territory

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/caturseptian/go-indonesia-territory/internal/dataset"
)

type snapshot struct {
	version       Version
	provinces     []Province
	regencies     []Regency
	districts     []District
	villages      []Village
	islands       []Island
	changes       []CodeChange
	provinceIdx   map[string]int
	regencyIdx    map[string]int
	districtIdx   map[string]int
	villageIdx    map[string]int
	islandIdx     map[string]int
	changesByCode map[string][]int
}

func corrupt(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrDataCorrupt, fmt.Sprintf(format, args...))
}

// buildSnapshot membaca dan memvalidasi seluruh data; hasilnya tidak pernah diubah lagi.
func buildSnapshot(fsys fs.FS) (*snapshot, error) {
	m, sum, err := dataset.ReadManifest(fsys)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDataCorrupt, err)
	}
	read := func(name string) ([][]string, error) {
		rows, err := dataset.ReadTable(fsys, m, name)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDataCorrupt, err)
		}
		return rows, nil
	}
	s := &snapshot{version: Version{Dataset: m.DatasetVersion, EffectiveDate: m.EffectiveDate, Source: m.Source, ManifestSHA256: sum}}

	rows, err := read("provinces")
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if err := checkCode(r[0], LevelProvince, ""); err != nil {
			return nil, err
		}
		sp, err := spatial(r[2], r[3], r[4], r[5])
		if err != nil {
			return nil, corrupt("provinsi %s: %v", r[0], err)
		}
		s.provinces = append(s.provinces, Province{Code: r[0], Name: r[1], Spatial: sp})
	}
	if s.provinceIdx, err = indexSorted(s.provinces, func(p Province) string { return p.Code }); err != nil {
		return nil, err
	}

	if rows, err = read("regencies"); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if err := checkCode(r[0], LevelRegency, r[1]); err != nil {
			return nil, err
		}
		if _, ok := s.provinceIdx[r[1]]; !ok {
			return nil, corrupt("kabupaten/kota %s: provinsi %s tidak ada", r[0], r[1])
		}
		t := RegencyType(r[2])
		if t != RegencyKabupaten && t != RegencyKota {
			return nil, corrupt("kabupaten/kota %s: tipe %q", r[0], r[2])
		}
		sp, err := spatial(r[4], r[5], r[6], r[7])
		if err != nil {
			return nil, corrupt("kabupaten/kota %s: %v", r[0], err)
		}
		s.regencies = append(s.regencies, Regency{Code: r[0], ProvinceCode: r[1], Type: t, Name: r[3], Spatial: sp})
	}
	if s.regencyIdx, err = indexSorted(s.regencies, func(r Regency) string { return r.Code }); err != nil {
		return nil, err
	}

	if rows, err = read("districts"); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if err := checkCode(r[0], LevelDistrict, r[1]); err != nil {
			return nil, err
		}
		if _, ok := s.regencyIdx[r[1]]; !ok {
			return nil, corrupt("kecamatan %s: kabupaten/kota %s tidak ada", r[0], r[1])
		}
		sp, err := spatial(r[3], r[4], r[5], r[6])
		if err != nil {
			return nil, corrupt("kecamatan %s: %v", r[0], err)
		}
		s.districts = append(s.districts, District{Code: r[0], RegencyCode: r[1], Name: r[2], Spatial: sp})
	}
	if s.districtIdx, err = indexSorted(s.districts, func(d District) string { return d.Code }); err != nil {
		return nil, err
	}

	if rows, err = read("villages"); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if err := checkCode(r[0], LevelVillage, r[1]); err != nil {
			return nil, err
		}
		if _, ok := s.districtIdx[r[1]]; !ok {
			return nil, corrupt("desa %s: kecamatan %s tidak ada", r[0], r[1])
		}
		t := VillageType(r[2])
		if t != VillageDesa && t != VillageKelurahan && t != VillageDesaAdat {
			return nil, corrupt("desa %s: tipe %q", r[0], r[2])
		}
		if r[4] != "" && !matchPattern(r[4], "NNNNN") {
			return nil, corrupt("desa %s: kode pos %q", r[0], r[4])
		}
		sp, err := spatial(r[5], r[6], r[7], r[8])
		if err != nil {
			return nil, corrupt("desa %s: %v", r[0], err)
		}
		s.villages = append(s.villages, Village{Code: r[0], DistrictCode: r[1], Type: t, Name: r[3], PostalCode: r[4], Spatial: sp})
	}
	if s.villageIdx, err = indexSorted(s.villages, func(v Village) string { return v.Code }); err != nil {
		return nil, err
	}

	if rows, err = read("islands"); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if !matchPattern(r[0], islandPattern) {
			return nil, corrupt("pulau: kode %q", r[0])
		}
		if _, ok := s.provinceIdx[r[2]]; !ok {
			return nil, corrupt("pulau %s: provinsi %s tidak ada", r[0], r[2])
		}
		if r[3] != "" {
			if _, ok := s.regencyIdx[r[3]]; !ok || !strings.HasPrefix(r[3], r[2]+".") {
				return nil, corrupt("pulau %s: kabupaten/kota %s tidak sah", r[0], r[3])
			}
		}
		center, err := coordinates(r[4], r[5])
		if err != nil {
			return nil, corrupt("pulau %s: %v", r[0], err)
		}
		area, err := optFloat(r[6])
		if err != nil {
			return nil, corrupt("pulau %s: luas: %v", r[0], err)
		}
		var inhabited *bool
		switch r[7] {
		case "":
		case "0", "1":
			v := r[7] == "1"
			inhabited = &v
		default:
			return nil, corrupt("pulau %s: inhabited %q", r[0], r[7])
		}
		s.islands = append(s.islands, Island{Code: r[0], Name: r[1], ProvinceCode: r[2], RegencyCode: r[3], Center: center, AreaKm2: area, Inhabited: inhabited})
	}
	if s.islandIdx, err = indexSorted(s.islands, func(i Island) string { return i.Code }); err != nil {
		return nil, err
	}

	if rows, err = read("code_changes"); err != nil {
		return nil, err
	}
	s.changesByCode = map[string][]int{}
	for i, r := range rows {
		lvl, kind := Level(r[2]), ChangeKind(r[3])
		if levelPattern(lvl) == "" || (kind != ChangeAdded && kind != ChangeRemoved && kind != ChangeRenamed) {
			return nil, corrupt("riwayat baris %d: tingkat %q / jenis %q", i+1, r[2], r[3])
		}
		if !matchPattern(r[4], levelPattern(lvl)) {
			return nil, corrupt("riwayat baris %d: kode %q", i+1, r[4])
		}
		s.changes = append(s.changes, CodeChange{FromDocument: r[0], ToDocument: r[1], Level: lvl, Change: kind, Code: r[4], OldName: r[5], NewName: r[6]})
		s.changesByCode[r[4]] = append(s.changesByCode[r[4]], i)
	}
	return s, nil
}

// checkCode: kode harus sah, di tingkat yang diharapkan, dan (untuk anak) berinduk parent.
func checkCode(code string, want Level, parent string) error {
	c, err := Parse(code)
	if err != nil || c.Level != want {
		return corrupt("kode %q bukan kode %s", code, want)
	}
	if c.Parent != parent {
		return corrupt("kode %q berinduk %q, bukan %q", code, c.Parent, parent)
	}
	return nil
}

// indexSorted mengurutkan in-place menurut kode, menolak kode ganda, dan mengembalikan indeks kode → posisi.
func indexSorted[T any](items []T, code func(T) string) (map[string]int, error) {
	slices.SortStableFunc(items, func(a, b T) int { return strings.Compare(code(a), code(b)) })
	idx := make(map[string]int, len(items))
	for i, it := range items {
		c := code(it)
		if _, dup := idx[c]; dup {
			return nil, corrupt("kode ganda %q", c)
		}
		idx[c] = i
	}
	return idx, nil
}

// childRange: rentang [lo, hi) anak dari parent dalam daftar terurut kode. Anak berawalan "parent." dan '/' adalah
// karakter tepat setelah '.', sehingga semua anak berada di antara "parent." dan "parent/".
func childRange(code func(int) string, n int, parent string) (int, int) {
	lo := sort.Search(n, func(i int) bool { return code(i) >= parent+"." })
	hi := sort.Search(n, func(i int) bool { return code(i) >= parent+"/" })
	return lo, hi
}

func optFloat(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func coordinates(lat, lon string) (*Coordinates, error) {
	if lat == "" && lon == "" {
		return nil, nil
	}
	la, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return nil, fmt.Errorf("lintang %q", lat)
	}
	lo, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		return nil, fmt.Errorf("bujur %q", lon)
	}
	return &Coordinates{Latitude: la, Longitude: lo}, nil
}

func spatial(lat, lon, status, bbox string) (Spatial, error) {
	center, err := coordinates(lat, lon)
	if err != nil {
		return Spatial{}, err
	}
	sp := Spatial{Center: center, Status: status}
	if bbox != "" {
		var v []float64
		if err := json.Unmarshal([]byte(bbox), &v); err != nil || len(v) != 4 {
			return Spatial{}, fmt.Errorf("bbox %q", bbox)
		}
		sp.BBox = &BBox{MinLon: v[0], MinLat: v[1], MaxLon: v[2], MaxLat: v[3]}
	}
	return sp, nil
}
