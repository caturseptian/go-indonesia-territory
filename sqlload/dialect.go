package sqlload

import (
	"fmt"
	"strconv"
)

// Dialect memilih SQL untuk database tertentu. Nilai yang sah hanya Postgres, MySQL, dan SQLite.
type Dialect struct{ name string }

// Dialect yang didukung.
var (
	Postgres = Dialect{name: "postgres"}
	MySQL    = Dialect{name: "mysql"} // juga MariaDB
	SQLite   = Dialect{name: "sqlite"}
)

func (d Dialect) String() string { return d.name }

type dialectSpec struct {
	quote            func(string) string
	placeholder      func(n int) string // n mulai 1
	maxParams        int
	transactionalDDL bool
	existsSQL        string // satu parameter: nama tabel
	tableSuffix      string
	typeName         func(column) string
	boolValue        func(bool) any
}

func dqQuote(s string) string { return `"` + s + `"` }

func commonType(c column, jsonType, boolType string) string {
	switch c.typ {
	case tVarchar:
		return fmt.Sprintf("VARCHAR(%d)", c.size)
	case tDecimal:
		return fmt.Sprintf("DECIMAL(%d,%d)", c.precision, c.scale)
	case tJSON:
		return jsonType
	case tBool:
		return boolType
	default:
		return "INTEGER"
	}
}

func (d Dialect) spec() (dialectSpec, bool) {
	switch d.name {
	case "postgres":
		return dialectSpec{
			quote:            dqQuote,
			placeholder:      func(n int) string { return "$" + strconv.Itoa(n) },
			maxParams:        65535,
			transactionalDDL: true,
			existsSQL:        "SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1",
			typeName:         func(c column) string { return commonType(c, "JSONB", "BOOLEAN") },
			boolValue:        func(b bool) any { return b },
		}, true
	case "mysql":
		return dialectSpec{
			quote:       func(s string) string { return "`" + s + "`" },
			placeholder: func(int) string { return "?" },
			maxParams:   65535,
			existsSQL:   "SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
			tableSuffix: " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",
			typeName:    func(c column) string { return commonType(c, "JSON", "TINYINT(1)") },
			boolValue: func(b bool) any {
				if b {
					return 1
				}
				return 0
			},
		}, true
	case "sqlite":
		return dialectSpec{
			quote:            dqQuote,
			placeholder:      func(int) string { return "?" },
			maxParams:        32766,
			transactionalDDL: true,
			existsSQL:        "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?",
			typeName:         func(c column) string { return commonType(c, "TEXT", "INTEGER") },
			boolValue: func(b bool) any {
				if b {
					return 1
				}
				return 0
			},
		}, true
	}
	return dialectSpec{}, false
}
