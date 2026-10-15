package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	territory "github.com/caturseptian/go-indonesia-territory"
	"github.com/caturseptian/go-indonesia-territory/sqlload"
)

type target struct {
	name    string
	driver  string
	dsn     string
	dialect sqlload.Dialect
}

// targets: SQLite selalu; Postgres/MySQL/MariaDB bila DSN diberikan lewat environment.
func targets(t *testing.T) []target {
	out := []target{{"sqlite", "sqlite", filepath.Join(t.TempDir(), "w.db") + "?_pragma=foreign_keys(1)", sqlload.SQLite}}
	for _, c := range []struct {
		name, env, driver string
		d                 sqlload.Dialect
	}{
		{"postgres", "TERRITORY_PG_DSN", "pgx", sqlload.Postgres},
		{"mysql", "TERRITORY_MYSQL_DSN", "mysql", sqlload.MySQL},
		{"mariadb", "TERRITORY_MARIADB_DSN", "mysql", sqlload.MySQL},
	} {
		if dsn := os.Getenv(c.env); dsn != "" {
			out = append(out, target{c.name, c.driver, dsn, c.d})
		} else if os.Getenv("CI") != "" {
			t.Fatalf("%s kosong di CI", c.env)
		}
	}
	return out
}

func open(t *testing.T, tg target) *sql.DB {
	db, err := sql.Open(tg.driver, tg.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func uniquePrefix() string { return fmt.Sprintf("t%d_", time.Now().UnixNano()%1e12) }

func dropAll(t *testing.T, db *sql.DB, o sqlload.Options, quote func(string) string) {
	names, err := sqlload.Tables(sqlload.Options{Dialect: o.Dialect, Level: sqlload.Village, With: sqlload.AllExtras, Prefix: o.Prefix})
	if err != nil {
		t.Fatal(err)
	}
	for i := len(names) - 1; i >= 0; i-- {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + quote(names[i]))
	}
}

func quoteFor(d sqlload.Dialect) func(string) string {
	if d == sqlload.MySQL {
		return func(s string) string { return "`" + s + "`" }
	}
	return func(s string) string { return `"` + s + `"` }
}

func ph(d sqlload.Dialect) string {
	if d == sqlload.Postgres {
		return "$1"
	}
	return "?"
}

func TestInstallVerifyReplace(t *testing.T) {
	for _, tg := range targets(t) {
		t.Run(tg.name, func(t *testing.T) {
			db := open(t, tg)
			ctx := context.Background()
			o := sqlload.Options{Dialect: tg.dialect, Level: sqlload.Village, With: sqlload.AllExtras, Prefix: uniquePrefix()}
			q := quoteFor(tg.dialect)
			t.Cleanup(func() { dropAll(t, db, o, q) })

			if err := sqlload.Install(ctx, db, o); err != nil {
				t.Fatal(err)
			}
			if err := sqlload.Verify(ctx, db, o); err != nil {
				t.Fatal(err)
			}
			if err := sqlload.Install(ctx, db, o); !errors.Is(err, sqlload.ErrAlreadyInstalled) {
				t.Fatalf("pasang ulang tanpa Replace: %v", err)
			}
			var name string
			if err := db.QueryRow("SELECT "+q("name")+" FROM "+q(o.Prefix+"villages")+" WHERE "+q("code")+" = "+ph(tg.dialect), "31.71.01.1001").Scan(&name); err != nil || name != "Gambir" {
				t.Fatalf("Gambir: %q %v", name, err)
			}
			o.Replace = true
			if err := sqlload.Install(ctx, db, o); err != nil {
				t.Fatal(err)
			}
			if err := sqlload.Verify(ctx, db, o); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Nama berisi kutip tunggal/backslash harus tersimpan persis (Review Focus 3).
func TestNamesWithQuotesStoredExactly(t *testing.T) {
	var probe territory.Village
	ps, _ := territory.Provinces()
search:
	for _, p := range ps {
		rs, _ := territory.RegenciesOf(p.Code)
		for _, r := range rs {
			ds, _ := territory.DistrictsOf(r.Code)
			for _, d := range ds {
				vs, _ := territory.VillagesOf(d.Code)
				for _, v := range vs {
					if strings.ContainsAny(v.Name, `'\`) {
						probe = v
						break search
					}
				}
			}
		}
	}
	if probe.Code == "" {
		t.Skip("tidak ada nama desa berkutip/backslash di data")
	}
	for _, tg := range targets(t) {
		t.Run(tg.name, func(t *testing.T) {
			db := open(t, tg)
			o := sqlload.Options{Dialect: tg.dialect, Level: sqlload.Village, Prefix: uniquePrefix()}
			q := quoteFor(tg.dialect)
			t.Cleanup(func() { dropAll(t, db, o, q) })
			if err := sqlload.Install(context.Background(), db, o); err != nil {
				t.Fatal(err)
			}
			var name string
			if err := db.QueryRow("SELECT "+q("name")+" FROM "+q(o.Prefix+"villages")+" WHERE "+q("code")+" = "+ph(tg.dialect), probe.Code).Scan(&name); err != nil || name != probe.Name {
				t.Fatalf("nama %q tersimpan sebagai %q (%v)", probe.Name, name, err)
			}
		})
	}
}

// Pulau tanpa kabupaten/kota tersimpan dengan regency_code NULL (Review Focus 5).
func TestIslandWithoutRegencyIsNull(t *testing.T) {
	for _, tg := range targets(t) {
		t.Run(tg.name, func(t *testing.T) {
			db := open(t, tg)
			o := sqlload.Options{Dialect: tg.dialect, Level: sqlload.Regency, With: sqlload.Islands, Prefix: uniquePrefix()}
			q := quoteFor(tg.dialect)
			t.Cleanup(func() { dropAll(t, db, o, q) })
			if err := sqlload.Install(context.Background(), db, o); err != nil {
				t.Fatal(err)
			}
			var empty, null int
			if err := db.QueryRow("SELECT COUNT(*) FROM " + q(o.Prefix+"islands") + " WHERE " + q("regency_code") + " = ''").Scan(&empty); err != nil || empty != 0 {
				t.Fatalf("regency_code kosong harus NULL: %d %v", empty, err)
			}
			if err := db.QueryRow("SELECT COUNT(*) FROM " + q(o.Prefix+"islands") + " WHERE " + q("regency_code") + " IS NULL").Scan(&null); err != nil || null != islandsWithoutRegency(t) {
				t.Fatalf("pulau tanpa kabupaten/kota di DB = %d, di data = %d (%v)", null, islandsWithoutRegency(t), err)
			}
		})
	}
}

// Galat di tengah DDL Postgres (nama index bentrok) tidak meninggalkan tabel apa pun.
func TestPostgresFailureLeavesNothing(t *testing.T) {
	for _, tg := range targets(t) {
		if tg.name != "postgres" {
			continue
		}
		db := open(t, tg)
		o := sqlload.Options{Dialect: tg.dialect, Level: sqlload.Regency, Prefix: uniquePrefix()}
		q := quoteFor(tg.dialect)
		blocker := o.Prefix + "blocker"
		if _, err := db.Exec("CREATE TABLE " + q(blocker) + " (x int)"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("CREATE INDEX " + q(o.Prefix+"regencies_name_idx") + " ON " + q(blocker) + " (x)"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Exec("DROP TABLE " + q(blocker)); dropAll(t, db, o, q) })
		if err := sqlload.Install(context.Background(), db, o); err == nil {
			t.Fatal("galat diharapkan")
		}
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = $1", o.Prefix+"provinces").Scan(&n); err != nil || n != 0 {
			t.Fatalf("tabel tersisa: %d %v", n, err)
		}
	}
}

// islandsWithoutRegency menghitung pulau tanpa kabupaten/kota dari API publik territory.
func islandsWithoutRegency(t *testing.T) int {
	t.Helper()
	ps, err := territory.Provinces()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, p := range ps {
		list, err := territory.IslandsOf(p.Code)
		if err != nil {
			t.Fatal(err)
		}
		for _, isl := range list {
			if isl.RegencyCode == "" {
				n++
			}
		}
	}
	if n == 0 {
		t.Fatal("data tidak memuat pulau tanpa kabupaten/kota")
	}
	return n
}

// Install yang dibatalkan di tengah pengisian data tidak meninggalkan tabel apa pun, di setiap dialek
// (Postgres/SQLite: rollback; MySQL/MariaDB: tabel yang baru dibuat dihapus lagi).
func TestCancelledInstallLeavesNothing(t *testing.T) {
	for _, tg := range targets(t) {
		t.Run(tg.name, func(t *testing.T) {
			db := open(t, tg)
			o := sqlload.Options{Dialect: tg.dialect, Level: sqlload.Village, With: sqlload.AllExtras, Prefix: uniquePrefix()}
			q := quoteFor(tg.dialect)
			t.Cleanup(func() { dropAll(t, db, o, q) })
			ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
			defer cancel()
			if err := sqlload.Install(ctx, db, o); err == nil {
				t.Skip("pemasangan selesai sebelum batas waktu; tidak bisa menguji pembatalan di mesin ini")
			}
			names, err := sqlload.Tables(sqlload.Options{Dialect: o.Dialect, Level: sqlload.Village, With: sqlload.AllExtras, Prefix: o.Prefix})
			if err != nil {
				t.Fatal(err)
			}
			for _, n := range names {
				var one int
				err := db.QueryRow(existsSQL(tg.dialect), n).Scan(&one)
				if err == nil {
					t.Fatalf("tabel %s tersisa setelah Install dibatalkan", n)
				}
				if !errors.Is(err, sql.ErrNoRows) {
					t.Fatal(err)
				}
			}
		})
	}
}

func existsSQL(d sqlload.Dialect) string {
	switch d {
	case sqlload.Postgres:
		return "SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1"
	case sqlload.MySQL:
		return "SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
	default:
		return "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?"
	}
}

// Verify sebelum Install (dan setelah tabel dihapus) harus ErrMismatch di setiap dialek.
func TestVerifyBeforeInstallIsMismatch(t *testing.T) {
	for _, tg := range targets(t) {
		t.Run(tg.name, func(t *testing.T) {
			db := open(t, tg)
			ctx := context.Background()
			o := sqlload.Options{Dialect: tg.dialect, Level: sqlload.Village, With: sqlload.AllExtras, Prefix: uniquePrefix()}
			q := quoteFor(tg.dialect)
			t.Cleanup(func() { dropAll(t, db, o, q) })

			if err := sqlload.Verify(ctx, db, o); !errors.Is(err, sqlload.ErrMismatch) {
				t.Fatalf("database kosong: mau ErrMismatch, dapat %v", err)
			}
			if err := sqlload.Install(ctx, db, o); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec("DROP TABLE " + q(o.Prefix+"code_changes")); err != nil {
				t.Fatal(err)
			}
			if err := sqlload.Verify(ctx, db, o); !errors.Is(err, sqlload.ErrMismatch) {
				t.Fatalf("satu tabel hilang: mau ErrMismatch, dapat %v", err)
			}
		})
	}
}
