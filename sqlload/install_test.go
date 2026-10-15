package sqlload

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestInvalidPrefixSendsNoSQL(t *testing.T) {
	f := &fakeDB{}
	db := openFake(t, f)
	err := Install(context.Background(), db, Options{Dialect: Postgres, Level: Province, Prefix: `x"; DROP TABLE users; --`})
	if !errors.Is(err, ErrInvalidOptions) || len(f.log) != 0 {
		t.Fatalf("err = %v, pernyataan terkirim = %v", err, f.log)
	}
}

func TestInstallRefusesExistingTablesWithoutReplace(t *testing.T) {
	f := &fakeDB{existing: map[string]bool{"territory_provinces": true}}
	db := openFake(t, f)
	err := Install(context.Background(), db, Options{Dialect: Postgres, Level: Province})
	if !errors.Is(err, ErrAlreadyInstalled) {
		t.Fatalf("err = %v", err)
	}
	for _, q := range f.log {
		if strings.HasPrefix(q, "CREATE") || strings.HasPrefix(q, "INSERT") || strings.HasPrefix(q, "DROP") {
			t.Fatalf("pernyataan tulis dikirim: %s", q)
		}
	}
}

func TestInstallPostgresUsesOneTransactionAndBindings(t *testing.T) {
	f := &fakeDB{}
	db := openFake(t, f)
	if err := Install(context.Background(), db, Options{Dialect: Postgres, Level: Province}); err != nil {
		t.Fatal(err)
	}
	var begin, commit, inserts int
	for i, q := range f.log {
		switch {
		case q == "BEGIN":
			begin++
		case q == "COMMIT":
			commit++
		case strings.HasPrefix(q, "INSERT"):
			inserts++
			if strings.Count(q, "$") != len(f.args[i]) {
				t.Fatalf("placeholder %d ≠ nilai %d: %s", strings.Count(q, "$"), len(f.args[i]), q)
			}
			if strings.Contains(q, "Aceh") {
				t.Fatalf("nilai data masuk ke teks SQL: %s", q)
			}
		}
	}
	if begin != 1 || commit != 1 || inserts < 2 {
		t.Fatalf("begin=%d commit=%d insert=%d\n%s", begin, commit, inserts, strings.Join(f.log, "\n"))
	}
}

func TestInstallMySQLDropsCreatedTablesWhenDataFails(t *testing.T) {
	f := &fakeDB{failOn: "INSERT INTO `territory_regencies`"}
	db := openFake(t, f)
	err := Install(context.Background(), db, Options{Dialect: MySQL, Level: Regency})
	if err == nil {
		t.Fatal("galat diharapkan")
	}
	joined := strings.Join(f.log, "\n")
	for _, want := range []string{"ROLLBACK", "DROP TABLE `territory_meta`", "DROP TABLE `territory_regencies`", "DROP TABLE `territory_provinces`"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("tidak ada %q:\n%s", want, joined)
		}
	}
}

func TestInstallHonoursCancelledContext(t *testing.T) {
	f := &fakeDB{}
	db := openFake(t, f)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Install(ctx, db, Options{Dialect: SQLite, Level: Province}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestBatchSizeRespectsPlaceholderLimit(t *testing.T) {
	spec, _ := SQLite.spec()
	if n := batchSize(spec, 9); n*9 > spec.maxParams || n > 1000 || n < 1 {
		t.Fatalf("batch = %d", n)
	}
	pg, _ := Postgres.spec()
	if got := insertSQL(pg, `"t"`, []string{`"a"`, `"b"`}, 2); got != `INSERT INTO "t" ("a", "b") VALUES ($1, $2), ($3, $4)` {
		t.Fatalf("insertSQL = %s", got)
	}
}
