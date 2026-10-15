package territory

// Coordinates adalah titik pusat wilayah (WGS84).
type Coordinates struct{ Latitude, Longitude float64 }

// BBox adalah kotak batas [MinLon, MinLat, MaxLon, MaxLat].
type BBox struct{ MinLon, MinLat, MaxLon, MaxLat float64 }

// Spatial: data spasial wilayah. Center dan BBox nil bila tidak tersedia; Status "available" atau "not_available".
type Spatial struct {
	Center *Coordinates
	BBox   *BBox
	Status string
}

// RegencyType: kabupaten atau kota.
type RegencyType string

const (
	RegencyKabupaten RegencyType = "kabupaten"
	RegencyKota      RegencyType = "kota"
)

// VillageType: desa, kelurahan, atau desa adat.
type VillageType string

const (
	VillageDesa      VillageType = "desa"
	VillageKelurahan VillageType = "kelurahan"
	VillageDesaAdat  VillageType = "desa_adat"
)

// ChangeKind: jenis perubahan kode antardokumen.
type ChangeKind string

const (
	ChangeAdded   ChangeKind = "added"
	ChangeRemoved ChangeKind = "removed"
	ChangeRenamed ChangeKind = "renamed"
)

// Province: provinsi.
type Province struct {
	Code, Name string
	Spatial    Spatial
}

// Regency: kabupaten atau kota; Type membedakan keduanya.
type Regency struct {
	Code, ProvinceCode string
	Type               RegencyType
	Name               string
	Spatial            Spatial
}

// District: kecamatan (atau distrik di Papua).
type District struct {
	Code, RegencyCode, Name string
	Spatial                 Spatial
}

// Village: desa atau kelurahan; Type membedakan keduanya.
type Village struct {
	Code, DistrictCode string
	Type               VillageType
	Name, PostalCode   string // PostalCode kosong bila tidak ada padanan pasti di data Pos Indonesia
	Spatial            Spatial
}

// Island: pulau. RegencyCode kosong bila pulau tidak tercatat di kabupaten/kota mana pun.
type Island struct {
	Code, Name, ProvinceCode, RegencyCode string
	Center                                *Coordinates
	AreaKm2                               *float64
	Inhabited                             *bool
}

// CodeChange: satu perubahan kode antara dua dokumen (OldName kosong untuk added, NewName kosong untuk removed).
type CodeChange struct {
	FromDocument, ToDocument string
	Level                    Level
	Change                   ChangeKind
	Code, OldName, NewName   string
}

// VersionInfo menjelaskan dataset tertanam.
type VersionInfo struct {
	Dataset, EffectiveDate, Source, ManifestSHA256 string
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func (s Spatial) clone() Spatial {
	return Spatial{Center: clonePtr(s.Center), BBox: clonePtr(s.BBox), Status: s.Status}
}

func (p Province) clone() Province { p.Spatial = p.Spatial.clone(); return p }
func (r Regency) clone() Regency   { r.Spatial = r.Spatial.clone(); return r }
func (d District) clone() District { d.Spatial = d.Spatial.clone(); return d }
func (v Village) clone() Village   { v.Spatial = v.Spatial.clone(); return v }
func (i Island) clone() Island {
	i.Center, i.AreaKm2, i.Inhabited = clonePtr(i.Center), clonePtr(i.AreaKm2), clonePtr(i.Inhabited)
	return i
}

func cloneAll[T any](in []T, clone func(T) T) []T {
	out := make([]T, len(in))
	for i, v := range in {
		out[i] = clone(v)
	}
	return out
}
