package tghandler

import (
	"errors"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type memoryIgnoreStore struct {
	users map[[2]int64]bool
	err   error
}

func (s *memoryIgnoreStore) IsUserIgnored(c, u int64) (bool, error) {
	return s.users[[2]int64{c, u}], s.err
}
func (s *memoryIgnoreStore) SetUserIgnored(c, u int64, b bool) error {
	if s.err != nil {
		return s.err
	}
	s.users[[2]int64{c, u}] = b
	return nil
}
func (s *memoryIgnoreStore) ListIgnoredUsers(c int64) ([]int64, error) {
	var ids []int64
	for k, v := range s.users {
		if k[0] == c && v {
			ids = append(ids, k[1])
		}
	}
	return ids, s.err
}
func ignoreMessage(text string) *tgbotapi.Message {
	m := &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: -100, Type: "supergroup"}, From: &tgbotapi.User{ID: 42}, Text: text}
	if strings.HasPrefix(text, "/") {
		m.Entities = []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: len(strings.Fields(text)[0])}}
	}
	return m
}
func TestSelfServiceIgnore(t *testing.T) {
	s := &memoryIgnoreStore{users: map[[2]int64]bool{}}
	var replies []string
	f := ignoreFilter{store: s, botUsername: "nafanya", isAdmin: func() bool { t.Fatal("self-service must not require admin lookup"); return false }, reply: func(s string) { replies = append(replies, s) }}
	if !f.handle(ignoreMessage("/ignoreme@NaFaNyA")) || !s.users[[2]int64{-100, 42}] {
		t.Fatal("opt-out failed")
	}
	for _, text := range []string{"Нафаня hello", "/start", "https://x.com/test/status/1", "a random sufficiently long message"} {
		if !f.handle(ignoreMessage(text)) {
			t.Fatalf("ignored message escaped: %s", text)
		}
	}
	m := ignoreMessage("hello")
	m.Chat.ID = -200
	if f.handle(m) {
		t.Fatal("ignore leaked into another group")
	}
	m = ignoreMessage("Нафаня hello")
	m.From.ID = 99
	m.ReplyToMessage = ignoreMessage("my message")
	if f.handle(m) {
		t.Fatal("human replying to ignored user should pass")
	}
	if !f.handle(ignoreMessage("/unignoreme")) || s.users[[2]int64{-100, 42}] {
		t.Fatal("opt-in failed while ignored")
	}
	if f.handle(ignoreMessage("hello")) {
		t.Fatal("opted-in user blocked")
	}
	if len(replies) != 2 {
		t.Fatalf("unexpected replies: %v", replies)
	}
}
func TestIgnorePermissionsAndValidation(t *testing.T) {
	cases := []struct {
		name, text string
		admin      bool
		reply      *tgbotapi.Message
		wantID     int64
		ignored    bool
	}{
		{name: "self", text: "/ignoreme", wantID: 42, ignored: true},
		{name: "self cannot target", text: "/ignoreme 99"},
		{name: "non-admin cannot add", text: "/ignoreuser 99"},
		{name: "non-admin cannot remove", text: "/unignoreuser 99"},
		{name: "admin ID", text: "/ignoreuser 99", admin: true, wantID: 99, ignored: true},
		{name: "admin reply", text: "/ignoreuser", admin: true, reply: &tgbotapi.Message{From: &tgbotapi.User{ID: 99}}, wantID: 99, ignored: true},
		{name: "admin remove", text: "/unignoreuser 99", admin: true, wantID: 99},
		{name: "negative ID", text: "/ignoreuser -99", admin: true},
		{name: "zero ID", text: "/ignoreuser 0", admin: true},
		{name: "username", text: "/ignoreuser @someone", admin: true},
		{name: "overflow", text: "/ignoreuser 999999999999999999999", admin: true},
		{name: "missing target", text: "/ignoreuser", admin: true},
		{name: "anonymous reply", text: "/ignoreuser", admin: true, reply: &tgbotapi.Message{From: &tgbotapi.User{ID: 99}, SenderChat: &tgbotapi.Chat{ID: -100}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &memoryIgnoreStore{users: map[[2]int64]bool{}}
			if tc.name == "admin remove" || tc.name == "non-admin cannot remove" {
				s.users[[2]int64{-100, 99}] = true
			}
			f := ignoreFilter{store: s, isAdmin: func() bool { return tc.admin }, reply: func(string) {}}
			m := ignoreMessage(tc.text)
			m.ReplyToMessage = tc.reply
			if !f.handle(m) {
				t.Fatal("command not handled")
			}
			if tc.wantID != 0 {
				if got, ok := s.users[[2]int64{-100, tc.wantID}]; !ok || got != tc.ignored {
					t.Fatal("wrong persisted setting")
				}
			} else if tc.name == "non-admin cannot remove" {
				if !s.users[[2]int64{-100, 99}] {
					t.Fatal("non-admin removed another user")
				}
			} else if len(s.users) != 0 {
				t.Fatal("unauthorized/invalid mutation")
			}
		})
	}
}
func TestIgnoreScopeAndErrors(t *testing.T) {
	s := &memoryIgnoreStore{users: map[[2]int64]bool{}}
	var replies []string
	f := ignoreFilter{store: s, botUsername: "nafanya", isAdmin: func() bool { return false }, reply: func(s string) { replies = append(replies, s) }}
	m := ignoreMessage("/ignoreme")
	m.Chat.Type = "private"
	if f.handle(m) || len(s.users) > 0 {
		t.Fatal("private chat changed")
	}
	if f.handle(ignoreMessage("/ignoreme@anotherbot")) || len(s.users) > 0 {
		t.Fatal("command for another bot consumed")
	}
	m = ignoreMessage("/ignoreme")
	m.SenderChat = &tgbotapi.Chat{ID: -100}
	if !f.handle(m) || len(s.users) > 0 {
		t.Fatal("anonymous sender persisted")
	}
	m = ignoreMessage("/ignoreme")
	m.From = nil
	if !f.handle(m) || len(s.users) > 0 {
		t.Fatal("missing sender persisted")
	}
	s.err = errors.New("database unavailable")
	replies = nil
	if !f.handle(ignoreMessage("hello")) || len(replies) > 0 {
		t.Fatal("lookup failure must silently stop dispatch")
	}
	f.handle(ignoreMessage("/ignoreme"))
	if len(s.users) > 0 || len(replies) != 1 || !strings.Contains(replies[0], "try again") {
		t.Fatal("write failure not reported")
	}
}
func TestIgnoredListAdminOnlyAndChunked(t *testing.T) {
	s := &memoryIgnoreStore{users: map[[2]int64]bool{}}
	for id := int64(1); id <= 1000; id++ {
		s.users[[2]int64{-100, id}] = true
	}
	var replies []string
	admin := false
	f := ignoreFilter{store: s, isAdmin: func() bool { return admin }, reply: func(s string) { replies = append(replies, s) }}
	f.handle(ignoreMessage("/ignoredusers"))
	if len(replies) != 1 || !strings.Contains(replies[0], "Only group admins") {
		t.Fatal("list exposed")
	}
	admin = true
	replies = nil
	f.handle(ignoreMessage("/ignoredusers"))
	if len(replies) < 2 {
		t.Fatal("large list not chunked")
	}
	for _, r := range replies {
		if len(r) > 3500 {
			t.Fatal("reply too large")
		}
	}
}
