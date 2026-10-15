package sqlload

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Verify pada database yang tabelnya belum (lengkap) terpasang harus ErrMismatch, bukan galat driver mentah,
// agar aplikasi bisa memutuskan "pasang ulang" dengan errors.Is.
func TestVerifyNotInstalledIsMismatch(t *testing.T) {
	cases := map[string]map[string]bool{
		"kosong":        {},
		"sebagian saja": {"territory_provinces": true, "territory_regencies": true},
		"tanpa meta":    {"territory_provinces": true, "territory_regencies": true, "territory_districts": true, "territory_villages": true},
	}
	for name, existing := range cases {
		t.Run(name, func(t *testing.T) {
			db := openFake(t, &fakeDB{existing: existing})
			err := Verify(context.Background(), db, Options{Dialect: Postgres, Level: Village})
			if !errors.Is(err, ErrMismatch) {
				t.Fatalf("mau ErrMismatch, dapat %v", err)
			}
			if !strings.Contains(err.Error(), "belum terpasang") {
				t.Fatalf("pesan tidak menyebut tabel yang belum terpasang: %v", err)
			}
		})
	}
}
