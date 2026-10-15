
# go-indonesia-territory

[![ci](https://github.com/caturseptian/go-indonesia-territory/actions/workflows/ci.yml/badge.svg)](https://github.com/caturseptian/go-indonesia-territory/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/caturseptian/go-indonesia-territory.svg)](https://pkg.go.dev/github.com/caturseptian/go-indonesia-territory)

[English](README.en.md)

Kode dan data wilayah administrasi Indonesia (kode wilayah Kepmendagri) untuk Go: provinsi, kabupaten/kota, kecamatan,
desa/kelurahan, kode pos, koordinat, pulau, dan riwayat perubahan kode. **Standard library saja** — `go.mod` tanpa
dependensi. Kembaran [laravel-indonesia-territory](https://github.com/caturseptian/laravel-indonesia-territory) dengan
data yang identik.

## Instalasi

Butuh Go 1.26.9 atau lebih baru (diuji di Go 1.26 dan 1.27).

```bash
go get github.com/caturseptian/go-indonesia-territory
```

## Lookup tanpa database

```go
import territory "github.com/caturseptian/go-indonesia-territory"

p, err := territory.ProvinceByCode("31")          // Daerah Khusus Ibukota Jakarta
desa, err := territory.VillagesOf("31.71.01")      // desa/kelurahan di Kecamatan Gambir, urut kode
path, err := territory.PathOf("31.71.01.1001")     // provinsi → kabupaten/kota → kecamatan → desa
c, err := territory.Parse("31.71.01.1001")         // format & tingkat, tanpa cek keberadaan
```

Fungsi lain: `RegencyByCode`, `DistrictByCode`, `VillageByCode`, `IslandByCode`, `Provinces`, `RegenciesOf`,
`DistrictsOf`, `IslandsOf`, `CodeChanges`, `Version`. Semua hasil berupa salinan dan urut kode, kecuali `CodeChanges`
yang mengikuti urutan data riwayat.

Data tertanam di binary (± 3,6 MB) dan dimuat sekali saat pertama dipakai; panggil `territory.Load()` saat aplikasi
mulai untuk memuat lebih awal. Kode harus persis berformat Kepmendagri (`31`, `31.71`, `31.71.01`, `31.71.01.1001`);
masukan lain ditolak `ErrInvalidCode` tanpa dinormalisasi. Galat lain: `ErrWrongLevel`, `ErrNotFound`.

## Memasang ke database

```go
import (
	_ "github.com/jackc/pgx/v5/stdlib" // driver pilihan aplikasi
	"github.com/caturseptian/go-indonesia-territory/sqlload"
)

db, _ := sql.Open("pgx", dsn)
opts := sqlload.Options{Dialect: sqlload.Postgres, Level: sqlload.Village, With: sqlload.AllExtras}
err := sqlload.Install(ctx, db, opts)  // buat tabel territory_* dan isi data
err = sqlload.Verify(ctx, db, opts)    // cocokkan jumlah baris & sampel dengan data tertanam
ddl, _ := sqlload.DDL(opts)            // atau ambil SQL-nya untuk migrasi sendiri
```

- Dialek: `sqlload.Postgres`, `sqlload.MySQL` (juga MariaDB), `sqlload.SQLite`.
- Semua nilai dikirim sebagai parameter; satu-satunya identifier yang bisa diatur adalah `Prefix`
  (`^[a-z][a-z0-9_]{0,30}$`).
- Tabel yang sudah ada tidak ditimpa (`ErrAlreadyInstalled`) kecuali `Replace: true`. `Replace` menghapus **semua**
  tabel skema ber-prefix tersebut, apa pun `Level`/`With` yang dipilih, lalu memasang ulang. Di MySQL/MariaDB
  penghapusan itu langsung di-commit, jadi bila pemasangan ulang gagal, tabel lama sudah tidak ada.
- PostgreSQL dan SQLite: satu transaksi. MySQL/MariaDB meng-commit DDL otomatis, jadi tabel dibuat dulu lalu data diisi
  dalam transaksi; bila gagal, tabel yang baru dibuat dihapus lagi.

## Data

| Tingkat | Jumlah |
|---|---:|
| Provinsi | 38 |
| Kabupaten/kota | 514 |
| Kecamatan | 7.285 |
| Desa/kelurahan | 83.762 |
| Pulau | 17.374 |

Sumber: Kepmendagri 300.2.2-2138 Tahun 2025 (diubah 300.2.2-2430 Tahun 2025), Kepmendagri 100.1.1-6117 Tahun 2022,
Permendagri 137 Tahun 2017, Permendagri 58 Tahun 2021, Badan Informasi Geospasial, Pos Indonesia. Diekstrak dan
divalidasi oleh [Indonesia Region API](https://indonesia-region.caturseptian.site). Proyek independen, bukan layanan
resmi pemerintah.

## Performa

| Operasi | Waktu (Apple M1 Pro, Go 1.26) | Alokasi |
|---|---|---|
| Muat pertama (dekompres + validasi semua data, sekali) | ±188 ms | 151 MB sementara |
| `VillageByCode` | 89 ns | 48 B |
| `VillagesOf` (satu kecamatan) | 874 ns | 992 B |

## Lisensi

MIT. Lihat [LICENSE](LICENSE).
