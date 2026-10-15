package dataset_test

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/caturseptian/go-indonesia-territory/internal/dataset"
	"github.com/caturseptian/go-indonesia-territory/internal/datasettest"
)

func TestReadTableReturnsRecordsWithoutHeader(t *testing.T) {
	fsys := datasettest.Build(t, map[string][][]string{
		"provinces": {{"11", "Aceh", "", "", "", ""}, {"31", "DKI", "", "", "", ""}},
	})
	m, _, err := dataset.ReadManifest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := dataset.ReadTable(fsys, m, "provinces")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1][0] != "31" || rows[1][1] != "DKI" {
		t.Fatalf("rows = %q", rows)
	}
}

func TestReadTableRejectsTamperedData(t *testing.T) {
	cases := map[string]func(fstest.MapFS, *dataset.Manifest){
		"sha salah": func(fsys fstest.MapFS, m *dataset.Manifest) {
			f := m.Files["provinces"]
			f.SHA256 = "00"
			m.Files["provinces"] = f
		},
		"jumlah baris salah": func(fsys fstest.MapFS, m *dataset.Manifest) {
			f := m.Files["provinces"]
			f.Rows = 3
			m.Files["provinces"] = f
		},
		"berkas hilang": func(fsys fstest.MapFS, m *dataset.Manifest) { delete(fsys, "provinces.csv.gz") },
	}
	for name, tamper := range cases {
		t.Run(name, func(t *testing.T) {
			fsys := datasettest.Build(t, map[string][][]string{"provinces": {{"11", "Aceh", "", "", "", ""}}})
			m, _, err := dataset.ReadManifest(fsys)
			if err != nil {
				t.Fatal(err)
			}
			tamper(fsys, &m)
			if _, err := dataset.ReadTable(fsys, m, "provinces"); !errors.Is(err, dataset.ErrCorrupt) {
				t.Fatalf("err = %v, mau ErrCorrupt", err)
			}
		})
	}
}

func TestReadTableRejectsWrongHeader(t *testing.T) {
	fsys := datasettest.Build(t, nil)
	fsys["provinces.csv.gz"] = fsys["regencies.csv.gz"]
	m, _, err := dataset.ReadManifest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	f := m.Files["provinces"]
	f.SHA256 = m.Files["regencies"].SHA256
	m.Files["provinces"] = f
	if _, err := dataset.ReadTable(fsys, m, "provinces"); !errors.Is(err, dataset.ErrCorrupt) {
		t.Fatalf("err = %v, mau ErrCorrupt", err)
	}
}

func TestReadManifestRequiresEveryTable(t *testing.T) {
	fsys := fstest.MapFS{"manifest.json": {Data: []byte(`{"dataset_version":"x","files":{}}`)}}
	if _, _, err := dataset.ReadManifest(fsys); !errors.Is(err, dataset.ErrCorrupt) {
		t.Fatalf("err = %v, mau ErrCorrupt", err)
	}
}

func TestEmbeddedDataIsComplete(t *testing.T) {
	fsys, err := dataset.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	m, sum, err := dataset.ReadManifest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum) != 64 || m.DatasetVersion == "" {
		t.Fatalf("manifest: %q %q", m.DatasetVersion, sum)
	}
	want := map[string]int{"provinces": 38, "regencies": 514, "districts": 7285, "villages": 83762, "islands": 17374}
	for _, tb := range dataset.Tables() {
		rows, err := dataset.ReadTable(fsys, m, tb.Name)
		if err != nil {
			t.Fatalf("%s: %v", tb.Name, err)
		}
		if n, ok := want[tb.Name]; ok && len(rows) != n {
			t.Fatalf("%s: %d baris, mau %d", tb.Name, len(rows), n)
		}
		if len(rows) == 0 {
			t.Fatalf("%s kosong", tb.Name)
		}
	}
}

func TestTablesReturnsCopies(t *testing.T) {
	a := dataset.Tables()
	a[0].Columns[0] = "diubah"
	if dataset.Tables()[0].Columns[0] != "code" {
		t.Fatal("Tables() membocorkan slice internal")
	}
}

func TestReadTableRejectsNegativeRowCount(t *testing.T) {
	fsys := datasettest.Build(t, nil)
	m, _, err := dataset.ReadManifest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	f := m.Files["provinces"]
	f.Rows = -1
	m.Files["provinces"] = f
	if _, err := dataset.ReadTable(fsys, m, "provinces"); !errors.Is(err, dataset.ErrCorrupt) {
		t.Fatalf("err = %v, mau ErrCorrupt (bukan panic)", err)
	}
}
