package telegrambot

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"vacancy_notifications_bot/internal/model"
	"vacancy_notifications_bot/internal/storage"

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
		switch update.Message.Text {
		case "✅ Subscribe":
			_, err := bot.Request(tgbotapi.NewDeleteMessage(
				update.Message.Chat.ID,
				update.Message.MessageID,
			))
			if err != nil {
				logMessage(slog.LevelError, "Failed to add delete message", "error", err)
			}
			userID := update.Message.From.ID
			err = storage.AddSubscriber(userID)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			if err != nil {
				logMessage(slog.LevelError, "Failed to add subscriber", "error", err)
				msg.Text = "Failed to subscribe"
			}
			msg.ReplyMarkup = SubscriptionKeyboard(true)
			msg.Text = fmt.Sprintf("Subscribed!")
			if _, err := bot.Send(msg); err != nil {
				logMessage(slog.LevelError, "Failed to send message", "error", err)
			}

		case "❌ Unsubscribe":
			_, err := bot.Request(tgbotapi.NewDeleteMessage(
				update.Message.Chat.ID,
				update.Message.MessageID,
			))
			if err != nil {
				logMessage(slog.LevelError, "Failed to add delete message", "error", err)
			}
			userID := update.Message.From.ID
			err = storage.DeleteSubscriber(userID)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			if err != nil {
				logMessage(slog.LevelError, "Failed to delete subscriber", "error", err)
				msg.Text = "Failed to unsubscribe"
			}
			msg.ReplyMarkup = SubscriptionKeyboard(false)
			msg.Text = fmt.Sprintf("Unsubscribed!")
			if _, err := bot.Send(msg); err != nil {
				logMessage(slog.LevelError, "Failed to send message", "error", err)
			}
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				userID := update.Message.From.ID
				subscribed, err := storage.IsSubscribed(userID)
				if err != nil {
					logMessage(slog.LevelError, "Failed to check subscription", "error", err)
				}

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "👋 **Welcome to Vacancy Notifications!**\n\nGet notified whenever new job vacancies become available.\n\nUse the button below to subscribe or unsubscribe from updates at any time.")
				msg.ParseMode = tgbotapi.ModeMarkdown

				msg.ReplyMarkup = SubscriptionKeyboard(subscribed)

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
		}
	}
	return nil
}

func constructVacancyLayout(vacancy model.Vacancy) string {
	return fmt.Sprintf(`<p><b>%s</b><br>
Experience: %s<br>
<a href="%s">Link to vacancy</a></p>`,
		html.EscapeString(vacancy.Title),
		html.EscapeString(vacancy.Experience),
		html.EscapeString(vacancy.URL),
	)
}

func sendRichMessage(chatID int64, messageText string) error {
	richMessage, err := json.Marshal(map[string]string{
		"html": messageText,
	})
	if err != nil {
		return err
	}

	_, err = bot.MakeRequest("sendRichMessage", tgbotapi.Params{
		"chat_id":      strconv.FormatInt(chatID, 10),
		"rich_message": string(richMessage),
	})

	return err
}

func NotifySubscribers(addedVacancies []model.Vacancy) error {
	subscribers, err := storage.GetSubscribers()
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	formattedVacancies := make([]string, 0, len(addedVacancies))
	for _, vacancy := range addedVacancies {
		formattedVacancies = append(formattedVacancies, constructVacancyLayout(vacancy))
	}

	messageText := "<p><b>New vacancies:</b></p>" +
		strings.Join(formattedVacancies, "<hr/>")

	for _, subscriber := range subscribers {
		if err := sendRichMessage(subscriber, messageText); err != nil {
			slog.Error(fmt.Sprintf("Failed to send update to %d: %v", subscriber, err))
		}
	}

	return nil
}

func SubscriptionKeyboard(subscribed bool) tgbotapi.ReplyKeyboardMarkup {
	label := "✅ Subscribe"

	if subscribed {
		label = "❌ Unsubscribe"
	}

	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(label),
		),
	)
}
