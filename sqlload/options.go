// Package sqlload membuat tabel wilayah dan memuat data tertanam ke database aplikasi lewat database/sql.
//
// Paket ini tidak mengimpor driver; aplikasi membuka *sql.DB dengan driver pilihannya (mis. pgx, go-sql-driver/mysql,
// modernc.org/sqlite). Semua nilai dikirim sebagai parameter; satu-satunya identifier yang bisa diatur adalah Prefix.
package sqlload

import (
	"errors"
	"fmt"
)

// Level adalah tingkat wilayah terdalam yang dipasang; tingkat di atasnya selalu ikut.
type Level int

// Nilai Level, dari yang paling dangkal.
const (
	Province Level = iota + 1
	Regency
	District
	Village
)

// Extras memilih data tambahan.
type Extras uint8

const (
	PostalCodes Extras = 1 << iota // kode pos desa (butuh Level Village)
	Coordinates                    // latitude, longitude, spatial_status, bbox
	Islands                        // tabel pulau (butuh Level >= Regency)
	CodeChanges                    // tabel riwayat perubahan kode
	AllExtras   = PostalCodes | Coordinates | Islands | CodeChanges
)

// DefaultPrefix dipakai bila Options.Prefix kosong.
const DefaultPrefix = "territory_"

// Options mengatur DDL, Install, dan Verify.
type Options struct {
	Dialect Dialect
	Level   Level
	With    Extras
	Prefix  string // ^[a-z][a-z0-9_]{0,30}$; kosong = DefaultPrefix
	Replace bool   // Install: hapus tabel ber-prefix yang ada lalu pasang ulang
}

// Galat yang dikembalikan paket ini; periksa dengan errors.Is.
var (
	// ErrInvalidOptions: Options tidak sah (dialect, level, extras, atau prefix).
	ErrInvalidOptions = errors.New("sqlload: opsi tidak sah")
	// ErrAlreadyInstalled: Install menemukan tabel ber-prefix yang sudah ada dan Replace false.
	ErrAlreadyInstalled = errors.New("sqlload: tabel sudah ada")
	// ErrMismatch: Verify menemukan tabel belum (lengkap) terpasang, meta kosong/berbeda, atau jumlah baris/sampel
	// yang berbeda dari data tertanam.
	ErrMismatch = errors.New("sqlload: isi database tidak cocok dengan data tertanam")
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidOptions, fmt.Sprintf(format, args...))
}

func normalize(o Options) (Options, dialectSpec, error) {
	spec, ok := o.Dialect.spec()
	if !ok {
		return o, spec, invalid("dialek wajib diisi (Postgres, MySQL, atau SQLite)")
	}
	if o.Level < Province || o.Level > Village {
		return o, spec, invalid("tingkat %d tidak dikenal", o.Level)
	}
	if o.With&^AllExtras != 0 {
		return o, spec, invalid("data tambahan %b tidak dikenal", o.With)
	}
	if o.With&PostalCodes != 0 && o.Level != Village {
		return o, spec, invalid("kode pos butuh tingkat Village")
	}
	if o.With&Islands != 0 && o.Level < Regency {
		return o, spec, invalid("pulau butuh tingkat Regency atau lebih dalam")
	}
	if o.Prefix == "" {
		o.Prefix = DefaultPrefix
	}
	if !validPrefix(o.Prefix) {
		return o, spec, invalid("prefix %q harus huruf kecil/angka/garis bawah, diawali huruf, maksimal 31 karakter", o.Prefix)
	}
	return o, spec, nil
}

func validPrefix(p string) bool {
	if len(p) < 1 || len(p) > 31 || p[0] < 'a' || p[0] > 'z' {
		return false
	}
	for i := 1; i < len(p); i++ {
		c := p[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}
