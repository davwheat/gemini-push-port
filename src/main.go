package main

import (
	"context"
	"gemini-push-port/logging"
	"gemini-push-port/pubsub"
	"gemini-push-port/rawstore"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-co-op/gocron/v2"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/joho/godotenv/autoload"
)

const serviceName = "gemini-push-port"

func main() {
	logging.InitialiseLogging(serviceName, true, logging.SentryConfig{
		DSN: "https://d63d2a2334e7211d78128ca6bab184d6@sentry.service.davw.network/7",
	})
	logger := logging.Logger

	logger.Infof("Starting consumer...")

	s, err := gocron.NewScheduler()
	if err != nil {
		logger.FatalE("failed to create scheduler", err)
	}

	r2s3config, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(os.Getenv("S3_COMPATIBLE_ACCESS_KEY_ID"), os.Getenv("S3_COMPATIBLE_SECRET_ACCESS_KEY"), "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		logger.FatalE("failed to load s3 config", err)
	}
	r2s3client := s3.NewFromConfig(r2s3config, func(opt *s3.Options) {
		opt.BaseEndpoint = aws.String(os.Getenv("S3_COMPATIBLE_ENDPOINT"))
	})

	_, err = s.NewJob(
		gocron.DurationJob(
			1*time.Minute,
		),
		gocron.NewTask(
			rawstore.DumpToBucketJob,
			r2s3client,
		),
		gocron.WithContext(context.Background()),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		logger.FatalE("failed to create dump to bucket job", err)
	}
	_, err = s.NewJob(
		gocron.DurationJob(
			1*time.Hour,
		),
		gocron.NewTask(
			rawstore.CleanUpLocalFilesJob,
		),
		gocron.WithContext(context.Background()),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		logger.FatalE("failed to create dump to bucket job", err)
	}

	rawMessagesChan := make(chan *rawstore.XmlMessageWithTime, 500_000)

	go pubsub.Thread(rawMessagesChan)
	go rawstore.Thread(rawMessagesChan)

	s.Start()

	sc := make(chan os.Signal)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	logger.Infof("Shutting down in 5 seconds")

	pubsub.IsShuttingDown = true // disconnect the feed so we stop getting more messages
	err = s.Shutdown()           // stop the scheduler
	if err != nil {
		logger.ErrorE("failed to shutdown scheduler", err)
	}

	// Wait for the process to finish processing any remaining messages
	time.Sleep(5 * time.Second)

	logger.Infof("Shutting down consumer...")
}
