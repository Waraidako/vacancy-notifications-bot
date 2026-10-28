package telegram_bot

import (
	"fmt"
	"log/slog"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var bot *tgbotapi.BotAPI = nil

func InitBot() error {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is required")
	}
	slog.Info("Imported bot key")

	botToInit, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return fmt.Errorf("failed to create bot: %v", err)
	}
	slog.Info("Bot created")

	botToInit.Debug = false
	slog.Info(fmt.Sprintf("Authorized on account %s", botToInit.Self.UserName))
	slog.Info("Bot is running")
	bot = botToInit

	return nil
}

func ListenToUpdates() error {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := bot.GetUpdatesChan(updateConfig)

	slog.Info("Listening for updates...")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "The bot works yay")

				if _, err := bot.Send(msg); err != nil {
					slog.Error("Failed to send message: %v", err)
				}
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Command unknown")
				_, err := bot.Send(msg)
				if err != nil {
					slog.Error("Failed to send message: %v", err)
				}
			}
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Non-command messages are not supported")
			_, err := bot.Send(msg)
			if err != nil {
				slog.Error("Failed to send message: %v", err)
			}
		}
	}
	return nil
}
