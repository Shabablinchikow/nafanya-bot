package tghandler

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shabablinchikow/nafanya-bot/internal/domain"
)

func TestHandler_promptCompiler(t *testing.T) {
	type fields struct {
		chats  []domain.Chat
		config domain.BotConfig
	}
	type args struct {
		id         int64
		promptType int
		update     tgbotapi.Update
		serious    bool
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantUserInput string
	}{
		{
			name: "Test with quote",
			fields: fields{
				chats: []domain.Chat{
					{
						ID:             123,
						QuestionPrompt: "Answer with {emotion}",
					},
				},
				config: domain.BotConfig{
					GoogleMaxTokens: 1000,
				},
			},
			args: args{
				id:         123,
				promptType: Question,
				update: tgbotapi.Update{
					Message: &tgbotapi.Message{
						From: &tgbotapi.User{
							FirstName: "John",
							LastName:  "Doe",
						},
						Text: "> quoted text\nwhat do you think?",
						ReplyToMessage: &tgbotapi.Message{
							Text: "this is a message",
						},
					},
				},
				serious: false,
			},
			wantUserInput: "Quoted message: quoted text\nJohn Doe: what do you think?",
		},
		{
			name: "Test without quote",
			fields: fields{
				chats: []domain.Chat{
					{
						ID:             123,
						QuestionPrompt: "Answer with {emotion}",
					},
				},
			},
			args: args{
				id:         123,
				promptType: Question,
				update: tgbotapi.Update{
					Message: &tgbotapi.Message{
						From: &tgbotapi.User{
							FirstName: "John",
							LastName:  "Doe",
						},
						Text: "what do you think?",
						ReplyToMessage: &tgbotapi.Message{
							From: &tgbotapi.User{
								FirstName: "Jane",
								LastName:  "Doe",
							},
							Text: "this is a message",
						},
					},
				},
				serious: false,
			},
			wantUserInput: "Jane Doe: this is a message\nJohn Doe: what do you think?",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				chats:  tt.fields.chats,
				config: tt.fields.config,
			}
			_, userInput, _, _ := h.promptCompiler(tt.args.id, tt.args.promptType, tt.args.update, tt.args.serious)
			if userInput != tt.wantUserInput {
				t.Errorf("Handler.promptCompiler() userInput = %v, want %v", userInput, tt.wantUserInput)
			}
		})
	}
}
