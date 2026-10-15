package territory

import "errors"

var (
	// ErrInvalidCode: masukan tidak berformat kode Kepmendagri.
	ErrInvalidCode = errors.New("territory: kode tidak berformat Kepmendagri")
	// ErrWrongLevel: kode sah tetapi untuk tingkat wilayah lain.
	ErrWrongLevel = errors.New("territory: kode berada di tingkat lain")
	// ErrNotFound: kode sah tetapi tidak ada di dataset.
	ErrNotFound = errors.New("territory: kode tidak ada di dataset")
	// ErrDataCorrupt: data tertanam gagal diverifikasi atau dibaca.
	ErrDataCorrupt = errors.New("territory: data tertanam rusak")
)
