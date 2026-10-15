package territory

import (
	"errors"
	"sync"
	"testing"
)

func TestLookupsAgainstEmbeddedData(t *testing.T) {
	p, err := ProvinceByCode("31")
	if err != nil || p.Name != "Daerah Khusus Ibukota Jakarta" {
		t.Fatalf("Province = %+v, %v", p, err)
	}
	r, err := RegencyByCode("31.71")
	if err != nil || r.Name != "Kota Administrasi Jakarta Pusat" || r.Type != RegencyKota || r.ProvinceCode != "31" {
		t.Fatalf("Regency = %+v, %v", r, err)
	}
	d, err := DistrictByCode("31.71.01")
	if err != nil || d.Name != "Gambir" {
		t.Fatalf("District = %+v, %v", d, err)
	}
	v, err := VillageByCode("31.71.01.1001")
	if err != nil || v.Name != "Gambir" || v.DistrictCode != "31.71.01" {
		t.Fatalf("Village = %+v, %v", v, err)
	}
}

func TestCountsAndOrder(t *testing.T) {
	ps, err := Provinces()
	if err != nil || len(ps) != 38 {
		t.Fatalf("Provinces: %d, %v", len(ps), err)
	}
	total := map[string]int{}
	for _, p := range ps {
		rs, err := RegenciesOf(p.Code)
		if err != nil {
			t.Fatal(err)
		}
		total["regencies"] += len(rs)
		for i, r := range rs {
			if r.ProvinceCode != p.Code || (i > 0 && rs[i-1].Code >= r.Code) {
				t.Fatalf("RegenciesOf(%s) salah/tidak urut di %s", p.Code, r.Code)
			}
			ds, err := DistrictsOf(r.Code)
			if err != nil {
				t.Fatal(err)
			}
			total["districts"] += len(ds)
			for _, d := range ds {
				vs, err := VillagesOf(d.Code)
				if err != nil {
					t.Fatal(err)
				}
				total["villages"] += len(vs)
			}
		}
	}
	want := map[string]int{"regencies": 514, "districts": 7285, "villages": 83762}
	for k, n := range want {
		if total[k] != n {
			t.Fatalf("%s = %d, mau %d", k, total[k], n)
		}
	}
}

func TestErrorsAreTyped(t *testing.T) {
	cases := []struct {
		call func() error
		want error
	}{
		{func() error { _, err := ProvinceByCode("31.71"); return err }, ErrWrongLevel},
		{func() error { _, err := ProvinceByCode("99"); return err }, ErrNotFound},
		{func() error { _, err := ProvinceByCode(" 31"); return err }, ErrInvalidCode},
		{func() error { _, err := VillageByCode("31.71.01"); return err }, ErrWrongLevel},
		{func() error { _, err := RegenciesOf("99"); return err }, ErrNotFound},
		{func() error { _, err := IslandByCode("31.71"); return err }, ErrInvalidCode},
		{func() error { _, err := IslandsOf("31.71.01"); return err }, ErrWrongLevel},
		{func() error { _, err := PathOf("99.01"); return err }, ErrNotFound},
		{func() error { _, err := CodeChanges("x"); return err }, ErrInvalidCode},
	}
	for i, c := range cases {
		if err := c.call(); !errors.Is(err, c.want) {
			t.Fatalf("kasus %d: err = %v, mau %v", i, err, c.want)
		}
	}
}

func TestResultsAreCopies(t *testing.T) {
	vs, err := VillagesOf("31.71.01")
	if err != nil || len(vs) == 0 {
		t.Fatal(err)
	}
	vs[0].Name = "diubah"
	if vs[0].Spatial.Center != nil {
		vs[0].Spatial.Center.Latitude = 99
	}
	again, _ := VillagesOf("31.71.01")
	if again[0].Name == "diubah" || (again[0].Spatial.Center != nil && again[0].Spatial.Center.Latitude == 99) {
		t.Fatal("mutasi hasil bocor ke data internal")
	}
}

func TestPath(t *testing.T) {
	p, err := PathOf("31.71.01.1001")
	if err != nil || p.Province.Code != "31" || p.Regency == nil || p.Regency.Code != "31.71" ||
		p.District == nil || p.District.Code != "31.71.01" || p.Village == nil || p.Village.Code != "31.71.01.1001" {
		t.Fatalf("Path = %+v, %v", p, err)
	}
	p, err = PathOf("31")
	if err != nil || p.Regency != nil || p.District != nil || p.Village != nil {
		t.Fatalf("Path(31) = %+v, %v", p, err)
	}
}

func TestIslandsWithoutRegencyIncludedInProvince(t *testing.T) {
	s, err := loadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	var probe Island
	for _, isl := range s.islands {
		if isl.RegencyCode == "" {
			probe = isl
			break
		}
	}
	if probe.Code == "" {
		t.Skip("data tidak memuat pulau tanpa kabupaten/kota")
	}
	list, err := IslandsOf(probe.ProvinceCode)
	if err != nil {
		t.Fatal(err)
	}
	for _, isl := range list {
		if isl.Code == probe.Code {
			return
		}
	}
	t.Fatalf("pulau %s tanpa kabupaten tidak ada di IslandsOf(%s)", probe.Code, probe.ProvinceCode)
}

func TestVersion(t *testing.T) {
	v, err := Version()
	if err != nil || v.Dataset == "" || len(v.ManifestSHA256) != 64 {
		t.Fatalf("Version = %+v, %v", v, err)
	}
}

func TestConcurrentFirstUse(t *testing.T) {
	var wg sync.WaitGroup
	names := make([]string, 64)
	for i := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := VillageByCode("31.71.01.1001")
			if err == nil {
				names[i] = v.Name
			}
		}()
	}
	wg.Wait()
	for i, n := range names {
		if n != "Gambir" {
			t.Fatalf("goroutine %d: %q", i, n)
		}
	}
}
