package main

import (
	"context"
	"log"

	telego "github.com/teterevlev/telemock-go"
)

const API_KEY = "API Key is not being used. This const is just for reverse compatibility"

func main() {
	bot, err := telego.NewBot(API_KEY)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Получаем канал обновлений через long polling
	updates, err := bot.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{})
	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		if update.Message != nil {
			msg := update.Message

			switch msg.Text {
			case "/start":
				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: msg.Chat.ID},
					Text:   "Hello",
				})
			case "/b":
				keyboard := telego.InlineKeyboardMarkup{
					InlineKeyboard: [][]telego.InlineKeyboardButton{
						{
							{Text: "Button 1", CallbackData: "1"},
							{Text: "Button abc", CallbackData: "abc"},
						},
					},
				}
				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID:      telego.ChatID{ID: msg.Chat.ID},
					Text:        "text",
					ReplyMarkup: &keyboard,
				})
			default:
				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID:           telego.ChatID{ID: msg.Chat.ID},
					Text:             "ack",
					ReplyToMessageID: msg.MessageID,
				})
			}
		}

		if update.CallbackQuery != nil {
			cq := update.CallbackQuery
			bot.SendMessage(ctx, &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: cq.From.ID},
				Text:   "Button pressed: " + cq.Data,
			})
		}
	}
}
