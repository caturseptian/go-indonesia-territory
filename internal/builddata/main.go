// Perintah builddata membangun internal/dataset/data dari Indonesia Region API. Hanya untuk rilis data.
//
//	go run ./internal/builddata [-base URL] [-out DIR]     tulis data
//	go run ./internal/builddata -check                     gagal bila data di repo berbeda dari API
//	go run ./internal/builddata -compare DIR               bandingkan isi dengan folder data lain (mis. paket Laravel)
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	base := flag.String("base", "https://indonesia-region.caturseptian.site", "alamat Indonesia Region API")
	out := flag.String("out", "internal/dataset/data", "folder data")
	check := flag.Bool("check", false, "hanya periksa, jangan tulis")
	compare := flag.String("compare", "", "folder data lain untuk dibandingkan isinya")
	flag.Parse()
	if err := run(*base, *out, *check, *compare); err != nil {
		fmt.Fprintln(os.Stderr, "builddata:", err)
		os.Exit(1)
	}
}

func run(base, out string, check bool, compare string) error {
	if compare != "" {
		report, err := compareDirs(os.DirFS(out), os.DirFS(compare))
		fmt.Print(report)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	files, err := build(ctx, &http.Client{Timeout: 10 * time.Minute}, base)
	if err != nil {
		return err
	}
	if check {
		if err := checkDir(files, os.DirFS(out)); err != nil {
			return fmt.Errorf("%s: %w; jalankan tanpa -check", out, err)
		}
		fmt.Println("sama")
		return nil
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(out, name), data, 0o644); err != nil {
			return err
		}
	}
	fmt.Println("selesai")
	return nil
}

// checkDir membandingkan hasil build dengan folder data menurut isinya: CSV dibandingkan setelah gzip dibuka dan
// manifest tanpa sha256, karena byte gzip bisa berubah antarversi Go meski isinya sama.
func checkDir(files map[string][]byte, cur fs.FS) error {
	for name, want := range files {
		got, err := fs.ReadFile(cur, name)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		switch {
		case strings.HasSuffix(name, ".gz"):
			w, err1 := gunzip(want)
			g, err2 := gunzip(got)
			if err1 != nil || err2 != nil || !bytes.Equal(w, g) {
				return fmt.Errorf("isi %s berbeda", name)
			}
		case name == "manifest.json":
			w, err1 := withoutHashes(want)
			g, err2 := withoutHashes(got)
			if err1 != nil || err2 != nil || !bytes.Equal(w, g) {
				return fmt.Errorf("%s berbeda", name)
			}
		default:
			if !bytes.Equal(want, got) {
				return fmt.Errorf("%s berbeda", name)
			}
		}
	}
	return nil
}

func gunzip(b []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(zr)
}

// withoutHashes menghapus semua kunci sha256 lalu menyusun ulang JSON dengan urutan kunci tetap.
func withoutHashes(b []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	var strip func(any)
	strip = func(x any) {
		if m, ok := x.(map[string]any); ok {
			delete(m, "sha256")
			for _, c := range m {
				strip(c)
			}
		}
	}
	strip(v)
	return json.Marshal(v)
}
