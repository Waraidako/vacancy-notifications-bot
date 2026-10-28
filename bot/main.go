package main

import (
	"fmt"
	"log/slog"
	"os"
	"vacancy_notifications_bot/internal/telegram_bot"
)

func main() {
	errorChannel := make(chan error)

	bot, err := telegram_bot.InitBot()
	if err != nil {
		slog.Error(fmt.Sprintf("Error initializing bot: %v", err))
		os.Exit(1)
	}

	go func() {
		if err := telegram_bot.ListenToUpdates(bot); err != nil {
			errorChannel <- fmt.Errorf("telegram bot error: %w", err)
		}
	}()

	if err := <-errorChannel; err != nil {
		slog.Error("FATAL: %w", err)
	}
}
