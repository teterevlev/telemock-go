package main

import (
	"context"
	"log"

	telego "github.com/teterevlev/telemock-go"
)

func main() {
	bot, err := telego.NewBot("mock-token")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

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
					Text:   "Hello from telemock!",
				})
			case "/buttons":
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
					Text:        "Choose a button:",
					ReplyMarkup: &keyboard,
				})
			default:
				bot.SendMessage(ctx, &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: msg.Chat.ID},
					Text:   "You said: " + msg.Text,
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
