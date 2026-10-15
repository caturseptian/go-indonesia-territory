package main

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
	"testing/fstest"
)

func gz(t *testing.T, s string, level int) []byte {
	t.Helper()
	var b bytes.Buffer
	zw, err := gzip.NewWriterLevel(&b, level)
	if err != nil {
		t.Fatal(err)
	}
	zw.Write([]byte(s))
	zw.Close()
	return b.Bytes()
}

// Versi Go lain bisa menghasilkan byte gzip berbeda untuk isi yang sama; -check harus menilai isinya.
func TestCheckDirIgnoresGzipBytesAndManifestHashes(t *testing.T) {
	fresh := map[string][]byte{
		"provinces.csv.gz": gz(t, "code,name\n11,ACEH\n", gzip.BestCompression),
		"manifest.json":    []byte(`{"dataset_version":"v1","files":{"provinces":{"file":"provinces.csv.gz","rows":1,"sha256":"aaa"}}}`),
	}
	cur := fstest.MapFS{
		"provinces.csv.gz": {Data: gz(t, "code,name\n11,ACEH\n", gzip.NoCompression)},
		"manifest.json":    {Data: []byte(`{"dataset_version":"v1","files":{"provinces":{"file":"provinces.csv.gz","rows":1,"sha256":"bbb"}}}`)},
	}
	if bytes.Equal(fresh["provinces.csv.gz"], cur["provinces.csv.gz"].Data) {
		t.Fatal("fixture: byte gzip seharusnya berbeda")
	}
	if err := checkDir(fresh, cur); err != nil {
		t.Fatalf("isi sama dinyatakan berbeda: %v", err)
	}
}

func TestCheckDirReportsContentDifferences(t *testing.T) {
	manifest := []byte(`{"dataset_version":"v1","files":{"provinces":{"file":"provinces.csv.gz","rows":1,"sha256":"aaa"}}}`)
	fresh := map[string][]byte{"provinces.csv.gz": gz(t, "code,name\n11,ACEH\n", gzip.BestCompression), "manifest.json": manifest}
	cases := map[string]fstest.MapFS{
		"isi CSV":     {"provinces.csv.gz": {Data: gz(t, "code,name\n11,ACEH BARU\n", gzip.BestCompression)}, "manifest.json": {Data: manifest}},
		"manifest":    {"provinces.csv.gz": {Data: fresh["provinces.csv.gz"]}, "manifest.json": {Data: []byte(strings.Replace(string(manifest), "v1", "v2", 1))}},
		"file hilang": {"manifest.json": {Data: manifest}},
		"bukan gzip":  {"provinces.csv.gz": {Data: []byte("code,name\n")}, "manifest.json": {Data: manifest}},
	}
	for name, cur := range cases {
		if err := checkDir(fresh, cur); err == nil {
			t.Errorf("%s: perbedaan tidak terdeteksi", name)
		}
	}
}
