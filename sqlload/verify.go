package sqlload

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

func mismatch(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrMismatch, fmt.Sprintf(format, args...))
}

// Verify mencocokkan meta, jumlah baris, dan sampel baris (pertama, tengah, terakhir) dengan data tertanam.
func Verify(ctx context.Context, db *sql.DB, o Options) error {
	o, spec, err := normalize(o)
	if err != nil {
		return err
	}
	d, err := loadData()
	if err != nil {
		return err
	}
	q := spec.quote
	for _, kv := range metaRows(o, d) {
		if kv[0] == "package_version" {
			continue
		}
		var got string
		err := db.QueryRowContext(ctx, "SELECT "+q("value")+" FROM "+q(o.Prefix+"meta")+" WHERE "+q("key")+" = "+spec.placeholder(1), kv[0]).Scan(&got)
		if err != nil {
			return fmt.Errorf("sqlload: membaca meta %s: %w", kv[0], err)
		}
		if got != kv[1] {
			return mismatch("meta %s = %q, mau %q", kv[0], got, kv[1])
		}
	}
	for _, t := range selected(o) {
		if t.dataset == "" {
			continue
		}
		name := o.Prefix + t.name
		rows := d.rows[t.dataset]
		var n int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+q(name)).Scan(&n); err != nil {
			return err
		}
		if n != len(rows) {
			return mismatch("%s berisi %d baris, mau %d", name, n, len(rows))
		}
		if len(rows) == 0 {
			continue
		}
		cols := make([]string, len(t.columns))
		for i, c := range t.columns {
			cols[i] = q(c.name)
		}
		key := t.primaryKey[0]
		for _, i := range []int{0, len(rows) / 2, len(rows) - 1} {
			want, err := rowValues(spec, t, d.header[t.dataset], rows[i], i+1)
			if err != nil {
				return err
			}
			keyVal := want[slicesIndex(t, key)]
			got := make([]sql.NullString, len(cols))
			ptrs := make([]any, len(cols))
			for j := range got {
				ptrs[j] = &got[j]
			}
			err = db.QueryRowContext(ctx, "SELECT "+strings.Join(cols, ", ")+" FROM "+q(name)+" WHERE "+q(key)+" = "+spec.placeholder(1), keyVal).Scan(ptrs...)
			if err != nil {
				return mismatch("%s: baris %v tidak terbaca: %v", name, keyVal, err)
			}
			for j, c := range t.columns {
				if !sameValue(c, want[j], got[j]) {
					return mismatch("%s %v kolom %s = %q, mau %v", name, keyVal, c.name, got[j].String, want[j])
				}
			}
		}
	}
	return nil
}

func slicesIndex(t table, name string) int {
	for i, c := range t.columns {
		if c.name == name {
			return i
		}
	}
	return -1
}

// sameValue membandingkan nilai tertanam dengan teks dari database, memperhitungkan representasi tiap dialek.
func sameValue(c column, want any, got sql.NullString) bool {
	if want == nil {
		return !got.Valid
	}
	if !got.Valid {
		return false
	}
	switch c.typ {
	case tDecimal:
		w, err1 := strconv.ParseFloat(fmt.Sprint(want), 64)
		g, err2 := strconv.ParseFloat(got.String, 64)
		return err1 == nil && err2 == nil && math.Abs(w-g) <= 0.5*math.Pow10(-c.scale)
	case tJSON: // bbox: larik 4 angka; database bisa menulis ulang spasi/format JSON
		var w, g []float64
		return json.Unmarshal([]byte(fmt.Sprint(want)), &w) == nil && json.Unmarshal([]byte(got.String), &g) == nil && slices.Equal(w, g)
	case tBool:
		g := strings.ToLower(got.String)
		wantTrue := fmt.Sprint(want) == "1" || fmt.Sprint(want) == "true"
		return (g == "1" || g == "true" || g == "t") == wantTrue
	case tInt:
		return fmt.Sprint(want) == got.String
	default:
		return fmt.Sprint(want) == got.String
	}
}
