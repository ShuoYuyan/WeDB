package handle_test

import (
	"testing"

	"github.com/wedb/wedb/drivers/odbc/handle"
)

func TestAllocEnv(t *testing.T) {
	h, e := handle.AllocEnv()
	if h == 0 || e == nil {
		t.Fatal("AllocEnv returned zero")
	}
	if e.Magic != handle.MagicEnv {
		t.Errorf("magic wrong: %x", e.Magic)
	}
	if got := handle.LookupEnv(h); got == nil {
		t.Errorf("LookupEnv failed")
	}
	if err := handle.FreeEnv(h); err != nil {
		t.Errorf("FreeEnv: %v", err)
	}
	if got := handle.LookupEnv(h); got != nil {
		t.Errorf("LookupEnv after free should be nil")
	}
}

func TestAllocDbc(t *testing.T) {
	eh, e := handle.AllocEnv()
	if eh == 0 {
		t.Fatal("AllocEnv zero")
	}
	dh, d, err := handle.AllocDbc(e)
	if err != nil {
		t.Fatalf("AllocDbc: %v", err)
	}
	if d.Magic != handle.MagicDbc {
		t.Errorf("dbc magic wrong")
	}
	if got := handle.LookupDbc(dh); got != d {
		t.Errorf("LookupDbc mismatch")
	}
	if err := handle.FreeDbc(dh); err != nil {
		t.Errorf("FreeDbc: %v", err)
	}
}

func TestAllocStmt(t *testing.T) {
	eh, e := handle.AllocEnv()
	_ = eh
	dh, d, _ := handle.AllocDbc(e)
	_ = dh
	sh, s, err := handle.AllocStmt(d)
	if err != nil {
		t.Fatalf("AllocStmt: %v", err)
	}
	if s.Magic != handle.MagicStmt {
		t.Errorf("stmt magic wrong")
	}
	if got := handle.LookupStmt(sh); got != s {
		t.Errorf("LookupStmt mismatch")
	}
	if err := handle.FreeStmt(sh); err != nil {
		t.Errorf("FreeStmt: %v", err)
	}
	_ = dh
}

func TestDiagPushAndNth(t *testing.T) {
	d := handle.NewDiag()
	d.Push("42000", 100, "syntax")
	d.Push("HY000", 200, "general")
	if d.Count() != 2 {
		t.Fatalf("Count=%d", d.Count())
	}
	rec, ok := d.Nth(1)
	if !ok {
		t.Fatal("Nth(1) failed")
	}
	if rec.SQLState != "HY000" || rec.NativeErr != 200 {
		t.Errorf("Nth(1)=%v", rec)
	}
	rec, ok = d.Nth(2)
	if !ok {
		t.Fatal("Nth(2) failed")
	}
	if rec.SQLState != "42000" {
		t.Errorf("Nth(2)=%v", rec)
	}
}

func TestBadHandleLookup(t *testing.T) {
	if got := handle.LookupEnv(0xDEADBEEF); got != nil {
		t.Errorf("expected nil for bad handle")
	}
	if got := handle.LookupDbc(0xDEADBEEF); got != nil {
		t.Errorf("expected nil for bad handle")
	}
	if got := handle.LookupStmt(0xDEADBEEF); got != nil {
		t.Errorf("expected nil for bad handle")
	}
}
