package storage

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	"vacancy_notifications_bot/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var sqlSchema string

var db *pgxpool.Pool

func checkIfDBExistsAndCreate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := pgx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}

	databaseName := config.Database
	config.Database = "postgres"

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	var exists bool

	err = conn.QueryRow(ctx,
		"select exists(select 1 from pg_database where datname = $1)",
		databaseName,
	).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = conn.Exec(ctx,
			"create database "+pgx.Identifier{databaseName}.Sanitize(),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func ConnectToDB() (*pgxpool.Pool, error) {
	slog.Info("Connecting to database...")

	err := checkIfDBExistsAndCreate()
	if err != nil {
		return nil, fmt.Errorf("error checking if db exists: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := database.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	slog.Info("Connected to database")
	db = database

	return database, nil
}

func InitDB() error {
	slog.Info("Initializing database...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, sqlSchema, pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}

	return nil
}

func GetSubscribers() ([]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ids, err := db.Query(ctx, "select telegram_uid from subscribers")
	if err != nil {
		return nil, fmt.Errorf("failed to get subscribers: %w", err)
	}
	defer ids.Close()

	var subscribers []int64
	for ids.Next() {
		var id int64
		if err = ids.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan subscribers: %w", err)
		}

		subscribers = append(subscribers, id)
	}

	if err = ids.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan subscribers: %w", err)
	}

	return subscribers, nil
}

func AddSubscriber(subscriber int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := db.Exec(
		ctx,
		"insert into subscribers(telegram_uid) values($1) on conflict do nothing",
		subscriber,
	)

	if err != nil {
		return fmt.Errorf("failed to add subscriber: %w", err)
	}

	return nil
}

func DeleteSubscriber(subscriber int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, "delete from subscribers where telegram_uid = $1", subscriber)
	if err != nil {
		return fmt.Errorf("failed to delete subscriber: %w", err)
	}

	return nil
}

func IsSubscribed(subscriber int64) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var subscribed bool

	err := db.QueryRow(ctx,
		"select exists(select 1 from subscribers where telegram_uid = $1)",
		subscriber,
	).Scan(&subscribed)

	if err != nil {
		return false, fmt.Errorf("failed to check subscription: %w", err)
	}

	return subscribed, nil
}

func GetVacancies() ([]model.Vacancy, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, "select vacancy_id, title, experience, url from vacancies")
	if err != nil {
		return nil, fmt.Errorf("failed to get vacancies: %w", err)
	}
	defer rows.Close()

	var vacancies []model.Vacancy

	for rows.Next() {
		var vacancy model.Vacancy

		if err := rows.Scan(
			&vacancy.ID,
			&vacancy.Title,
			&vacancy.Experience,
			&vacancy.URL,
		); err != nil {
			return nil, fmt.Errorf("failed to scan vacancies: %w", err)
		}

		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("failed to scan vacancies: %w", err)
		}

		vacancies = append(vacancies, vacancy)
	}

	return vacancies, nil
}

func CheckVacancyDiff(newVacancies []model.Vacancy) (addedVacancies []model.Vacancy, deletedVacancies []model.Vacancy, err error) {
	addedVacancies = make([]model.Vacancy, 0)
	deletedVacancies = make([]model.Vacancy, 0)

	oldVacancies, err := GetVacancies()
	if err != nil {
		return nil, nil, fmt.Errorf("diff check failed - failed to get old vacancies: %w", err)
	}

	existingURLs := make(map[string]struct{}, len(oldVacancies))
	for _, vacancy := range oldVacancies {
		existingURLs[vacancy.URL] = struct{}{}
	}
	for _, vacancy := range newVacancies {
		if _, exists := existingURLs[vacancy.URL]; !exists {
			addedVacancies = append(addedVacancies, vacancy)
			existingURLs[vacancy.URL] = struct{}{}
		}
	}

	incomingURLs := make(map[string]struct{}, len(newVacancies))
	for _, vacancy := range newVacancies {
		incomingURLs[vacancy.URL] = struct{}{}
	}
	for _, vacancy := range oldVacancies {
		if _, exists := incomingURLs[vacancy.URL]; !exists {
			deletedVacancies = append(deletedVacancies, vacancy)
		}
	}

	return addedVacancies, deletedVacancies, nil
}

func quote(toQuote string) string {
	return "'" + strings.ReplaceAll(toQuote, "'", "''") + "'"
}

func AddVacancies(vacancies []model.Vacancy) error {
	if len(vacancies) == 0 {
		return nil
	}

	query := "insert into vacancies (vacancy_id, title, experience, url) values "
	vacancyStrings := make([]string, 0, len(vacancies))
	for _, vacancy := range vacancies {
		vacancyString := "(" +
			quote(vacancy.ID) + ", " +
			quote(vacancy.Title) + ", " +
			quote(vacancy.Experience) + ", " +
			quote(vacancy.URL) + ")"
		vacancyStrings = append(vacancyStrings, vacancyString)
	}
	query += strings.Join(vacancyStrings, ", ") + " on conflict (url) do nothing"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to add vacancies: %w", err)
	}

	return nil
}

func DeleteVacancies(vacancies []model.Vacancy) error {
	if len(vacancies) == 0 {
		return nil
	}

	urls := make([]string, 0, len(vacancies))
	for _, vacancy := range vacancies {
		urls = append(urls, quote(vacancy.URL))
	}
	query := "delete from vacancies where url in (" + strings.Join(urls, ", ") + ")"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete vacancies: %w", err)
	}

	return nil
}

func SyncVacancies(vacancies []model.Vacancy) ([]model.Vacancy, error) {
	addedVacancies, deletedVacancies, err := CheckVacancyDiff(vacancies)
	if err != nil {
		return nil, fmt.Errorf("failed to sync vacancies: %w", err)
	}

	err = AddVacancies(addedVacancies)
	if err != nil {
		return nil, fmt.Errorf("failed to sync vacancies: %w", err)
	}

	err = DeleteVacancies(deletedVacancies)
	if err != nil {
		return nil, fmt.Errorf("failed to sync vacancies: %w", err)
	}

	return addedVacancies, nil
}
