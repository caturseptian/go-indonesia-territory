// Perintah builddata membangun internal/dataset/data dari Indonesia Region API. Hanya untuk rilis data.
//
//	go run ./internal/builddata [-base URL] [-out DIR]     tulis data
//	go run ./internal/builddata -check                     gagal bila data di repo berbeda dari API
//	go run ./internal/builddata -compare DIR               bandingkan isi dengan folder data lain (mis. paket Laravel)
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	for name, data := range files {
		path := filepath.Join(out, name)
		if check {
			cur, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(cur, data) {
				return fmt.Errorf("%s berbeda dari API; jalankan tanpa -check", path)
			}
			continue
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
	}
	fmt.Println("selesai")
	return nil
}
