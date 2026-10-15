package territory

import (
	"errors"
	"testing"

	"github.com/caturseptian/go-indonesia-territory/internal/datasettest"
)

func validTables() map[string][][]string {
	return map[string][][]string{
		"provinces": {{"31", "DKI Jakarta", "-6.2", "106.8", "available", "[106.6,-6.4,106.9,-6.0]"}, {"11", "Aceh", "", "", "not_available", ""}},
		"regencies": {{"31.71", "31", "kota", "Kota Jakarta Pusat", "", "", "not_available", ""}},
		"districts": {{"31.71.01", "31.71", "Gambir", "", "", "not_available", ""}},
		"villages": {{"31.71.01.1001", "31.71.01", "kelurahan", "Gambir", "10110", "-6.17", "106.82", "available", "[106.8,-6.2,106.9,-6.1]"},
			{"31.71.01.1002", "31.71.01", "desa_adat", "Kampung O'Neil", "", "", "", "not_available", ""}},
		"islands":      {{"31.01.40001", "Pulau Uji", "31", "", "-5.6", "106.5", "0.25", "1"}},
		"code_changes": {{"A", "B", "village", "renamed", "31.71.01.1001", "Lama", "Gambir"}},
	}
}

func TestBuildSnapshotSortsAndIndexes(t *testing.T) {
	s, err := buildSnapshot(datasettest.Build(t, validTables()))
	if err != nil {
		t.Fatal(err)
	}
	if s.provinces[0].Code != "11" || s.provinces[1].Code != "31" {
		t.Fatalf("provinsi tidak urut: %+v", s.provinces)
	}
	v := s.villages[s.villageIdx["31.71.01.1001"]]
	if v.PostalCode != "10110" || v.Type != VillageKelurahan || v.Spatial.Center == nil || *v.Spatial.BBox != (BBox{106.8, -6.2, 106.9, -6.1}) {
		t.Fatalf("desa = %+v", v)
	}
	if s.villages[s.villageIdx["31.71.01.1002"]].Spatial.Center != nil {
		t.Fatal("koordinat kosong harus nil")
	}
	isl := s.islands[s.islandIdx["31.01.40001"]]
	if isl.RegencyCode != "" || isl.AreaKm2 == nil || *isl.AreaKm2 != 0.25 || isl.Inhabited == nil || !*isl.Inhabited {
		t.Fatalf("pulau = %+v", isl)
	}
	if got := s.changesByCode["31.71.01.1001"]; len(got) != 1 {
		t.Fatalf("riwayat = %v", got)
	}
}

func TestBuildSnapshotRejectsInconsistentData(t *testing.T) {
	cases := map[string]func(map[string][][]string){
		"induk tidak ada":       func(m map[string][][]string) { m["regencies"][0][1] = "99" },
		"induk bukan awalan":    func(m map[string][][]string) { m["districts"][0][1] = "31.72" },
		"format kode salah":     func(m map[string][][]string) { m["provinces"][0][0] = "3 1" },
		"tingkat kode salah":    func(m map[string][][]string) { m["districts"][0][0] = "31.71" },
		"tipe tidak dikenal":    func(m map[string][][]string) { m["villages"][0][2] = "kampung" },
		"angka rusak":           func(m map[string][][]string) { m["villages"][0][5] = "abc" },
		"bbox rusak":            func(m map[string][][]string) { m["provinces"][0][5] = "[1,2]" },
		"kode ganda":            func(m map[string][][]string) { m["provinces"][1][0] = "31" },
		"pulau tanpa provinsi":  func(m map[string][][]string) { m["islands"][0][2] = "99" },
		"inhabited tidak sah":   func(m map[string][][]string) { m["islands"][0][7] = "ya" },
		"jenis perubahan salah": func(m map[string][][]string) { m["code_changes"][0][3] = "moved" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			tables := validTables()
			mutate(tables)
			if _, err := buildSnapshot(datasettest.Build(t, tables)); !errors.Is(err, ErrDataCorrupt) {
				t.Fatalf("err = %v, mau ErrDataCorrupt", err)
			}
		})
	}
}

func TestChildRange(t *testing.T) {
	codes := []string{"11.01", "31.71", "31.72", "32.01"}
	lo, hi := childRange(func(i int) string { return codes[i] }, len(codes), "31")
	if lo != 1 || hi != 3 {
		t.Fatalf("rentang = %d..%d", lo, hi)
	}
	if lo, hi := childRange(func(i int) string { return codes[i] }, len(codes), "33"); lo != hi {
		t.Fatalf("induk tanpa anak harus kosong, dapat %d..%d", lo, hi)
	}
}
