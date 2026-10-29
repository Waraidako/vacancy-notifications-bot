package main

import (
	"fmt"
	"log/slog"
	"os"
	"vacancy_notifications_bot/internal/storage"

	"vacancy_notifications_bot/internal/grpcserver"
	"vacancy_notifications_bot/internal/telegrambot"
)

func main() {
	errorChannel := make(chan error)

	db, err := storage.ConnectToDB()
	if err != nil {
		slog.Error(fmt.Sprintf("Error connecting to DB: %v", err))
		os.Exit(1)
	}

	err = telegrambot.InitBot()
	if err != nil {
		slog.Error(fmt.Sprintf("Error initializing bot: %v", err))
		os.Exit(1)
	}

	go func() {
		if err := telegrambot.ListenToUpdates(); err != nil {
			errorChannel <- fmt.Errorf("telegram bot error: %w", err)
		}
	}()

	go func() {
		if err := grpcserver.StartServer(); err != nil {
			errorChannel <- fmt.Errorf("grpc server error: %w", err)
		}
	}()

	if err := <-errorChannel; err != nil {
		slog.Error("FATAL: %w", "error", err)
	}
}
