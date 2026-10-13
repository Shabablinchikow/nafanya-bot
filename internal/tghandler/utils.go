package tghandler

import (
	"crypto/rand"
	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/cfg"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
	"golang.org/x/exp/slices"
	"log"
	"math/big"
	"mvdan.cc/xurls/v2"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	Question           = 1
	RandomInterference = 2
)

const Serious = "с серьезным отношением"

type chatCache struct {
	lastRand        time.Time
	GoogleMaxTokens int
	OAIMaxTokens    int
}

// emotionList is a list of strings containing all available emotions
var emotionList = []string{
	"с нейтральным отношением",
	"с пессимизмом",
	"с оптимизмом",
	"с сарказмом",
	"с раздражением",
	"с жестким негативом",
}

func (h *Handler) isItTime(chat int64) bool {
	defer sentry.Recover()

	idx := slices.IndexFunc(h.chats, func(channel domain.Chat) bool {
		return channel.ID == chat
	})
	if idx == -1 {
		return false
	}

	if h.chats[idx].Type == domain.ChatTypePrivate {
		return false
	}

	nBig, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
	n := nBig.Int64()

	h.chatCacheMux.Lock()
	if _, ok := h.chatCache[chat]; !ok {
		newCache := chatCache{lastRand: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)}
		h.chatCache[chat] = newCache
	}

	agroLevel := int64(h.chats[idx].AgroLevel)
	cooldown := time.Duration(h.chats[idx].AgroCooldown)

	lastRand := h.chatCache[chat].lastRand
	h.chatCacheMux.Unlock()

	if n > (100-agroLevel) && time.Since(lastRand) > (cooldown*time.Minute) {
		h.chatCacheMux.Lock()
		newCache := chatCache{lastRand: time.Now()}
		h.chatCache[chat] = newCache
		h.chatCacheMux.Unlock()
		return true
	}

	return false
}

// isPhotoMessage reports whether a message carries a photo. Telegram sends a
// photo as a slice of sizes rather than a single field, so an empty slice means
// there is no image.
func isPhotoMessage(message *tgbotapi.Message) bool {
	return message != nil && len(message.Photo) > 0
}

func (h *Handler) isPersonal(update tgbotapi.Update) bool {
	if strings.HasPrefix(update.Message.Text, "Нафаня") || strings.HasPrefix(update.Message.Text, "нафаня") || strings.HasPrefix(update.Message.Text, "@grok") {
		return true
	} else if update.Message.ReplyToMessage != nil && !h.checkIfURLReply(update) {
		if update.Message.ReplyToMessage.From.ID != h.bot.Self.ID {
			return false
		}
		// A picture Nafanya just drew ends the exchange. Replies under it are
		// people reacting to the image, not asking him anything, so answering
		// every one of them turns a drawing into a thread. Addressing him by
		// name still works: that is handled above, before this branch.
		return !isPhotoMessage(update.Message.ReplyToMessage)
	}
	return false
}

func isDraw(update tgbotapi.Update) bool {
	return strings.Contains(update.Message.Text, "нарисуй") || strings.Contains(update.Message.Text, "Нарисуй")
}

func isBanana(update tgbotapi.Update) bool {
	return strings.Contains(update.Message.Text, "сгенерируй") || strings.Contains(update.Message.Text, "Сгенерируй")
}

func isDrawAny(update tgbotapi.Update) bool {
	return isDraw(update) || isBanana(update)
}

// seriousMemory bounds how long a serious answer keeps colouring replies to
// it. Long enough for a conversation to continue, short enough that the map
// cannot grow without limit. State lives in memory only, so a restart drops it
// and the thread reverts to the usual tone until someone says "серьёзно" again.
const seriousMemory = 6 * time.Hour

// rememberSerious records that one of Nafanya's own messages was a serious
// answer. Expired entries for the chat are dropped on the way in, which keeps
// the map bounded without a separate sweeper.
func (h *Handler) rememberSerious(chat int64, messageID int) {
	h.seriousMsgsMux.Lock()
	defer h.seriousMsgsMux.Unlock()

	msgs, ok := h.seriousMsgs[chat]
	if !ok {
		msgs = make(map[int]time.Time)
		h.seriousMsgs[chat] = msgs
	}

	cutoff := time.Now().Add(-seriousMemory)
	for id, at := range msgs {
		if at.Before(cutoff) {
			delete(msgs, id)
		}
	}

	msgs[messageID] = time.Now()
}

