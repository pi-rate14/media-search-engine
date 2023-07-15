package main

import (
	"net/http"
	"os"

	"github.com/Shopify/sarama"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

type app struct {
	producer   sarama.AsyncProducer
	awsHandler *AWS
	consumer   sarama.Consumer
	redis      *redis.Client
}

func NewConfig() *app {

	err := godotenv.Load(".env")
	awsHandler, err := NewAWS(
		os.Getenv("AWS_REGION"),
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
	)

	client := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Set up a Kafka producer using the configuration
	kf, err := NewKafkaProducer()
	if err != nil {
		panic(err)
	}
	return &app{
		producer:   kf.producer,
		redis:      client,
		awsHandler: awsHandler,
	}
}

func main() {
	app := NewConfig()
	go app.listenForJobs()
	router := mux.NewRouter()
	router.HandleFunc("/create", app.handleVideoUpload)
	router.HandleFunc("/createFromLink", app.handleYoutubeLink)
	router.HandleFunc("/status", app.jobStatus)
	router.HandleFunc("/query", app.serveQuery)
	corsHandler := cors.Default().Handler(router)
	err := http.ListenAndServe(":8085", corsHandler)
	if err != nil {
		panic(err)
	}
}
