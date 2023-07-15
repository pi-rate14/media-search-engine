package main

import (
	"os"

	"github.com/Shopify/sarama"
	"github.com/joho/godotenv"
)

type app struct {
	producer   sarama.AsyncProducer
	awsHandler *AWS
	consumer   sarama.Consumer
}

func NewConfig() *app {

	err := godotenv.Load(".env")
	awsHandler, err := NewAWS(
		os.Getenv("AWS_REGION"),
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
	)
	kf, err := NewKafkaProducer()
	if err != nil {
		panic(err)
	}
	return &app{
		producer:   kf.producer,
		awsHandler: awsHandler,
	}
}

func main() {
	app := NewConfig()
	app.listenForJobs()
	// get the current working directory
}
