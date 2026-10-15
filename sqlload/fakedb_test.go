package sqlload

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// fakeDB merekam setiap pernyataan; existing berisi tabel yang dianggap sudah ada; failOn membuat pernyataan yang
// mengandung teks itu gagal.
type fakeDB struct {
	mu       sync.Mutex
	log      []string
	args     [][]driver.NamedValue
	existing map[string]bool
	failOn   string
}

func (f *fakeDB) record(q string, args []driver.NamedValue) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, q)
	f.args = append(f.args, args)
	if f.failOn != "" && strings.Contains(q, f.failOn) {
		return errors.New("gagal disengaja")
	}
	return nil
}

func (f *fakeDB) Open(string) (driver.Conn, error) { return &fakeConn{f}, nil }

type fakeConn struct{ f *fakeDB }

func (c *fakeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("Prepare tidak dipakai")
}
func (c *fakeConn) Close() error              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error) { _ = c.f.record("BEGIN", nil); return fakeTx{c.f}, nil }
func (c *fakeConn) ExecContext(_ context.Context, q string, a []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), c.f.record(q, a)
}
func (c *fakeConn) QueryContext(_ context.Context, q string, a []driver.NamedValue) (driver.Rows, error) {
	if err := c.f.record(q, a); err != nil {
		return nil, err
	}
	if len(a) == 1 && c.f.existing[a[0].Value.(string)] {
		return &fakeRows{n: 1}, nil
	}
	return &fakeRows{}, nil
}

type fakeTx struct{ f *fakeDB }

func (t fakeTx) Commit() error   { return t.f.record("COMMIT", nil) }
func (t fakeTx) Rollback() error { return t.f.record("ROLLBACK", nil) }

type fakeRows struct{ n int }

func (r *fakeRows) Columns() []string { return []string{"x"} }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.n == 0 {
		return io.EOF
	}
	r.n--
	dest[0] = int64(1)
	return nil
}

func openFake(t *testing.T, f *fakeDB) *sql.DB {
	t.Helper()
	name := "fake-" + t.Name()
	sql.Register(name, f)
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}
