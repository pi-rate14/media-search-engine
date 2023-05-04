package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/redis/go-redis/v9"
)

func (app *app) handleVideoUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form data from the request
	err := r.ParseMultipartForm(32 << 20) // max file size of 32MB
	if err != nil {
		http.Error(w, "Failed to parse multipart form data", http.StatusBadRequest)
		return
	}

	// Get the uploaded file from the form data
	file, multipartFileHeader, err := r.FormFile("video")
	if err != nil {
		http.Error(w, "Failed to get uploaded file from form data", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileHeader := make([]byte, 512)

	// Copy the headers into the FileHeader buffer
	if _, err := file.Read(fileHeader); err != nil {
		http.Error(w, "cant't read file", 500)
	}

	// set position back to start.
	if _, err := file.Seek(0, 0); err != nil {
		http.Error(w, "cant't read file", 500)
	}

	log.Printf("Name: %#v\n", multipartFileHeader.Filename)
	log.Printf("MIME: %#v\n", http.DetectContentType(fileHeader))

	fileName := strings.ReplaceAll(multipartFileHeader.Filename, "-_-", "")
	filepath, err := app.awsHandler.UploadVideoFile(&file, fileName)
	if err != nil {
		http.Error(w, "Failed to upload file to S3", http.StatusBadRequest)
		return
	}

	tokens := strings.Split(filepath, "-_-")
	uploadedFileName := tokens[1]
	fmt.Printf("uploaded file name: %s", uploadedFileName)
	folder := strings.Split(tokens[0], "/")[0]
	fmt.Printf("uploaded folder name: %s", folder)

	uuid := strings.Split(tokens[0], "/")[1]
	fmt.Printf("uploaded uuid name: %s", uuid)

	message := &sarama.ProducerMessage{
		Topic: "JOBS",
		Value: sarama.ByteEncoder(uuid),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("filename"),
				Value: []byte(uploadedFileName),
			},
			{
				Key:   []byte("folder"),
				Value: []byte(folder),
			},
		},
	}
	fmt.Println(message)
	app.producer.Input() <- message

	ctx := context.Background()

	err = app.redis.Set(ctx, uuid, "1", 0).Err()
	if err != nil {
		panic(err)
	}
	fmt.Printf("SET VALUE FOR %s: %s\n", uuid, "1")

	// Send a response back to the client
	w.Write([]byte(fmt.Sprintf("job queued with UUID: %s and filename %s/%s\n", uuid, folder, fileName)))
}

func (app *app) jobStatus(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	query := r.URL.Query()
	jobID := query.Get("job_id") //filters="color"
	if jobID == "" {
		http.Error(w, "job ID is empty", http.StatusBadRequest)

	}

	val, err := app.redis.Get(ctx, jobID).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			http.Error(w, "Failed to serve request because of redis error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}

	fmt.Printf("GOT VALUE FOR %s: %s\n", jobID, val)

	switch val {
	case "0":
		w.Write([]byte(fmt.Sprintf("job with UUID: %s\nSTATUS: FAILED", jobID)))
	case "1":
		w.Write([]byte(fmt.Sprintf("job with UUID: %s\nSTATUS: PENDING", jobID)))
	case "2":
		w.Write([]byte(fmt.Sprintf("job with UUID: %s\nSTATUS: COMPLETED", jobID)))
	}
}

type Query struct {
	QueryString string `json:"query"`
	Uuid        string `json:"uuid"`
}

type QueryResponse struct {
	Data string `json:"data"`
}

func (app *app) serveQuery(w http.ResponseWriter, r *http.Request) {
	var q Query
	ctx := context.Background()

	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	val, err := app.redis.Get(ctx, q.Uuid).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			http.Error(w, "Failed to serve request because of redis error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}

	if val != "2" {
		http.Error(w, "job doesn't exist. Check using status API.", http.StatusInternalServerError)
	}

	queryJson, err := json.Marshal(q)
	if err != nil {
		http.Error(w, "cant marshal body", http.StatusInternalServerError)
	}

	resp, err := http.Post("http://localhost:8086/query", "application/json",
		bytes.NewBuffer(queryJson),
	)

	if err != nil {
		http.Error(w, "cant get response from GPT", http.StatusInternalServerError)
	}

	var gptRes string
	err = json.NewDecoder(resp.Body).Decode(&gptRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// jData, err := json.Marshal(gptRes.Data)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }
	// w.Header().Set("Content-Type", "application/json")
	// w.Write(jData)
	w.Write([]byte(gptRes))
}

func (app *app) listenForJobs() error {
	fmt.Println("Listening for jobs...")
	// Create a Kafka consumer
	consumer, err := sarama.NewConsumer([]string{"localhost:9092"}, nil)
	if err != nil {
		return err
	}
	defer consumer.Close()

	// Subscribe to the Kafka topic
	partitionConsumer, err := consumer.ConsumePartition("FINISHED_JOBS", 0, sarama.OffsetNewest)
	if err != nil {
		return err
	}
	defer partitionConsumer.Close()

	for {
		select {
		case message := <-partitionConsumer.Messages():
			fmt.Println("Found job!")
			uuid := string(message.Value)
			fmt.Println("UUID: ", uuid)

			ctx := context.Background()

			err = app.redis.Set(ctx, strings.Trim(uuid, "\""), "2", 0).Err()
			fmt.Printf("SET VALUE FOR %s\n: %s", uuid, "2")
			if err != nil {
				panic(err)
			}

		case err := <-partitionConsumer.Errors():
			panic(err)
		}
	}
}
