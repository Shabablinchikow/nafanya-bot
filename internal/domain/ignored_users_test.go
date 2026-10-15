package domain

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// This test uses real PostgreSQL when supplied a disposable database DSN.
// A unique table prefix isolates each run; only those tables are cleaned up.
func TestIgnoredUsersPostgres(t *testing.T) {
	dsn := os.Getenv("NAFANYA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set NAFANYA_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	config := &gorm.Config{NamingStrategy: schema.NamingStrategy{TablePrefix: fmt.Sprintf("ignore_test_%d_", time.Now().UnixNano())}}
	open := func() *Handler {
		h, err := NewHandler(dsn, config, 1)
		if err != nil {
			t.Fatal(err)
		}
		db, err := h.db.DB()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		return h
	}
	h := open()
	t.Cleanup(func() {
		if err := h.db.Migrator().DropTable(&IgnoredUser{}, &BotConfig{}, &Chat{}); err != nil {
			t.Error(err)
		}
	})
	if ids, err := h.ListIgnoredUsers(-100); err != nil || len(ids) != 0 {
		t.Fatalf("initial list: %v %v", ids, err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.SetUserIgnored(-100, 42, true); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if err := h.SetUserIgnored(-100, 99, true); err != nil {
		t.Fatal(err)
	}
	if err := h.SetUserIgnored(-200, 42, true); err != nil {
		t.Fatal(err)
	}
	// A new handler reruns startup migrations and reads persisted data without cache.
	reopened := open()
	if ids, err := reopened.ListIgnoredUsers(-100); err != nil || !reflect.DeepEqual(ids, []int64{42, 99}) {
		t.Fatalf("persisted list: %v %v", ids, err)
	}
	if ignored, err := reopened.IsUserIgnored(-300, 42); err != nil || ignored {
		t.Fatal("scope leaked", err)
	}
	for i := 0; i < 2; i++ {
		if err := reopened.SetUserIgnored(-100, 42, false); err != nil {
			t.Fatal(err)
		}
	}
	if ignored, err := h.IsUserIgnored(-100, 42); err != nil || ignored {
		t.Fatal("opt-in failed", err)
	}
	if ignored, err := h.IsUserIgnored(-200, 42); err != nil || !ignored {
		t.Fatal("other group changed", err)
	}
	if err := h.SetUserIgnored(-100, 42, true); err != nil {
		t.Fatal("re-add failed", err)
	}
}
