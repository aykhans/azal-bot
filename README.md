<h1 align="center">Azal-Bot: Receive Notifications for Flights on https://azal.az</h1>

## Installation

### With Docker (Recommended)
Pull the Docker image from Docker Hub:
```sh
docker pull aykhans/dodo:latest
```

### With Binary File
You can download the binaries from the [releases](https://github.com/aykhans/azal-bot/releases) section.

### Build from Source
To build Azal-Bot from source, you need to have [Go 1.22+](https://golang.org/dl/) installed. Follow these steps:

1. **Clone the repository:**

    ```sh
    git clone https://github.com/aykhans/azal-bot.git
    ```

2. **Navigate to the project directory:**

    ```sh
    cd azal-bot
    ```

3. **Build the project:**

    ```sh
    go build -ldflags "-s -w" -o azal-bot
    ```

This will generate an executable named `azal-bot` in the project directory.

## Usage

### Basic Usage
Search for flights from 2024-09-24T00:00:00 to 2024-09-27T23:59:59 every 60 seconds, then print the results to the CLI:
```sh
azal-bot \
    --first-date 2024-09-24 \
    --last-date 2024-09-27 \
    --from NAJ \
    --to BAK 
```
With Docker:
```sh
docker run --rm -d \
    aykhans/azal-bot \
    --first-date 2024-09-24 \
    --last-date 2024-09-27 \
    --from NAJ \
    --to BAK 
```

### With All Flags
Search for flights from 2024-09-24T15:00:00 to 2024-09-27T21:32:10 every 120 seconds, print the results to the CLI, and send a notification via Telegram if any flights are found:
```sh
azal-bot \
    --first-date 2024-09-24T15:00:00 \
    --last-date 2024-09-27T21:32:10 \
    --repeat-interval 120 \  # seconds
    --from NAJ \
    --to BAK \
    --telegram-bot-key "key" \
    --telegram-chat-id "id"
```
With Docker:
```sh
docker run --rm -d \
    aykhans/azal-bot \
    --first-date 2024-09-24T15:00:00 \
    --last-date 2024-09-27T21:32:10 \
    --repeat-interval 120 \  # seconds
    --from NAJ \
    --to BAK \
    --telegram-bot-key "key" \
    --telegram-chat-id "id"
```
