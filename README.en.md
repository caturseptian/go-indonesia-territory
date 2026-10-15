# go-indonesia-territory

[![ci](https://github.com/caturseptian/go-indonesia-territory/actions/workflows/ci.yml/badge.svg)](https://github.com/caturseptian/go-indonesia-territory/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/caturseptian/go-indonesia-territory.svg)](https://pkg.go.dev/github.com/caturseptian/go-indonesia-territory)

[Bahasa Indonesia](README.md)

Indonesian administrative region codes and data (Kepmendagri region codes, *kode wilayah*) for Go: provinces,
regencies/cities, districts, villages, postal codes, coordinates, islands, and the history of code changes.
**Standard library only** — `go.mod` has no dependencies. Twin of
[laravel-indonesia-territory](https://github.com/caturseptian/laravel-indonesia-territory) with identical data.

## Installation

Requires Go 1.26.9 or newer (tested on Go 1.26 and 1.27).

```bash
go get github.com/caturseptian/go-indonesia-territory
```

## Lookups without a database

```go
import territory "github.com/caturseptian/go-indonesia-territory"

p, err := territory.ProvinceByCode("31")          // Daerah Khusus Ibukota Jakarta
villages, err := territory.VillagesOf("31.71.01") // villages in Gambir district, ordered by code
path, err := territory.PathOf("31.71.01.1001")     // province → regency/city → district → village
c, err := territory.Parse("31.71.01.1001")         // format & level, without checking existence
```

Other functions: `RegencyByCode`, `DistrictByCode`, `VillageByCode`, `IslandByCode`, `Provinces`, `RegenciesOf`,
`DistrictsOf`, `IslandsOf`, `CodeChanges`, `Version`. Every result is a copy and ordered by code, except
`CodeChanges`, which follows the order of the history data.

The data is embedded in the binary (about 3.6 MB) and loaded once on first use; call `territory.Load()` at startup to
load it early. Codes must follow the Kepmendagri format exactly (`31`, `31.71`, `31.71.01`, `31.71.01.1001`); any
other input is rejected with `ErrInvalidCode` and is never normalised. Other errors: `ErrWrongLevel`, `ErrNotFound`.
`Parse` only knows the four administrative levels; look up island codes (`11.01.40001`) with `IslandByCode`.

## Installing into a database

```go
import (
	_ "github.com/jackc/pgx/v5/stdlib" // the driver your application chooses
	"github.com/caturseptian/go-indonesia-territory/sqlload"
)

db, _ := sql.Open("pgx", dsn)
opts := sqlload.Options{Dialect: sqlload.Postgres, Level: sqlload.Village, With: sqlload.AllExtras}
err := sqlload.Install(ctx, db, opts)  // create territory_* tables and load the data
err = sqlload.Verify(ctx, db, opts)    // compare row counts & samples with the embedded data
ddl, _ := sqlload.DDL(opts)            // or take the SQL for your own migrations
```

- Dialects: `sqlload.Postgres`, `sqlload.MySQL` (also MariaDB), `sqlload.SQLite`.
- Every value is sent as a parameter; the only identifier you can set is `Prefix` (`^[a-z][a-z0-9_]{0,30}$`).
- `Verify` returns `ErrMismatch` when the contents differ, the tables are not (fully) installed, or the meta table is
  empty, so `errors.Is(err, sqlload.ErrMismatch)` is enough to decide on a reinstall. Connection/permission errors are
  returned as they are.
- MySQL/MariaDB: the DSN must name a database (`.../db_name`); without one, Install and Verify fail with a
  configuration error. Tables are looked up in the active schema (`current_schema()` on PostgreSQL, `DATABASE()` on
  MySQL, the main SQLite database); tables in other `search_path` schemas or in TEMP/ATTACHed databases do not count.
  The configuration error matches `errors.Is(err, sqlload.ErrInvalidOptions)`. Do not switch databases per connection
  (`USE`) within the same pool.
- Existing tables are never overwritten (`ErrAlreadyInstalled`) unless `Replace: true`. `Replace` drops **every**
  schema table with that prefix, whatever `Level`/`With` you chose, then installs again. On MySQL/MariaDB the drop
  commits immediately, so if the reinstall fails the old tables are already gone.
- PostgreSQL and SQLite: a single transaction. MySQL/MariaDB commit DDL implicitly, so the tables are created first
  and the data is loaded in a transaction; on failure the newly created tables are dropped again.

## Data

| Level | Count |
|---|---:|
| Provinces | 38 |
| Regencies/cities | 514 |
| Districts | 7,285 |
| Villages | 83,762 |
| Islands | 17,374 |

Sources: Kepmendagri 300.2.2-2138 Tahun 2025 (amended by 300.2.2-2430 Tahun 2025), Kepmendagri 100.1.1-6117 Tahun
2022, Permendagri 137 Tahun 2017, Permendagri 58 Tahun 2021, Badan Informasi Geospasial, Pos Indonesia. Extracted and
validated by [Indonesia Region API](https://indonesia-region.caturseptian.site). An independent project, not an
official government service.

## Performance

| Operation | Time (Apple M1 Pro, Go 1.26) | Allocations |
|---|---|---|
| First load (decompress + validate all data, once) | ~188 ms | 151 MB transient |
| `VillageByCode` | 89 ns | 48 B |
| `VillagesOf` (one district) | 874 ns | 992 B |

## License

MIT. See [LICENSE](LICENSE).
