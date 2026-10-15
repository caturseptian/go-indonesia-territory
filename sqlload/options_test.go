package sqlload

import (
	"errors"
	"testing"
)

func TestNormalizeRejectsInvalidOptions(t *testing.T) {
	cases := map[string]Options{
		"tanpa dialek":           {Level: Village},
		"tingkat nol":            {Dialect: Postgres},
		"tingkat berlebih":       {Dialect: Postgres, Level: Village + 1},
		"extra tak dikenal":      {Dialect: Postgres, Level: Village, With: 1 << 7},
		"kode pos tanpa desa":    {Dialect: Postgres, Level: District, With: PostalCodes},
		"pulau tanpa kab/kota":   {Dialect: Postgres, Level: Province, With: Islands},
		"prefix kutip":           {Dialect: Postgres, Level: Village, Prefix: `x"; DROP TABLE x; --`},
		"prefix spasi":           {Dialect: Postgres, Level: Village, Prefix: "a b"},
		"prefix huruf besar":     {Dialect: Postgres, Level: Village, Prefix: "Territory_"},
		"prefix angka depan":     {Dialect: Postgres, Level: Village, Prefix: "1t_"},
		"prefix unicode":         {Dialect: Postgres, Level: Village, Prefix: "ｔ_"},
		"prefix terlalu panjang": {Dialect: Postgres, Level: Village, Prefix: "abcdefghijklmnopqrstuvwxyzabcdef"},
	}
	for name, o := range cases {
		if _, _, err := normalize(o); !errors.Is(err, ErrInvalidOptions) {
			t.Fatalf("%s: err = %v, mau ErrInvalidOptions", name, err)
		}
	}
	o, _, err := normalize(Options{Dialect: MySQL, Level: Village, With: AllExtras})
	if err != nil || o.Prefix != DefaultPrefix {
		t.Fatalf("opsi sah ditolak: %+v %v", o, err)
	}
}

func TestTablesFollowLevelAndExtras(t *testing.T) {
	got, err := Tables(Options{Dialect: SQLite, Level: District, Prefix: "w_"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"w_provinces", "w_regencies", "w_districts", "w_meta"}
	if len(got) != len(want) {
		t.Fatalf("Tables = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Tables = %v", got)
		}
	}
}
