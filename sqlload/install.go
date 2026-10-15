package sqlload

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/caturseptian/go-indonesia-territory/internal/dataset"
)

type data struct {
	manifest    dataset.Manifest
	manifestSHA string
	rows        map[string][][]string
	header      map[string][]string
}

func loadData() (data, error) {
	fsys, err := dataset.Embedded()
	if err != nil {
		return data{}, err
	}
	m, sum, err := dataset.ReadManifest(fsys)
	if err != nil {
		return data{}, err
	}
	d := data{manifest: m, manifestSHA: sum, rows: map[string][][]string{}, header: map[string][]string{}}
	for _, t := range dataset.Tables() {
		rows, err := dataset.ReadTable(fsys, m, t.Name)
		if err != nil {
			return data{}, err
		}
		d.rows[t.Name], d.header[t.Name] = rows, t.Columns
	}
	return d, nil
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Install membuat tabel dan memuat data tertanam. Postgres dan SQLite: satu transaksi. MySQL/MariaDB: DDL dulu
// (MySQL meng-commit DDL otomatis), data dalam transaksi, dan tabel yang baru dibuat dihapus lagi bila gagal.
func Install(ctx context.Context, db *sql.DB, o Options) error {
	o, spec, err := normalize(o)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	d, err := loadData()
	if err != nil {
		return err
	}
	existing, err := existingTables(ctx, db, spec, o)
	if err != nil {
		return err
	}
	if len(existing) > 0 && !o.Replace {
		return fmt.Errorf("%w: %s (pakai Replace untuk memasang ulang)", ErrAlreadyInstalled, strings.Join(existing, ", "))
	}
	// Transaksi dibuka tanpa pembatalan otomatis: bila ctx dibatalkan, pernyataan berikutnya gagal dan rollback
	// dijalankan sinkron oleh defer, sehingga koneksi sudah dilepas saat Install kembali (database/sql
	// me-rollback transaksi yang konteksnya batal secara asinkron).
	txCtx := context.WithoutCancel(ctx)
	if spec.transactionalDDL {
		tx, err := db.BeginTx(txCtx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err := dropTables(ctx, tx, spec, existing); err != nil {
			return err
		}
		if err := execAll(ctx, tx, ddlStatements(spec, o)); err != nil {
			return err
		}
		if err := insertAll(ctx, tx, spec, o, d); err != nil {
			return err
		}
		return tx.Commit()
	}
	if err := dropTables(ctx, db, spec, existing); err != nil {
		return err
	}
	var created []string // urut pembuatan; dropTables menghapus dari belakang (anak dulu)
	cleanup := func(cause error) error {
		// Konteks pemanggil mungkin sudah dibatalkan; pembersihan tetap jalan, tetapi dibatasi waktunya.
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		return errors.Join(cause, dropTables(cctx, db, spec, created))
	}
	for _, stmt := range ddlStatements(spec, o) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return cleanup(err)
		}
		if name, ok := createdTable(stmt, spec, o); ok {
			created = append(created, name)
		}
	}
	tx, err := db.BeginTx(txCtx, nil)
	if err != nil {
		return cleanup(err)
	}
	if err := insertAll(ctx, tx, spec, o, d); err != nil {
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			err = errors.Join(err, rerr)
		}
		return cleanup(err)
	}
	if err := tx.Commit(); err != nil {
		return cleanup(err)
	}
	return nil
}

func createdTable(stmt string, spec dialectSpec, o Options) (string, bool) {
	names, _ := Tables(o)
	for _, n := range names {
		if strings.HasPrefix(stmt, "CREATE TABLE "+spec.quote(n)+" ") {
			return n, true
		}
	}
	return "", false
}

