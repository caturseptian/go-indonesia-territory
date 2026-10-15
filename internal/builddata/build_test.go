package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/caturseptian/go-indonesia-territory/internal/dataset"
)

func fakeAPI(t *testing.T, override map[string]string) *httptest.Server {
	pages := map[string]string{
		"/api/v1/meta.json":                  `{"dataset_version":"uji-1","effective_date":"2025-06-23","counts":{"provinces":1,"regencies":1,"regencies_by_type":{"kota":1},"districts":1,"villages":1,"villages_by_type":{"kelurahan":1},"islands":1}}`,
		"/api/v1/provinces.json":             `[{"code":"31","name":"DKI Jakarta","latitude":-6.2,"longitude":106.816666,"spatial_status":"available","bbox":[106.39,-6.37,106.97,-5.2]}]`,
		"/api/v1/regencies.json":             `[{"code":"31.71","province_code":"31","type":"kota","name":"Kota Jakarta Pusat","latitude":null,"longitude":null,"spatial_status":"not_available","bbox":null}]`,
		"/api/v1/districts.json":             `[{"code":"31.71.01","regency_code":"31.71","name":"Gambir","latitude":null,"longitude":null,"spatial_status":"not_available","bbox":null}]`,
		"/api/v1/provinces/31/villages.json": `[{"code":"31.71.01.1001","district_code":"31.71.01","type":"kelurahan","name":"Gambir","postal_code":"10110","latitude":-6.17,"longitude":106.82,"spatial_status":"available","bbox":[106.8,-6.2,106.9,-6.1]}]`,
		"/api/v1/islands.json":               `[{"code":"31.01.40001","name":"Pulau Uji","province_code":"31","regency_code":null,"latitude":-5.6,"longitude":106.5,"area_km2":0.25,"inhabited":true}]`,
		"/api/v1/changes.json":               `{"changesets":[{"url":"/api/v1/changes/a-b.json"}]}`,
		"/api/v1/changes/a-b.json":           `{"from":"A","to":"B","added":{"provinces":[{"code":"31","name":"DKI"}]},"removed":{},"renamed":{"villages":[{"code":"31.71.01.1001","from":"Lama","to":"Gambir"}]}}`,
	}
	for k, v := range override {
		pages[k] = v
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestBuildIsDeterministicAndReadable(t *testing.T) {
	srv := fakeAPI(t, nil)
	a, err := build(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	b, err := build(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	for name := range a {
		if !bytes.Equal(a[name], b[name]) {
			t.Fatalf("%s tidak deterministik", name)
		}
	}
	fsys := fstest.MapFS{}
	for name, data := range a {
		fsys[name] = &fstest.MapFile{Data: data}
	}
	m, _, err := dataset.ReadManifest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(a["manifest.json"]), "boundaries") || m.DatasetVersion != "uji-1" {
		t.Fatalf("manifest: %s", a["manifest.json"])
	}
	prov, _ := dataset.ReadTable(fsys, m, "provinces")
	if prov[0][2] != "-6.2" || prov[0][3] != "106.816666" || prov[0][5] != "[106.39,-6.37,106.97,-5.2]" {
		t.Fatalf("angka/JSON tidak disalin apa adanya: %q", prov[0])
	}
	isl, _ := dataset.ReadTable(fsys, m, "islands")
	if isl[0][3] != "" || isl[0][7] != "1" {
		t.Fatalf("pulau: %q", isl[0])
	}
	ch, _ := dataset.ReadTable(fsys, m, "code_changes")
	if len(ch) != 2 || ch[0][2] != "province" || ch[0][3] != "added" || ch[0][6] != "DKI" ||
		ch[1][3] != "renamed" || ch[1][5] != "Lama" || ch[1][6] != "Gambir" {
		t.Fatalf("riwayat: %q", ch)
	}
}

func TestBuildFailsWhenCountsDisagree(t *testing.T) {
	srv := fakeAPI(t, map[string]string{
		"/api/v1/meta.json": `{"dataset_version":"x","effective_date":"x","counts":{"provinces":2,"regencies":1,"districts":1,"villages":1}}`,
	})
	if _, err := build(context.Background(), srv.Client(), srv.URL); err == nil {
		t.Fatal("galat jumlah diharapkan")
	}
}

func TestCanonicalSHAIgnoresQuotingButNotContent(t *testing.T) {
	h := []string{"code", "name"}
	a := canonicalSHA([][]string{{"11", "Kab Aceh"}}, h)
	if a != canonicalSHA([][]string{{"11", "Kab Aceh"}}, h) || a == canonicalSHA([][]string{{"11", "Kab  Aceh"}}, h) {
		t.Fatal("hash kanonis salah")
	}
}

// Angka yang sama dengan penulisan berbeda (PHP menulis float kecil sebagai 1.8E-5) dianggap identik; teks tidak.
func TestCanonicalSHANumericColumnsCompareByValue(t *testing.T) {
	h := []string{"code", "latitude"}
	if canonicalSHA([][]string{{"11", "0.000018"}}, h) != canonicalSHA([][]string{{"11", "1.8E-5"}}, h) {
		t.Fatal("0.000018 dan 1.8E-5 harus identik")
	}
	if canonicalSHA([][]string{{"11", "0.000018"}}, h) == canonicalSHA([][]string{{"11", "0.000019"}}, h) {
		t.Fatal("nilai berbeda harus berbeda")
	}
	if canonicalSHA([][]string{{"1.8E-5", "0"}}, h) == canonicalSHA([][]string{{"0.000018", "0"}}, h) {
		t.Fatal("kolom teks (code) tidak boleh dinormalisasi")
	}
}