// wasSerious reports whether the given message of Nafanya's was answered
// seriously and is still within seriousMemory.
func (h *Handler) wasSerious(chat int64, messageID int) bool {
	h.seriousMsgsMux.Lock()
	defer h.seriousMsgsMux.Unlock()

	at, ok := h.seriousMsgs[chat][messageID]
	return ok && time.Since(at) < seriousMemory
}

// isSeriousRequest decides whether this message should be answered seriously:
// either it says so itself, or it is a reply to an answer that already was.
//
// The second case is what carries the tone through a thread. Telegram only
// populates one level of ReplyToMessage, so the chain cannot be walked back to
// the original "нафаня серьёзно" — hence remembering the answers instead. Each
// serious answer is recorded in turn, so the tone persists for as long as
// people keep replying.
func (h *Handler) isSeriousRequest(update tgbotapi.Update) bool {
	if isSerious(update) {
		return true
	}

	reply := update.Message.ReplyToMessage
	if reply == nil || reply.From == nil || reply.From.ID != h.bot.Self.ID {
		return false
	}

	return h.wasSerious(update.Message.Chat.ID, reply.MessageID)
}

func isSerious(update tgbotapi.Update) bool {
	// Matched case-insensitively, and with ё folded to е, so "Серьёзно" and
	// "СЕРЬЕЗНО" count the same as "серьезно". Both spellings are common and
	// phone keyboards disagree about which one they produce.
	text := strings.ToLower(update.Message.Text)
	text = strings.ReplaceAll(text, "ё", "е")
	return strings.Contains(text, "серьезно")
}

func getCleanDrawPrompt(update string) string {
	regex := regexp.MustCompile(`[Нн]афаня[, ]*([Нн]арисуй|[Сс]генерируй) `)
	return regex.ReplaceAllString(update, "")
}

func rollEmotion() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(emotionList))))
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
	n := nBig.Int64()

	return emotionList[n]
}

func (h *Handler) promptCompiler(id int64, promptType int, update tgbotapi.Update, serious bool) (prompt string, userInput string, model string, maxTokens int) {
	idx := slices.IndexFunc(h.chats, func(channel domain.Chat) bool {
		return channel.ID == id
	})

	curChannel := h.chats[idx]

	userInput = update.Message.From.FirstName + " " + update.Message.From.LastName + ": " + update.Message.Text

	nextMess := update.Message.ReplyToMessage

	if nextMess != nil {
		if !nextMess.From.IsBot {
			userInput = nextMess.From.FirstName + " " + nextMess.From.LastName + ": " + nextMess.Text + "\n" + userInput
		} else {
			userInput = nextMess.From.FirstName + ": " + nextMess.Text + "\n" + userInput
		}
	}

	switch promptType {
	case Question:
		var emotion string
		if serious {
			emotion = Serious
		} else {
			emotion = rollEmotion()
		}
		prompt = strings.ReplaceAll(curChannel.QuestionPrompt, "{emotion}", emotion)
	case RandomInterference:
		prompt = strings.ReplaceAll(curChannel.RandomInterferencePrompt, "{emotion}", rollEmotion())
	}

	// Return correct max tokens based on model
	aiModel := h.chats[idx].AIModel
	if aiModel == string(cfg.AIModelGemini35) {
		maxTokens = h.config.GoogleMaxTokens
	} else {
		maxTokens = h.config.OAIMaxTokens
	}

	return prompt, userInput, aiModel, maxTokens
}

func (h *Handler) reloadChannels() {
	var err error
	h.chats, err = h.db.GetAllChannelsConfig()
	if err != nil {
		sentry.CaptureException(err)
		panic(err)
	}
}

