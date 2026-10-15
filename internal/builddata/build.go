package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/caturseptian/go-indonesia-territory/internal/dataset"
)

func fetch(ctx context.Context, client *http.Client, url string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "go-indonesia-territory builddata")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: status %d", url, resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	return dec.Decode(into)
}

// cell menyalin nilai JSON apa adanya: angka sebagai token aslinya, bool 1/0, null kosong, array/objek sebagai JSON ringkas.
func cell(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", nil
	case bool:
		if x {
			return "1", nil
		}
		return "0", nil
	case string:
		return x, nil
	case json.Number:
		return x.String(), nil
	default:
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(x); err != nil {
			return "", err
		}
		return strings.TrimSuffix(b.String(), "\n"), nil
	}
}

func gzipCSV(header []string, rows []map[string]any, keyCol string) ([]byte, int, error) {
	var raw bytes.Buffer
	zw, err := gzip.NewWriterLevel(&raw, gzip.BestCompression) // Header nol: tanpa ModTime dan Name
	if err != nil {
		return nil, 0, err
	}
	w := csv.NewWriter(zw)
	if err := w.Write(header); err != nil {
		return nil, 0, err
	}
	for i, r := range rows {
		rec := make([]string, len(header))
		for j, col := range header {
			if rec[j], err = cell(r[col]); err != nil {
				return nil, 0, err
			}
		}
		code, err := cell(r["code"])
		if err != nil {
			return nil, 0, err
		}
		key, err := cell(r[keyCol])
		if err != nil {
			return nil, 0, err
		}
		if code == "" || key == "" { // sama dengan bin/build-data PHP: code dan name/level wajib ada
			return nil, 0, fmt.Errorf("baris %d tanpa code/%s: %v", i+1, keyCol, r)
		}
		if err := w.Write(rec); err != nil {
			return nil, 0, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, 0, err
	}
	if err := zw.Close(); err != nil {
		return nil, 0, err
	}
	return raw.Bytes(), len(rows), nil
}

type changeset struct {
	From, To string
	Added    map[string][]map[string]any `json:"added"`
	Removed  map[string][]map[string]any `json:"removed"`
	Renamed  map[string][]map[string]any `json:"renamed"`
}

// build mengunduh semua tabel dan menghasilkan berkas data + manifest.json (sama dengan bin/build-data PHP).
func build(ctx context.Context, client *http.Client, base string) (map[string][]byte, error) {
	base = strings.TrimRight(base, "/")
	api := base + "/api/v1"
	var meta struct {
		DatasetVersion string         `json:"dataset_version"`
		EffectiveDate  string         `json:"effective_date"`
		Counts         map[string]any `json:"counts"` // berisi juga rincian per tipe (objek)
	}
	if err := fetch(ctx, client, api+"/meta.json", &meta); err != nil {
		return nil, err
	}
	var provinces, regencies, districts, islands []map[string]any
	for path, into := range map[string]*[]map[string]any{"/provinces.json": &provinces, "/regencies.json": &regencies, "/districts.json": &districts, "/islands.json": &islands} {
		if err := fetch(ctx, client, api+path, into); err != nil {
			return nil, err
		}
	}
	var villages []map[string]any
	for _, p := range provinces {
		var part []map[string]any
		if err := fetch(ctx, client, fmt.Sprintf("%s/provinces/%s/villages.json", api, p["code"]), &part); err != nil {
			return nil, err
		}
		villages = append(villages, part...)
	}
	var index struct {
		Changesets []struct{ URL string } `json:"changesets"`
	}
	if err := fetch(ctx, client, api+"/changes.json", &index); err != nil {
		return nil, err
	}
	var changes []map[string]any
	levels := []struct{ plural, level string }{{"provinces", "province"}, {"regencies", "regency"}, {"districts", "district"}, {"villages", "village"}}
	for _, cs := range index.Changesets {
		var d changeset
		if err := fetch(ctx, client, base+cs.URL, &d); err != nil {
			return nil, err
		}
		for _, l := range levels {
			common := func(change string) map[string]any {
				return map[string]any{"from_document": d.From, "to_document": d.To, "level": l.level, "change": change}
			}
			for _, r := range d.Added[l.plural] {
				m := common("added")
				m["code"], m["new_name"] = r["code"], r["name"]
				changes = append(changes, m)
			}
			for _, r := range d.Removed[l.plural] {
				m := common("removed")
				m["code"], m["old_name"] = r["code"], r["name"]
				changes = append(changes, m)
			}
			for _, r := range d.Renamed[l.plural] {
				m := common("renamed")
				m["code"], m["old_name"], m["new_name"] = r["code"], r["from"], r["to"]
				changes = append(changes, m)
			}
		}
	}
	source := map[string][]map[string]any{"provinces": provinces, "regencies": regencies, "districts": districts, "villages": villages, "islands": islands, "code_changes": changes}

	out := map[string][]byte{}
	m := dataset.Manifest{DatasetVersion: meta.DatasetVersion, EffectiveDate: meta.EffectiveDate, Source: "https://indonesia-region.caturseptian.site", Files: map[string]dataset.File{}}
	for _, t := range dataset.Tables() {
		key := "name"
		if t.Name == "code_changes" {
			key = "level"
		}
		data, n, err := gzipCSV(t.Columns, source[t.Name], key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t.Name, err)
		}
		if err := checkCount(meta.Counts, t.Name, n); err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		file := t.Name + ".csv.gz"
		out[file] = data
		m.Files[t.Name] = dataset.File{File: file, Rows: n, SHA256: hex.EncodeToString(sum[:])}
	}
	var mb bytes.Buffer
	enc := json.NewEncoder(&mb)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	out["manifest.json"] = mb.Bytes()
	return out, nil
}

