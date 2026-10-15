// Package dataset membaca data wilayah tertanam: manifest, verifikasi sha256, dan record CSV per tabel.
package dataset

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
)

//go:embed data/manifest.json data/*.csv.gz
var embedded embed.FS

// ErrCorrupt menandai data yang tidak cocok dengan manifest atau formatnya rusak.
var ErrCorrupt = errors.New("dataset: data rusak")

// Table adalah satu tabel data beserta kolom CSV-nya, urut sesuai berkas.
type Table struct {
	Name    string
	Columns []string
}

// tables sama dengan laravel-indonesia-territory (bin/build-data, COLUMNS); urutan juga urutan pemasangan.
var tables = []Table{
	{Name: "provinces", Columns: []string{"code", "name", "latitude", "longitude", "spatial_status", "bbox"}},
	{Name: "regencies", Columns: []string{"code", "province_code", "type", "name", "latitude", "longitude", "spatial_status", "bbox"}},
	{Name: "districts", Columns: []string{"code", "regency_code", "name", "latitude", "longitude", "spatial_status", "bbox"}},
	{Name: "villages", Columns: []string{"code", "district_code", "type", "name", "postal_code", "latitude", "longitude", "spatial_status", "bbox"}},
	{Name: "islands", Columns: []string{"code", "name", "province_code", "regency_code", "latitude", "longitude", "area_km2", "inhabited"}},
	{Name: "code_changes", Columns: []string{"from_document", "to_document", "level", "change", "code", "old_name", "new_name"}},
}

// Tables mengembalikan salinan daftar tabel.
func Tables() []Table {
	out := make([]Table, len(tables))
	for i, t := range tables {
		out[i] = Table{Name: t.Name, Columns: slices.Clone(t.Columns)}
	}
	return out
}

// TableByName mengembalikan salinan satu tabel.
func TableByName(name string) (Table, bool) {
	for _, t := range tables {
		if t.Name == name {
			return Table{Name: t.Name, Columns: slices.Clone(t.Columns)}, true
		}
	}
	return Table{}, false
}

// File adalah entri manifest untuk satu tabel.
type File struct {
	File   string `json:"file"`
	Rows   int    `json:"rows"`
	SHA256 string `json:"sha256"`
}

// Manifest menjelaskan versi dataset dan setiap berkas tabel.
type Manifest struct {
	DatasetVersion string          `json:"dataset_version"`
	EffectiveDate  string          `json:"effective_date"`
	Source         string          `json:"source"`
	Files          map[string]File `json:"files"`
}

// Embedded mengembalikan data tertanam dengan akar = folder data.
func Embedded() (fs.FS, error) {
	return fs.Sub(embedded, "data")
}

// ReadManifest membaca manifest.json dan mengembalikan sha256 (hex) dari byte manifest.
func ReadManifest(fsys fs.FS) (Manifest, string, error) {
	b, err := fs.ReadFile(fsys, "manifest.json")
	if err != nil {
		return Manifest{}, "", fmt.Errorf("%w: manifest: %w", ErrCorrupt, err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, "", fmt.Errorf("%w: manifest: %w", ErrCorrupt, err)
	}
	for _, t := range tables {
		if _, ok := m.Files[t.Name]; !ok {
			return Manifest{}, "", fmt.Errorf("%w: manifest tanpa tabel %s", ErrCorrupt, t.Name)
		}
	}
	sum := sha256.Sum256(b)
	return m, hex.EncodeToString(sum[:]), nil
}

// ReadTable membaca satu tabel setelah memverifikasi sha256 berkas, header, dan jumlah baris terhadap manifest.
func ReadTable(fsys fs.FS, m Manifest, name string) ([][]string, error) {
	t, ok := TableByName(name)
	if !ok {
		return nil, fmt.Errorf("dataset: tabel tidak dikenal %q", name)
	}
	f, ok := m.Files[name]
	if !ok {
		return nil, fmt.Errorf("%w: manifest tanpa tabel %s", ErrCorrupt, name)
	}
	if f.Rows < 0 {
		return nil, fmt.Errorf("%w: jumlah baris %s negatif (%d)", ErrCorrupt, name, f.Rows)
	}
	raw, err := fs.ReadFile(fsys, f.File)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrCorrupt, f.File, err)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != f.SHA256 {
		return nil, fmt.Errorf("%w: sha256 %s tidak cocok dengan manifest", ErrCorrupt, f.File)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrCorrupt, f.File, err)
	}
	defer zr.Close()
	r := csv.NewReader(zr)
	r.FieldsPerRecord = len(t.Columns)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrCorrupt, f.File, err)
	}
	if !slices.Equal(header, t.Columns) {
		return nil, fmt.Errorf("%w: header %s = %q", ErrCorrupt, f.File, header)
	}
	rows := make([][]string, 0, f.Rows)
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrCorrupt, f.File, err)
		}
		rows = append(rows, rec)
	}
	if len(rows) != f.Rows {
		return nil, fmt.Errorf("%w: %s berisi %d baris, manifest %d", ErrCorrupt, f.File, len(rows), f.Rows)
	}
	return rows, nil
}
