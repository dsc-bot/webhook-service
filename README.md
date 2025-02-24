# Webhook Service

The service that sends the webhooks out

## How to run

### Production

1. Clone the repository (`git clone https://github.com/dsc-bot/fresh-data-service.git`)
2. Run `docker build -t webhook-service .`
3. Run `docker run -d -e LOG_LEVEL=info webhook-service`

### Development

1. Clone the repository (`git clone https://github.com/dsc-bot/fresh-data-service.git`)
2. Run `go mod download`
3. Run `go mod verify`
4. Run `go run main.go`