// checkCount mencocokkan jumlah baris dengan meta.json untuk empat tingkat wilayah, sama dengan bin/build-data PHP.
func checkCount(counts map[string]any, table string, n int) error {
	switch table {
	case "provinces", "regencies", "districts", "villages":
	default:
		return nil
	}
	num, ok := counts[table].(json.Number)
	if !ok {
		return fmt.Errorf("meta.json tanpa jumlah %s", table)
	}
	want, err := num.Int64()
	if err != nil || want != int64(n) {
		return fmt.Errorf("jumlah %s (%d) tidak sama dengan meta.json (%s)", table, n, num)
	}
	return nil
}

// numericColumns dibandingkan menurut nilai: PHP menulis float kecil sebagai 1.8E-5, API (dan builddata) 0.000018.
var numericColumns = map[string]bool{"latitude": true, "longitude": true, "area_km2": true}

// canonicalSHA: sha256 isi record (pemisah \x1f antarkolom, \x1e akhir baris), tidak bergantung gaya kutip CSV;
// kolom angka dinormalisasi ke representasi float terpendek.
func canonicalSHA(rows [][]string, header []string) string {
	h := sha256.New()
	write := func(rec []string, normalize bool) {
		out := rec
		if normalize {
			out = make([]string, len(rec))
			for i, v := range rec {
				out[i] = v
				if numericColumns[header[i]] && v != "" {
					if f, err := strconv.ParseFloat(v, 64); err == nil {
						out[i] = strconv.FormatFloat(f, 'g', -1, 64)
					}
				}
			}
		}
		io.WriteString(h, strings.Join(out, "\x1f"))
		io.WriteString(h, "\x1e")
	}
	write(header, false)
	for _, r := range rows {
		write(r, true)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// compareDirs membandingkan isi dua folder data per tabel; hasilnya laporan, galat bila ada perbedaan.
func compareDirs(a, b fs.FS) (string, error) {
	ma, _, err := dataset.ReadManifest(a)
	if err != nil {
		return "", err
	}
	mb, _, err := dataset.ReadManifest(b)
	if err != nil {
		return "", err
	}
	var report strings.Builder
	diff := false
	for _, t := range dataset.Tables() {
		ra, err := dataset.ReadTable(a, ma, t.Name)
		if err != nil {
			return "", err
		}
		rb, err := dataset.ReadTable(b, mb, t.Name)
		if err != nil {
			return "", err
		}
		sa, sb := canonicalSHA(ra, t.Columns), canonicalSHA(rb, t.Columns)
		status := "identik"
		if sa != sb || len(ra) != len(rb) {
			status, diff = "BERBEDA", true
		}
		fmt.Fprintf(&report, "%-13s %7d baris  %s  %s\n", t.Name, len(ra), sa[:16], status)
	}
	if diff {
		return report.String(), fmt.Errorf("data berbeda")
	}
	return report.String(), nil
}
