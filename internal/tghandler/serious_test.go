package tghandler

import (
	"sync"
	"testing"
	"time"
)

func newSeriousHandler() *Handler {
	return &Handler{seriousMsgs: make(map[int64]map[int]time.Time)}
}

func TestWasSeriousRemembersRecordedMessages(t *testing.T) {
	h := newSeriousHandler()
	h.rememberSerious(10, 100)

	if !h.wasSerious(10, 100) {
		t.Error("a recorded message should be reported as serious")
	}
	if h.wasSerious(10, 101) {
		t.Error("an unrecorded message in the same chat should not be serious")
	}
	if h.wasSerious(20, 100) {
		t.Error("the same message id in another chat should not be serious")
	}
}

func TestWasSeriousExpires(t *testing.T) {
	h := newSeriousHandler()
	h.seriousMsgs[10] = map[int]time.Time{
		100: time.Now().Add(-seriousMemory - time.Minute),
		101: time.Now(),
	}

	if h.wasSerious(10, 100) {
		t.Error("an entry older than seriousMemory should no longer count")
	}
	if !h.wasSerious(10, 101) {
		t.Error("a fresh entry should still count")
	}
}

func TestRememberSeriousPrunesExpiredEntries(t *testing.T) {
	h := newSeriousHandler()
	h.seriousMsgs[10] = map[int]time.Time{
		1: time.Now().Add(-seriousMemory - time.Hour),
		2: time.Now().Add(-seriousMemory - time.Minute),
	}

	h.rememberSerious(10, 3)

	if got := len(h.seriousMsgs[10]); got != 1 {
		t.Errorf("expired entries should be dropped on write, map holds %d entries", got)
	}
	if !h.wasSerious(10, 3) {
		t.Error("the newly recorded message should be serious")
	}
}

func TestSeriousTrackingIsConcurrencySafe(t *testing.T) {
	h := newSeriousHandler()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(n int) { defer wg.Done(); h.rememberSerious(int64(n%3), n) }(i)
		go func(n int) { defer wg.Done(); h.wasSerious(int64(n%3), n) }(i)
	}
	wg.Wait()
}
