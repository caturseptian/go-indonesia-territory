# Changelog

## 1.0.0 - 2026-10-15

Rilis pertama.

- Paket `territory`: lookup provinsi, kabupaten/kota, kecamatan, desa/kelurahan, pulau, rantai wilayah, riwayat
  perubahan kode, dari data tertanam yang diverifikasi sha256.
- Paket `sqlload`: DDL deterministik dan pemasangan data ke PostgreSQL, MySQL/MariaDB, SQLite dengan parameter binding.
- Butuh Go 1.26.9 atau lebih baru (perbaikan keamanan `net/http` GO-2026-6617); diuji di Go 1.26 dan 1.27.
- Dataset `kepmendagri-300.2.2-2138-2025+2430-2025`, isi identik dengan laravel-indonesia-territory v1.0.1
  (dibandingkan per tabel dengan `builddata -compare`).
