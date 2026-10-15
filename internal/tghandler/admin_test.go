package tghandler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestNewGroupIgnoreAdminAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name    string
		result  string
		allowed bool
	}{
		{"administrator", `{"ok":true,"result":[{"user":{"id":42},"status":"administrator"}]}`, true},
		{"creator", `{"ok":true,"result":[{"user":{"id":42},"status":"creator"}]}`, true},
		{"non-admin", `{"ok":true,"result":[{"user":{"id":99},"status":"creator"}]}`, false},
		{"API failure", `{"ok":false,"error_code":403,"description":"Forbidden"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lookups := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/bottest/getMe":
					fmt.Fprint(w, `{"ok":true,"result":{"id":1,"is_bot":true,"username":"nafanya"}}`)
				case "/bottest/getChatAdministrators":
					lookups++
					if err := r.ParseForm(); err != nil {
						t.Error(err)
					}
					if r.Form.Get("chat_id") != "-100" {
						t.Errorf("wrong chat: %v", r.Form)
					}
					fmt.Fprint(w, tc.result)
				default:
					t.Errorf("unexpected Telegram call: %s", r.URL.Path)
					http.Error(w, "unexpected request", 500)
				}
			}))
			defer server.Close()
			bot, err := tgbotapi.NewBotAPIWithClient("test", server.URL+"/bot%s/%s", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			h := &Handler{bot: bot} // No registered/cached chats: the first-ever group command.
			m := ignoreMessage("/ignoreuser 99")
			store := &memoryIgnoreStore{users: map[[2]int64]bool{}}
			f := ignoreFilter{store: store, isAdmin: func() bool { return h.isChatAdmin(tgbotapi.Update{Message: m}) }, reply: func(string) {}}
			if !f.handle(m) {
				t.Fatal("command not handled")
			}
			if lookups != 1 {
				t.Fatalf("expected live admin lookup, got %d", lookups)
			}
			if store.users[[2]int64{-100, 99}] != tc.allowed {
				t.Fatal("wrong authorization outcome")
			}
		})
	}
}

func TestChatAdminInvalidSendersAndPrivateChat(t *testing.T) {
	h := &Handler{} // No bot: these cases must not make API calls.
	for _, m := range []*tgbotapi.Message{nil, {}, {Chat: &tgbotapi.Chat{Type: "group"}}, {Chat: &tgbotapi.Chat{Type: "group"}, From: &tgbotapi.User{ID: 42}, SenderChat: &tgbotapi.Chat{ID: -100}}} {
		if h.isChatAdmin(tgbotapi.Update{Message: m}) {
			t.Fatal("unidentifiable sender authorized")
		}
	}
	m := ignoreMessage("/chatConfigure")
	m.Chat.Type = "private"
	if !h.isChatAdmin(tgbotapi.Update{Message: m}) {
		t.Fatal("private-chat access changed")
	}
}

func TestSenderlessGroupMessageIsSilentlyDropped(t *testing.T) {
	f := ignoreFilter{reply: func(string) { t.Fatal("ordinary senderless message must not receive a reply") }}
	m := ignoreMessage("hello")
	m.From = nil
	if !f.handle(m) {
		t.Fatal("senderless message reached normal dispatch")
	}
}