// existingTables memeriksa semua tabel skema (semua tingkat dan tambahan) ber-prefix ini, urut pembuatan.
func existingTables(ctx context.Context, db *sql.DB, spec dialectSpec, o Options) ([]string, error) {
	var out []string
	for _, t := range schema() {
		name := o.Prefix + t.name
		var one int
		err := db.QueryRowContext(ctx, spec.existsSQL, name).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, nil
}

func dropTables(ctx context.Context, ex execer, spec dialectSpec, names []string) error {
	ordered := slices.Clone(names)
	slices.Reverse(ordered) // anak dulu
	for _, n := range ordered {
		if _, err := ex.ExecContext(ctx, "DROP TABLE "+spec.quote(n)); err != nil {
			return err
		}
	}
	return nil
}

func execAll(ctx context.Context, ex execer, stmts []string) error {
	for _, s := range stmts {
		if _, err := ex.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func batchSize(spec dialectSpec, ncols int) int {
	return max(1, min(1000, spec.maxParams/ncols))
}

func insertSQL(spec dialectSpec, name string, cols []string, n int) string {
	var b strings.Builder
	b.WriteString("INSERT INTO " + name + " (" + strings.Join(cols, ", ") + ") VALUES ")
	p := 1
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(")
		for j := range cols {
			if j > 0 {
				b.WriteString(", ")
			}
			b.WriteString(spec.placeholder(p))
			p++
		}
		b.WriteString(")")
	}
	return b.String()
}

func rowValues(spec dialectSpec, t table, header []string, rec []string, seq int) ([]any, error) {
	out := make([]any, 0, len(t.columns))
	for _, c := range t.columns {
		if c.source == "" {
			out = append(out, seq)
			continue
		}
		i := slices.Index(header, c.source)
		if i < 0 {
			return nil, fmt.Errorf("sqlload: kolom sumber %s.%s tidak ada", t.dataset, c.source)
		}
		v := rec[i]
		switch {
		case v == "":
			out = append(out, nil)
		case c.typ == tBool:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("sqlload: %s.%s = %q", t.dataset, c.source, v)
			}
			out = append(out, spec.boolValue(b))
		default:
			out = append(out, v)
		}
	}
	return out, nil
}

func insertAll(ctx context.Context, ex execer, spec dialectSpec, o Options, d data) error {
	for _, t := range selected(o) {
		if t.dataset == "" {
			continue
		}
		rows := d.rows[t.dataset]
		cols := make([]string, len(t.columns))
		for i, c := range t.columns {
			cols[i] = spec.quote(c.name)
		}
		n := batchSize(spec, len(cols))
		for start := 0; start < len(rows); start += n {
			if err := ctx.Err(); err != nil {
				return err
			}
			end := min(start+n, len(rows))
			args := make([]any, 0, (end-start)*len(cols))
			for i := start; i < end; i++ {
				vals, err := rowValues(spec, t, d.header[t.dataset], rows[i], i+1)
				if err != nil {
					return err
				}
				args = append(args, vals...)
			}
			if _, err := ex.ExecContext(ctx, insertSQL(spec, spec.quote(o.Prefix+t.name), cols, end-start), args...); err != nil {
				return fmt.Errorf("sqlload: mengisi %s: %w", o.Prefix+t.name, err)
			}
		}
	}
	return insertMeta(ctx, ex, spec, o, d)
}

func packageVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, m := range bi.Deps {
			if m.Path == "github.com/caturseptian/go-indonesia-territory" {
				return m.Version
			}
		}
		if bi.Main.Path == "github.com/caturseptian/go-indonesia-territory" {
			return bi.Main.Version
		}
	}
	return "(devel)"
}

func extrasNames(e Extras) string {
	var out []string
	for _, x := range []struct {
		e    Extras
		name string
	}{{PostalCodes, "postal_codes"}, {Coordinates, "coordinates"}, {Islands, "islands"}, {CodeChanges, "code_changes"}} {
		if e&x.e != 0 {
			out = append(out, x.name)
		}
	}
	return strings.Join(out, ",")
}

func metaRows(o Options, d data) [][2]string {
	return [][2]string{
		{"dataset_version", d.manifest.DatasetVersion},
		{"effective_date", d.manifest.EffectiveDate},
		{"manifest_sha256", d.manifestSHA},
		{"level", strconv.Itoa(int(o.Level))},
		{"with", extrasNames(o.With)},
		{"package_version", packageVersion()},
	}
}

func insertMeta(ctx context.Context, ex execer, spec dialectSpec, o Options, d data) error {
	rows := metaRows(o, d)
	args := make([]any, 0, len(rows)*2)
	for _, r := range rows {
		args = append(args, r[0], r[1])
	}
	_, err := ex.ExecContext(ctx, insertSQL(spec, spec.quote(o.Prefix+"meta"), []string{spec.quote("key"), spec.quote("value")}, len(rows)), args...)
	return err
}
