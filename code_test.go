package territory

import (
	"errors"
	"strings"
	"testing"
)

func TestParseValidCodes(t *testing.T) {
	cases := []struct {
		in     string
		level  Level
		parent string
	}{
		{"31", LevelProvince, ""},
		{"31.71", LevelRegency, "31"},
		{"31.71.01", LevelDistrict, "31.71"},
		{"31.71.01.1001", LevelVillage, "31.71.01"},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.in, err)
		}
		if got != (Code{Value: c.in, Level: c.level, Parent: c.parent}) {
			t.Fatalf("Parse(%q) = %+v", c.in, got)
		}
	}
}

func TestParseRejectsWithoutNormalising(t *testing.T) {
	for _, in := range []string{"", " 31", "31 ", "3171", "31.7a", "31.71.", "31..71", "3", "311", "31.71.01.100", "31.71.01.10011",
		"31.71.40001", "٣١", "31\u00a0", "31.71.01.1001\n"} {
		if _, err := Parse(in); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("Parse(%q) err = %v, mau ErrInvalidCode", in, err)
		}
	}
}

func TestMatchPatternIsland(t *testing.T) {
	if !matchPattern("11.01.40001", islandPattern) || matchPattern("11.01.30001", islandPattern) {
		t.Fatal("pola pulau salah")
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"31", "31.71", "31.71.01", "31.71.01.1001", "", "x", "31.71.40001"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		c, err := Parse(s)
		if err != nil {
			if !errors.Is(err, ErrInvalidCode) {
				t.Fatalf("galat tak terduga %v", err)
			}
			return
		}
		if c.Value != s || !matchPattern(s, levelPattern(c.Level)) {
			t.Fatalf("Parse(%q) = %+v tidak cocok pola", s, c)
		}
		if c.Level != LevelProvince && !strings.HasPrefix(s, c.Parent+".") {
			t.Fatalf("Parent %q bukan awalan %q", c.Parent, s)
		}
	})
}

// Kode pulau bukan tingkat administratif: Parse menolaknya, IslandByCode menerimanya (sesuai godoc Parse).
func TestParseRejectsIslandCodeButIslandByCodeAccepts(t *testing.T) {
	if _, err := Parse("11.01.40001"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("Parse kode pulau: mau ErrInvalidCode, dapat %v", err)
	}
	if i, err := IslandByCode("11.01.40001"); err != nil || i.Name != "Pulau Batukapal" {
		t.Fatalf("IslandByCode: %+v %v", i, err)
	}
}
