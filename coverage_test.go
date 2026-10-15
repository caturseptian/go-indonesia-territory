package territory

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/caturseptian/go-indonesia-territory/internal/datasettest"
)

func TestLoadSucceeds(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
}

func TestIslandByCode(t *testing.T) {
	s, err := loadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	want := s.islands[0]
	got, err := IslandByCode(want.Code)
	if err != nil || got.Code != want.Code || got.Name != want.Name {
		t.Fatalf("IslandByCode(%s) = %+v, %v", want.Code, got, err)
	}
	if _, err := IslandByCode("11.01.49999"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("pulau tidak ada: %v", err)
	}
}

func TestIslandsOfRegencyAndErrors(t *testing.T) {
	s, err := loadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	var reg string
	for _, isl := range s.islands {
		if isl.RegencyCode != "" {
			reg = isl.RegencyCode
			break
		}
	}
	list, err := IslandsOf(reg)
	if err != nil || len(list) == 0 {
		t.Fatalf("IslandsOf(%s) = %d, %v", reg, len(list), err)
	}
	for _, isl := range list {
		if isl.RegencyCode != reg {
			t.Fatalf("pulau %s bukan milik %s", isl.Code, reg)
		}
	}
	if _, err := IslandsOf("99"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("provinsi tidak ada: %v", err)
	}
	if _, err := IslandsOf("x"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("kode tidak sah: %v", err)
	}
}

func TestCodeChanges(t *testing.T) {
	s, err := loadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	code := s.changes[0].Code
	got, err := CodeChanges(code)
	if err != nil || len(got) == 0 || got[0].Code != code {
		t.Fatalf("CodeChanges(%s) = %+v, %v", code, got, err)
	}
	none, err := CodeChanges("11.01.01.9999")
	if err != nil || len(none) != 0 {
		t.Fatalf("kode tanpa riwayat: %+v, %v", none, err)
	}
}

func TestPathOfEveryLevelAndMissingParts(t *testing.T) {
	for code, depth := range map[string]int{"31.71": 2, "31.71.01": 3} {
		p, err := PathOf(code)
		if err != nil {
			t.Fatal(err)
		}
		got := 1
		if p.Regency != nil {
			got++
		}
		if p.District != nil {
			got++
		}
		if p.Village != nil {
			got++
		}
		if got != depth {
			t.Fatalf("PathOf(%s) kedalaman %d, mau %d", code, got, depth)
		}
	}
	for _, code := range []string{"31.99", "31.71.99", "31.71.01.9999"} {
		if _, err := PathOf(code); !errors.Is(err, ErrNotFound) {
			t.Fatalf("PathOf(%s): %v", code, err)
		}
	}
	if _, err := PathOf("31 "); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("PathOf kode tidak sah: %v", err)
	}
}

func TestLevelPatternUnknown(t *testing.T) {
	if levelPattern("kota") != "" {
		t.Fatal("tingkat tidak dikenal harus kosong")
	}
}

func TestBuildSnapshotMoreCorruption(t *testing.T) {
	cases := map[string]func(map[string][][]string){
		"kecamatan tanpa kabupaten": func(m map[string][][]string) { m["districts"][0][0], m["districts"][0][1] = "31.79.01", "31.79" },
		"desa tanpa kecamatan":      func(m map[string][][]string) { m["villages"][0][0], m["villages"][0][1] = "31.71.09.1001", "31.71.09" },
		"kode pos tidak sah":        func(m map[string][][]string) { m["villages"][0][4] = "1011" },
		"pulau kabupaten salah":     func(m map[string][][]string) { m["islands"][0][3] = "31.99" },
		"pulau kode salah":          func(m map[string][][]string) { m["islands"][0][0] = "31.01.30001" },
		"luas pulau rusak":          func(m map[string][][]string) { m["islands"][0][6] = "x" },
		"bujur rusak":               func(m map[string][][]string) { m["villages"][0][6] = "x" },
		"koordinat pulau rusak":     func(m map[string][][]string) { m["islands"][0][4] = "x" },
		"kode riwayat salah":        func(m map[string][][]string) { m["code_changes"][0][4] = "31.71" },
		"tipe kabupaten salah":      func(m map[string][][]string) { m["regencies"][0][2] = "kotamadya" },
		"spasial kabupaten rusak":   func(m map[string][][]string) { m["regencies"][0][4] = "x" },
		"spasial kecamatan rusak":   func(m map[string][][]string) { m["districts"][0][3] = "x" },
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

func TestBuildSnapshotMissingFiles(t *testing.T) {
	if _, err := buildSnapshot(fstest.MapFS{}); !errors.Is(err, ErrDataCorrupt) {
		t.Fatalf("tanpa manifest: %v", err)
	}
	for _, file := range []string{"provinces.csv.gz", "regencies.csv.gz", "districts.csv.gz", "villages.csv.gz", "islands.csv.gz", "code_changes.csv.gz"} {
		fsys := datasettest.Build(t, validTables())
		delete(fsys, file)
		if _, err := buildSnapshot(fsys); !errors.Is(err, ErrDataCorrupt) {
			t.Fatalf("tanpa %s: %v", file, err)
		}
	}
}
