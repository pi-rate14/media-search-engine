package main

import (
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/joho/godotenv"
)

type app struct {
	producer   sarama.AsyncProducer
	awsHandler *AWS
	consumer   sarama.Consumer
}

func NewConfig() *app {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal       // Wait for all in-sync replicas to acknowledge
	config.Producer.Compression = sarama.CompressionSnappy   // Use snappy compression
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	err := godotenv.Load(".env")
	awsHandler, err := NewAWS(
		os.Getenv("AWS_REGION"),
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
	)

	// Set up a Kafka producer using the configuration
	producer, err := sarama.NewAsyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		panic(err)
	}
	return &app{
		producer:   producer,
		awsHandler: awsHandler,
	}
}

func main() {
	app := NewConfig()
	app.listenForJobs()
	// get the current working directory
}
