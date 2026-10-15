// Package territory menyediakan kode dan data wilayah administrasi Indonesia (Kepmendagri): provinsi, kabupaten/kota,
// kecamatan, desa/kelurahan, kode pos, koordinat, pulau, dan riwayat perubahan kode.
//
// Data tertanam di binary dan dimuat sekali saat pertama dipakai. Semua fungsi murni: masukan yang sama selalu
// menghasilkan keluaran yang sama, hasil berupa salinan, dan galat dikembalikan secara eksplisit.
//
//	p, err := territory.Province("31")
//	villages, err := territory.VillagesOf("31.71.01")
//
// Kode harus persis berformat Kepmendagri (NN, NN.NN, NN.NN.NN, NN.NN.NN.NNNN); masukan lain ditolak dengan
// ErrInvalidCode tanpa dinormalisasi.
package territory
