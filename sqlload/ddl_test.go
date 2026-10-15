package sqlload

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "tulis ulang golden DDL")

func TestDDLGolden(t *testing.T) {
	for _, d := range []Dialect{Postgres, MySQL, SQLite} {
		got, err := DDL(Options{Dialect: d, Level: Village, With: AllExtras})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("testdata", "ddl", d.String()+".sql")
		if *update {
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Fatalf("%s: DDL berbeda dari golden\n--- dapat ---\n%s", d, got)
		}
	}
}

func TestDDLIsDeterministicAndQuoted(t *testing.T) {
	a, _ := DDL(Options{Dialect: Postgres, Level: Village, With: AllExtras, Prefix: "geo_"})
	b, _ := DDL(Options{Dialect: Postgres, Level: Village, With: AllExtras, Prefix: "geo_"})
	if a != b {
		t.Fatal("DDL tidak deterministik")
	}
	if !strings.Contains(a, `CREATE TABLE "geo_villages"`) || strings.Contains(a, "postal_code") == false {
		t.Fatalf("DDL Postgres tidak sesuai:\n%s", a)
	}
	m, _ := DDL(Options{Dialect: MySQL, Level: Province})
	if !strings.Contains(m, "CREATE TABLE `territory_provinces`") || strings.Contains(m, "regencies") {
		t.Fatalf("DDL MySQL tidak sesuai:\n%s", m)
	}
}