// sendMessage posts a reply and returns what Telegram accepted. The returned
// message is the zero value when sending failed; callers that only want the
// side effect can ignore it.
func (h *Handler) sendMessage(update tgbotapi.Update, message string) tgbotapi.Message {
	// Telegram rejects empty text with "Bad Request: message text is empty".
	// The model occasionally returns an empty string, so send a placeholder
	// instead of dropping the reply entirely.
	if strings.TrimSpace(message) == "" {
		message = "🤔"
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	msg.ReplyToMessageID = update.Message.MessageID

	sent, err := h.bot.Send(msg)
	if err != nil {
		sentry.CaptureException(err)
		log.Println(err)
		return tgbotapi.Message{}
	}

	return sent
}

func (h *Handler) deleteMessage(update tgbotapi.Update) {
	msg := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)

	// deleteMessage returns `true`, not a Message — use Request to avoid the
	// "unmarshal bool into tgbotapi.Message" error that Send would raise.
	_, err := h.bot.Request(msg)
	if err != nil {
		sentry.CaptureException(err)
		log.Println(err)
	}
}

func (h *Handler) sendImageByBytes(update tgbotapi.Update, data []byte, mimeType string) {
	ext := "png"
	if mimeType == "image/jpeg" {
		ext = "jpg"
	}
	file := tgbotapi.FileBytes{Name: "image." + ext, Bytes: data}
	photo := tgbotapi.NewPhoto(update.Message.Chat.ID, file)
	photo.ReplyToMessageID = update.Message.MessageID

	_, err := h.bot.Send(photo)
	if err != nil {
		sentry.CaptureException(err)
		log.Println(err)
	}
}

func (h *Handler) sendAction(update tgbotapi.Update, action string) {
	msg := tgbotapi.NewChatAction(update.Message.Chat.ID, action)

	// A chat action returns `true`, not a Message — use Request to avoid the
	// "unmarshal bool into tgbotapi.Message" error that Send would raise.
	_, err := h.bot.Request(msg)
	if err != nil {
		sentry.CaptureException(err)
		log.Println(err)
	}
}

// startAction sends a chat action immediately and keeps refreshing it every 4s
// (Telegram clears the action after ~5s) so "typing…"/"uploading photo…" stays
// visible for the whole time the bot prepares a reply or image. Call the
// returned stop func once the reply is sent.
func (h *Handler) startAction(update tgbotapi.Update, action string) (stop func()) {
	h.sendAction(update, action)

	ticker := time.NewTicker(4 * time.Second)
	done := make(chan struct{})
	go func() {
		defer sentry.Recover()
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				h.sendAction(update, action)
			}
		}
	}()

	return func() { close(done) }
}

func (h *Handler) isSupportedURL(update tgbotapi.Update) bool {
	rxRelaxed := xurls.Relaxed()
	urls := rxRelaxed.FindAllString(update.Message.Text, -1)
	for _, url := range urls {
		if strings.Contains(url, "https://twitter.com") || strings.Contains(url, "https://www.twitter.com") || strings.Contains(url, "https://x.com") || strings.Contains(url, "https://www.x.com") {
			if strings.Contains(url, "status") {
				return true
			}
		}
		if strings.Contains(url, "https://instagram.com") || strings.Contains(url, "https://www.instagram.com") {
			return true
		}
	}
	return false
}

func (h *Handler) updateMaxTokens(update tgbotapi.Update) {
	if h.isAdmin(update.Message.From.ID) {
		tokens, err := strconv.Atoi(update.Message.CommandArguments())
		if err != nil {
			h.sendMessage(update, "Invalid number")
			return
		}
		err2 := h.db.UpdateMaxTokens(tokens)
		if err2 != nil {
			h.sendMessage(update, "Error updating max tokens")
			return
		}
		h.sendMessage(update, "Max tokens updated")
		return
	}

	h.sendMessage(update, "You are not an admin")
}

func (h *Handler) checkIfURLReply(update tgbotapi.Update) bool {
	if strings.Contains(update.Message.ReplyToMessage.Text, "Saved") && strings.Contains(update.Message.ReplyToMessage.Text, "a click:") {
		return true
	}

	return false
}
