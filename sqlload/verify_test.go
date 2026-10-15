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

func TestVerifyNotInstalledListsMissingTablesInOrder(t *testing.T) {
	db := openFake(t, &fakeDB{existing: map[string]bool{"territory_provinces": true}})
	err := Verify(context.Background(), db, Options{Dialect: Postgres, Level: Village})
	want := "sqlload: isi database tidak cocok dengan data tertanam: tabel belum terpasang: " +
		"territory_regencies, territory_districts, territory_villages, territory_meta"
	if err == nil || err.Error() != want {
		t.Fatalf("pesan:\n dapat %v\n mau   %s", err, want)
	}
}

// Tabel lengkap tetapi meta kosong (mis. proses mati setelah DDL di MySQL, atau tabel di-TRUNCATE) juga ErrMismatch.
func TestVerifyEmptyMetaIsMismatch(t *testing.T) {
	all := map[string]bool{"territory_provinces": true, "territory_regencies": true, "territory_districts": true,
		"territory_villages": true, "territory_meta": true}
	db := openFake(t, &fakeDB{existing: all})
	err := Verify(context.Background(), db, Options{Dialect: Postgres, Level: Village})
	if !errors.Is(err, ErrMismatch) || !strings.Contains(err.Error(), "meta dataset_version tidak ada") {
		t.Fatalf("mau ErrMismatch meta tidak ada, dapat %v", err)
	}
}

// Galat saat memeriksa katalog (koneksi, izin) tidak boleh disamarkan sebagai ErrMismatch.
func TestVerifyCatalogErrorIsNotMismatch(t *testing.T) {
	db := openFake(t, &fakeDB{failOn: "information_schema"})
	err := Verify(context.Background(), db, Options{Dialect: Postgres, Level: Village})
	if err == nil || errors.Is(err, ErrMismatch) {
		t.Fatalf("galat katalog harus muncul apa adanya, dapat %v", err)
	}
}

// Koneksi MySQL tanpa database terpilih adalah galat konfigurasi, bukan "tabel belum terpasang".
func TestMySQLWithoutDatabaseIsConfigError(t *testing.T) {
	for _, call := range []string{"Verify", "Install"} {
		t.Run(call, func(t *testing.T) {
			db := openFake(t, &fakeDB{noDatabase: true})
			o := Options{Dialect: MySQL, Level: Village}
			var err error
			if call == "Verify" {
				err = Verify(context.Background(), db, o)
			} else {
				err = Install(context.Background(), db, o)
			}
			if !errors.Is(err, ErrInvalidOptions) || errors.Is(err, ErrMismatch) || !strings.Contains(err.Error(), "belum memilih database") {
				t.Fatalf("mau galat konfigurasi, dapat %v", err)
			}
		})
	}
}
