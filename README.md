# Gemini push port storer

Stores messages pushed through the Gemini product on the Rail Data Marketplace, dumps them into flat files, and pushes
them to cloud storage.

This implementation can be used as a reference for building your own service which consumes data from the Gemini push
port.

## Local development

This project uses Go 1.25.

For development, copy the `.env.example` file at the root of the repository to `src/.env` and fill in the missing
values, using your own PubSub credentials from the Rail Data Marketplace, and your own S3-compatible storage credentials
and endpoint.

Simply run `main.go` in your favourite Go IDE or on the command line, and you're ready to go.

Messages from the push port will be stored at the location specified by the `PUSH_PORT_DUMP_WORKDIR` environment
variable in flat files organised into directories by year, month and day. For example, a message received on 19
September 2025 at 16:45 will be stored in `${PUSH_PORT_DUMP_WORKDIR}/2025/09/19/16.pport`.

Every minute, the service will attempt to push the current hour and previous hour's flat files to the configured
S3-compatible storage. Every hour, the service will attempt to delete flat files older than one week so that it doesn't
fill up your local disk. Intervals for both of these tasks can be configured within `src/main.go`.

## Deployment

Copy the `.env.example` file at the root of the repository to `.env` and fill in the missing values, using your own
PubSub credentials from the Rail Data Marketplace, and your own S3-compatible storage credentials and endpoint.

Build the docker container and run it with Docker Compose:

```bash
docker compose up --build
```

The service will start and run in the background.
