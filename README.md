# Vacancy Notifications Bot

A self-hosted Telegram bot that notifies subscribers about newly available job vacancies. It combines a scheduled Node.js subscription service with a Go notification service, communicates over gRPC, and uses PostgreSQL to track vacancies and subscribers.

The project currently includes an **HH.ru-style mock source for Ozon Bank vacancies**. Its subscription-script interface is designed to accommodate additional sources without changing the Go service.

## Features

- **Scheduled updates:** A Node.js service runs subscription scripts through cron, rather than keeping a Node process running between updates.
- **Pluggable sources:** Add a JavaScript file implementing the common `updateSubscription()` interface to supply vacancies from another source.
- **gRPC communication:** Node.js sends one aggregated batch using a shared Protocol Buffers contract. Go acknowledges receipt without waiting for database synchronization or Telegram delivery.
- **Vacancy synchronization:** PostgreSQL tracks the current set of vacancies. The Go service identifies new and disappeared listings by their unique URLs, inserts new ones, and removes those no longer present.
- **Telegram subscriptions:** Users can subscribe and unsubscribe using a reply keyboard. Subscription state is stored in PostgreSQL.
- **Formatted notifications:** Subscribers receive new vacancies with titles, experience requirements, and links.
- **Docker Compose:** Runs both services and PostgreSQL, with health checks and persistent database storage.

## Architecture

```text
Node.js subscription service                 Go notification service
┌────────────────────────────┐               ┌─────────────────────────────┐
│ cron                       │               │ gRPC server                 │
│   ↓                        │               │   ↓                         │
│ load subscription scripts  │── gRPC ──────>│ acknowledge receipt         │
│   ↓                        │<─ ack ────────│   ↓                         │
│ aggregate vacancy results  │               │ process batch in goroutine  │
└────────────────────────────┘               │   ↓                         │
                                             │ PostgreSQL vacancy diff     │
                                             │   ↓                         │
                                             │ notify Telegram subscribers │
                                             └─────────────────────────────┘
```

**Receipt is not processing confirmation.** A successful `PushVacancies` acknowledgment means Go has received the batch and handed it off for processing; it does not mean database changes or Telegram notifications have completed.

Vacancies are compared using their **URLs**, not source-specific IDs. This allows multiple sources to use overlapping vacancy IDs. The database represents the currently observed set: when a listing disappears it is removed, and if the same URL returns later, it is treated as new again.

## Getting started

### Requirements

- Docker and Docker Compose
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- `make` (optional; the equivalent Docker Compose command is shown below)

### 1. Clone the repository

```bash
git clone https://github.com/Waraidako/vacanacy-notifications-bot.git
cd vacanacy-notifications-bot
```

### 2. Configure environment variables

Copy the provided template:

```bash
cp .env.example .env
```

Fill in the variables in `.env`:

```dotenv
TELEGRAM_BOT_TOKEN=your_telegram_bot_token

POSTGRES_USER=your_postgres_user
POSTGRES_PASSWORD=your_postgres_password
POSTGRES_DB=vacancy_notifications

DB_PORT=5433
DB_INTERNAL_PORT=5432
BOT_GRPC_PORT=50051
```

`DB_PORT` is the host-facing PostgreSQL port; using `5433` avoids conflicts with a local server already listening on `5432`. Containers communicate with PostgreSQL over the internal port, and the subscription service reaches Go at `bot:50051` on the Compose network.

### 3. Build and start

```bash
docker compose up --build --abort-on-container-failure
```

Or use the included Makefile:

```bash
make up
```

**Note:** The repository's `make up` target runs `sudo docker compose ...`.

On startup, the Go service connects to PostgreSQL, initializes the tables from its embedded SQL schema, initializes the Telegram bot, and starts the gRPC server. Docker Compose health checks ensure the dependent containers start in order.

Stop the stack with `Ctrl+C` or, from another terminal:

```bash
docker compose down
```

The PostgreSQL data is retained in the `postgres_data` Docker volume.

## Using the Telegram bot

Open your bot in Telegram and press **Start**, or send `/start`.

The bot presents a reply-keyboard button reflecting your subscription status:

- **✅ Subscribe** — register your Telegram user ID to receive notifications.
- **❌ Unsubscribe** — remove your user ID from the subscriber table and stop notifications.

When the subscription service reports newly observed vacancies, subscribed users receive formatted messages containing vacancy details and clickable links.

## Adding a subscription source

The Node service loads `.js` files directly inside:

```text
subscription-service/src/subscriptions/
```

Each file must export an asynchronous `updateSubscription()` function that returns an array of normalized vacancies:

```js
export async function updateSubscription() {
    // Fetch, validate, and normalize vacancies from your source.
    return [
        {
            id: "source-specific-id",
            title: "Backend Developer",
            experience: "1-3",
            url: "https://example.com/jobs/123"
        }
    ];
}
```

### Vacancy contract

| Field        | Type     | Meaning                                           |
| ------------ | -------- | ------------------------------------------------- |
| `id`         | `string` | Vacancy identifier supplied by the source         |
| `title`      | `string` | Job title                                         |
| `experience` | `string` | Experience requirement or normalized range        |
| `url`        | `string` | Unique URL identifying the listing across sources |

All four fields are required by the Go/PostgreSQL schema. The source script is responsible for fetching, validating, and normalizing its own data before returning it. The Go service deliberately does not depend on individual source APIs or their original JSON formats.

The aggregator combines the results of the loaded modules into one gRPC request. Because **missing listings are deleted** from PostgreSQL, new source integrations should return complete, trustworthy results for their source rather than a partial page.

### Included mock source

`hh-ru-ozon-bank-mock.js` loads and validates an HH.ru-shaped JSON fixture, filters it for matching Ozon Bank vacancies, and returns a randomized subset. It **does not make real HH.ru API requests**. The random sampling is useful for exercising the add/delete diff and notification behavior.

## Update schedule

The Node container runs cron in the foreground. The current `subscription-service/crontab` runs the update script **once per minute**, for testing:

```cron
*/1 * * * * root cd /subscription-service && /usr/local/bin/node src/update-subscriptions.js >> /proc/1/fd/1 2>&1
```

To run every six hours instead, change the cron expression to `0 */6 * * *` and rebuild the subscription-service image.

## Project structure

```text
.
├── bot/
│   ├── main.go                      # Initialize services and supervise long-running goroutines
│   ├── internal/
│   │   ├── grpcserver/              # Receive vacancy batches and coordinate processing
│   │   ├── model/                   # Shared Go vacancy type
│   │   ├── storage/                 # PostgreSQL queries, schema, and vacancy synchronization
│   │   └── telegrambot/             # Bot commands, subscriptions, and notifications
│   └── proto/                       # Generated Go Protobuf and gRPC code
├── proto/
│   └── subscriptions.proto          # Shared gRPC contract
├── subscription-service/
│   ├── crontab                      # Scheduled update interval
│   └── src/
│       ├── grpc-client.js           # Send batches to Go
│       ├── update-subscriptions.js  # Run sources and aggregate results
│       └── subscriptions/           # Pluggable source scripts and mock data
├── compose.yaml
├── Makefile
└── .env.example
```

## Regenerating the Go gRPC code

If you modify `proto/subscriptions.proto`, regenerate the Go bindings from the repository root:

```bash
make proto
```

This target requires `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` to be installed and available on your `PATH`. The JavaScript service loads the `.proto` contract dynamically at runtime.
