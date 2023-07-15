package main

import (
	"fmt"
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

}

func joinChunkFilenames(inputFileName string, chunkCount int) string {

	listFilename := fmt.Sprintf("cache/%s_chunk_list.txt", inputFileName)
	f, err := os.Create(listFilename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for i := 0; i < chunkCount; i++ {
		filename := fmt.Sprintf("chunk_%s_%d.wav", inputFileName, i)
		_, err = f.WriteString(fmt.Sprintf("file '%s'\n", filename))
		if err != nil {
			panic(err)
		}
	}
	return listFilename
}
