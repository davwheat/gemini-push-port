package pubsub

import (
	"context"
	"crypto/tls"
	"errors"
	"gemini-push-port/logging"
	"gemini-push-port/rawstore"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

var IsShuttingDown = false // global flag to indicate shutdown

const messageLogInterval = 100

const maxBatchSize = 1_000_000 // 1 MB
const minBatchSize = 100       // 100 B

func Thread(rawMessageChan chan *rawstore.XmlMessageWithTime) {
	failedAttempts := 0
	messageCounter := 0

	var r *kafka.Reader
	defer func(r *kafka.Reader) {
		if r == nil {
			logging.Logger.Infof("Kafka reader is nil, nothing to close.")
			return
		}

		logging.Logger.Infof("Closing Kafka reader...")
		err := r.Close()
		if err != nil {
			log.Fatal("failed to close reader:", err)
		}
	}(r)

	topic := os.Getenv("KAFKA_TOPIC")
	host := os.Getenv("KAFKA_HOST")

	group := os.Getenv("CONSUMER_GROUP")
	username := os.Getenv("CONSUMER_USERNAME")
	password := os.Getenv("CONSUMER_PASSWORD")

	rawChanFailures := 0

outer:
	for {
		if failedAttempts > 0 {
			seconds := 1 << (min(failedAttempts, 8) - 1)

			logging.Logger.Warnf("%d failed connection attempts. Waiting %d seconds for next attempt", failedAttempts, seconds)
			time.Sleep(time.Duration(seconds) * time.Second)
			logging.Logger.Warnf("Starting next connection attempt...")
		}

		mechanism := plain.Mechanism{
			Username: username,
			Password: password,
		}
		dialer := &kafka.Dialer{
			Timeout:       10 * time.Second,
			DualStack:     true,
			SASLMechanism: mechanism,
			TLS:           &tls.Config{},
		}

		r = kafka.NewReader(kafka.ReaderConfig{
			Brokers:   []string{host},
			GroupID:   group,
			Topic:     topic,
			Dialer:    dialer,
			Partition: 0,
			MinBytes:  minBatchSize,
			MaxBytes:  maxBatchSize,
		})
		logging.Logger.Infof("Created reader for Kafka topic %s on host %s", topic, host)

		for {
			ctx := context.Background()
			m, err := r.FetchMessage(ctx)
			if err != nil {
				logging.Logger.Errorf(err, "failed to read pubsub message")
				failedAttempts++
				continue outer
			}

			messageCounter++
			if messageCounter%messageLogInterval == 0 {
				logging.Logger.Infof("Consumed %d messages", messageCounter)
				messageCounter = 0
			}

			rawMsg := rawstore.XmlMessageWithTime{
				MessageTime: m.Time.UTC(),
				Message:     string(m.Value),
			}

			select {
			case rawMessageChan <- &rawMsg:
				// don't reset to 0, in case its flapping around the full mark
				if rawChanFailures >= 2 {
					rawChanFailures -= 2
				}

				// commit the message
				err := r.CommitMessages(ctx, m)
				if err != nil {
					logging.Logger.Errorf(err, "failed to commit message: %s", string(m.Value))
				}
			default:
				logging.Logger.ErrorMsg("Raw message channel full, discarding value")
				rawChanFailures++
				if rawChanFailures > 200 {
					logging.Logger.Fatal(errors.New("raw message queue stuck for 200 messages, quitting"))
				}
			}

			if IsShuttingDown {
				logging.Logger.Infof("IsShuttingDown flag set, stopping message consumption...")
				break outer
			}
		}
	}
}
