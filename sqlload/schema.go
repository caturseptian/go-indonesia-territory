package sqlload

type colType int

const (
	tVarchar colType = iota
	tDecimal
	tJSON
	tBool
	tInt
)

type column struct {
	name             string
	typ              colType
	size             int
	precision, scale int
	nullable         bool
	extra            Extras // 0 = selalu ada
	source           string // kolom CSV sumber; kosong = diisi khusus (seq)
}

type foreignKey struct{ column, refTable, refColumn string }

type table struct {
	name        string
	level       Level  // tabel ikut bila level <= Options.Level
	extra       Extras // 0 = selalu ikut
	dataset     string // nama tabel data tertanam; kosong = meta
	columns     []column
	primaryKey  []string
	foreignKeys []foreignKey
	indexes     [][]string
}

func spatialCols() []column {
	return []column{
		{name: "latitude", typ: tDecimal, precision: 9, scale: 6, nullable: true, extra: Coordinates, source: "latitude"},
		{name: "longitude", typ: tDecimal, precision: 9, scale: 6, nullable: true, extra: Coordinates, source: "longitude"},
		{name: "spatial_status", typ: tVarchar, size: 16, nullable: true, extra: Coordinates, source: "spatial_status"},
		{name: "bbox", typ: tJSON, nullable: true, extra: Coordinates, source: "bbox"},
	}
}

func cols(base []column, more ...[]column) []column {
	out := append([]column{}, base...)
	for _, m := range more {
		out = append(out, m...)
	}
	return out
}

// schema: deklarasi tunggal untuk semua dialek, urut induk → anak (ukuran sama dengan laravel-indonesia-territory).
func schema() []table {
	return []table{
		{name: "provinces", level: Province, dataset: "provinces", primaryKey: []string{"code"},
			columns: cols([]column{
				{name: "code", typ: tVarchar, size: 2, source: "code"},
				{name: "name", typ: tVarchar, size: 255, source: "name"},
			}, spatialCols())},
		{name: "regencies", level: Regency, dataset: "regencies", primaryKey: []string{"code"},
			foreignKeys: []foreignKey{{"province_code", "provinces", "code"}}, indexes: [][]string{{"province_code"}, {"name"}},
			columns: cols([]column{
				{name: "code", typ: tVarchar, size: 5, source: "code"},
				{name: "province_code", typ: tVarchar, size: 2, source: "province_code"},
				{name: "type", typ: tVarchar, size: 9, source: "type"},
				{name: "name", typ: tVarchar, size: 255, source: "name"},
			}, spatialCols())},
		{name: "districts", level: District, dataset: "districts", primaryKey: []string{"code"},
			foreignKeys: []foreignKey{{"regency_code", "regencies", "code"}}, indexes: [][]string{{"regency_code"}, {"name"}},
			columns: cols([]column{
				{name: "code", typ: tVarchar, size: 8, source: "code"},
				{name: "regency_code", typ: tVarchar, size: 5, source: "regency_code"},
				{name: "name", typ: tVarchar, size: 255, source: "name"},
			}, spatialCols())},
		{name: "villages", level: Village, dataset: "villages", primaryKey: []string{"code"},
			foreignKeys: []foreignKey{{"district_code", "districts", "code"}}, indexes: [][]string{{"district_code"}, {"name"}, {"postal_code"}},
			columns: cols([]column{
				{name: "code", typ: tVarchar, size: 13, source: "code"},
				{name: "district_code", typ: tVarchar, size: 8, source: "district_code"},
				{name: "type", typ: tVarchar, size: 9, source: "type"},
				{name: "name", typ: tVarchar, size: 255, source: "name"},
				{name: "postal_code", typ: tVarchar, size: 5, nullable: true, extra: PostalCodes, source: "postal_code"},
			}, spatialCols())},
		{name: "islands", level: Regency, extra: Islands, dataset: "islands", primaryKey: []string{"code"},
			foreignKeys: []foreignKey{{"province_code", "provinces", "code"}, {"regency_code", "regencies", "code"}},
			indexes:     [][]string{{"province_code"}, {"regency_code"}, {"name"}},
			columns: []column{
				{name: "code", typ: tVarchar, size: 11, source: "code"},
				{name: "name", typ: tVarchar, size: 255, source: "name"},
				{name: "province_code", typ: tVarchar, size: 2, source: "province_code"},
				{name: "regency_code", typ: tVarchar, size: 5, nullable: true, source: "regency_code"},
				{name: "latitude", typ: tDecimal, precision: 9, scale: 6, nullable: true, source: "latitude"},
				{name: "longitude", typ: tDecimal, precision: 9, scale: 6, nullable: true, source: "longitude"},
				{name: "area_km2", typ: tDecimal, precision: 12, scale: 4, nullable: true, source: "area_km2"},
				{name: "inhabited", typ: tBool, nullable: true, source: "inhabited"},
			}},
		{name: "code_changes", level: Province, extra: CodeChanges, dataset: "code_changes", primaryKey: []string{"seq"},
			indexes: [][]string{{"code"}},
			columns: []column{
				{name: "seq", typ: tInt},
				{name: "from_document", typ: tVarchar, size: 64, source: "from_document"},
				{name: "to_document", typ: tVarchar, size: 64, source: "to_document"},
				{name: "level", typ: tVarchar, size: 8, source: "level"},
				{name: "change", typ: tVarchar, size: 8, source: "change"},
				{name: "code", typ: tVarchar, size: 13, source: "code"},
				{name: "old_name", typ: tVarchar, size: 255, nullable: true, source: "old_name"},
				{name: "new_name", typ: tVarchar, size: 255, nullable: true, source: "new_name"},
			}},
		{name: "meta", level: Province, primaryKey: []string{"key"},
			columns: []column{
				{name: "key", typ: tVarchar, size: 64},
				{name: "value", typ: tVarchar, size: 255},
			}},
	}
}

// selected: tabel dan kolom yang ikut untuk opsi ini, urut pembuatan.
func selected(o Options) []table {
	var out []table
	for _, t := range schema() {
		if t.level > o.Level || (t.extra != 0 && o.With&t.extra == 0) {
			continue
		}
		var keep []column
		for _, c := range t.columns {
			if c.extra == 0 || o.With&c.extra != 0 {
				keep = append(keep, c)
			}
		}
		t.columns = keep
		var idx [][]string
		for _, ix := range t.indexes {
			if hasColumn(t, ix[0]) {
				idx = append(idx, ix)
			}
		}
		t.indexes = idx
		out = append(out, t)
	}
	return out
}

func hasColumn(t table, name string) bool {
	for _, c := range t.columns {
		if c.name == name {
			return true
		}
	}
	return false
}
