// Package datasettest membangun data uji kecil berformat sama dengan data tertanam.
package datasettest

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"testing"
	"testing/fstest"

	"github.com/caturseptian/go-indonesia-territory/internal/dataset"
)

// Build menulis setiap tabel (record tanpa header) sebagai CSV ter-gzip dan manifest yang cocok. Tabel yang tidak
// disebut ditulis kosong (header saja).
func Build(t testing.TB, tables map[string][][]string) fstest.MapFS {
	t.Helper()
	fsys := fstest.MapFS{}
	m := dataset.Manifest{DatasetVersion: "uji", EffectiveDate: "2025-06-23", Source: "uji", Files: map[string]dataset.File{}}
	for _, tb := range dataset.Tables() {
		var raw bytes.Buffer
		zw, err := gzip.NewWriterLevel(&raw, gzip.BestCompression)
		if err != nil {
			t.Fatal(err)
		}
		w := csv.NewWriter(zw)
		if err := w.Write(tb.Columns); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteAll(tables[tb.Name]); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw.Bytes())
		file := tb.Name + ".csv.gz"
		fsys[file] = &fstest.MapFile{Data: raw.Bytes()}
		m.Files[tb.Name] = dataset.File{File: file, Rows: len(tables[tb.Name]), SHA256: hex.EncodeToString(sum[:])}
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	fsys["manifest.json"] = &fstest.MapFile{Data: b}
	return fsys
}
