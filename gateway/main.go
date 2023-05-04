package main

import (
	"net/http"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type app struct {
	producer   sarama.AsyncProducer
	awsHandler *AWS
	consumer   sarama.Consumer
	redis      *redis.Client
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

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Set up a Kafka producer using the configuration
	producer, err := sarama.NewAsyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		panic(err)
	}
	return &app{
		producer:   producer,
		redis:      client,
		awsHandler: awsHandler,
	}
}

func main() {
	app := NewConfig()
	go app.listenForJobs()
	http.HandleFunc("/create", app.handleVideoUpload)
	http.HandleFunc("/status", app.jobStatus)
	http.HandleFunc("/query", app.serveQuery)
	err := http.ListenAndServe(":8085", nil)
	if err != nil {
		panic(err)
	}
}
