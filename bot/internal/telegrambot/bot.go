package telegrambot

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var bot *tgbotapi.BotAPI = nil

var logPrefix = "[BOT]  "

func logMessage(level slog.Level, message string, args ...any) {
	slog.Log(context.Background(), level, logPrefix+message, args...)
}

func InitBot() error {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is required")
	}
	logMessage(slog.LevelInfo, "Imported bot key")

	botToInit, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return fmt.Errorf("failed to create bot: %v", err)
	}
	logMessage(slog.LevelInfo, "Bot created")

	botToInit.Debug = false
	logMessage(slog.LevelInfo, fmt.Sprintf("Authorized on account %s", botToInit.Self.UserName))
	logMessage(slog.LevelInfo, "Bot is running")
	bot = botToInit

	return nil
}

func ListenToUpdates() error {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := bot.GetUpdatesChan(updateConfig)

	logMessage(slog.LevelInfo, "Listening for updates...")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "The bot works yay")

				if _, err := bot.Send(msg); err != nil {
					logMessage(slog.LevelError, "Failed to send message", "error", err)
				}
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Command unknown")
				_, err := bot.Send(msg)
				if err != nil {
					logMessage(slog.LevelError, "Failed to send message", "error", err)
				}
			}
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Non-command messages are not supported")
			_, err := bot.Send(msg)
			if err != nil {
				logMessage(slog.LevelError, "Failed to send message", "error", err)
			}
		}
	}
	return nil
}
