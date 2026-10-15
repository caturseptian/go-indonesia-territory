package sqlload

import "strings"

// DDL merender CREATE TABLE dan CREATE INDEX untuk opsi ini; keluaran selalu sama untuk opsi yang sama.
func DDL(o Options) (string, error) {
	o, spec, err := normalize(o)
	if err != nil {
		return "", err
	}
	return strings.Join(ddlStatements(spec, o), ";\n\n") + ";\n", nil
}

// Tables mengembalikan nama tabel ber-prefix untuk opsi ini, urut pembuatan (hapus dengan urutan terbalik).
func Tables(o Options) ([]string, error) {
	o, _, err := normalize(o)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, t := range selected(o) {
		out = append(out, o.Prefix+t.name)
	}
	return out, nil
}

func ddlStatements(spec dialectSpec, o Options) []string {
	var out []string
	q := spec.quote
	for _, t := range selected(o) {
		var lines []string
		for _, c := range t.columns {
			l := "  " + q(c.name) + " " + spec.typeName(c)
			if !c.nullable {
				l += " NOT NULL"
			}
			lines = append(lines, l)
		}
		pk := make([]string, len(t.primaryKey))
		for i, k := range t.primaryKey {
			pk[i] = q(k)
		}
		lines = append(lines, "  PRIMARY KEY ("+strings.Join(pk, ", ")+")")
		for _, fk := range t.foreignKeys {
			if !hasColumn(t, fk.column) {
				continue
			}
			lines = append(lines, "  FOREIGN KEY ("+q(fk.column)+") REFERENCES "+q(o.Prefix+fk.refTable)+" ("+q(fk.refColumn)+") ON DELETE CASCADE")
		}
		out = append(out, "CREATE TABLE "+q(o.Prefix+t.name)+" (\n"+strings.Join(lines, ",\n")+"\n)"+spec.tableSuffix)
		for _, ix := range t.indexes {
			cs := make([]string, len(ix))
			for i, c := range ix {
				cs[i] = q(c)
			}
			name := o.Prefix + t.name + "_" + strings.Join(ix, "_") + "_idx"
			out = append(out, "CREATE INDEX "+q(name)+" ON "+q(o.Prefix+t.name)+" ("+strings.Join(cs, ", ")+")")
		}
	}
	return out
}
