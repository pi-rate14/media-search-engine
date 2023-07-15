package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/kkdai/youtube/v2"
	"github.com/redis/go-redis/v9"
)

type jobResponse struct {
	PollUrl string `json:"poll_url"`
	UUID    string `json:"uuid"`
}

func (app *app) handleVideoUpload(w http.ResponseWriter, r *http.Request) {
	setupCorsResponse(&w, r)
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
	filepath, err := app.awsHandler.UploadVideoFile(file, fileName)
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
	var resp jobResponse
	resp.PollUrl = fmt.Sprintf("http://localhost:8085/status?job_id=%s", uuid)
	resp.UUID = uuid
	w.WriteHeader(http.StatusCreated)
	respJson, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to upload file to S3", http.StatusBadRequest)
		return
	}
	w.Write(respJson)
}

type statusResponse struct {
	UUID   string `json:"uuid"`
	Status string `json:"status"`
}

func (app *app) jobStatus(w http.ResponseWriter, r *http.Request) {
	setupCorsResponse(&w, r)
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

	var s statusResponse
	s.UUID = jobID

	fmt.Printf("GOT VALUE FOR %s: %s\n", jobID, val)

	switch val {
	case "0":
		s.Status = "FAILED"
		jsonR, err := json.Marshal(s)
		if err != nil {
			http.Error(w, "Failed to serve request because of redis error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		w.Write(jsonR)
	case "1":
		s.Status = "PENDING"
		jsonR, err := json.Marshal(s)
		if err != nil {
			http.Error(w, "Failed to serve request because of redis error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		w.Write(jsonR)
	case "2":
		s.Status = "COMPLETED"
		jsonR, err := json.Marshal(s)
		if err != nil {
			http.Error(w, "Failed to serve request because of redis error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		w.Write(jsonR)
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
	setupCorsResponse(&w, r)
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

	resp, err := http.Post("http://gpt-service:8086/query", "application/json",
		bytes.NewBuffer(queryJson),
	)

	if err != nil {
		http.Error(w, "cant get response from GPT", http.StatusInternalServerError)
	}

	var gptRes QueryResponse
	err = json.NewDecoder(resp.Body).Decode(&gptRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse, err := json.Marshal(gptRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response to the HTTP client
	w.Write(jsonResponse) // jData, err := json.Marshal(gptRes.Data)
}

func (app *app) listenForJobs() error {
	fmt.Println("Listening for jobs...")
	// Create a Kafka consumer
	consumer, err := sarama.NewConsumer([]string{"broker:29092"}, nil)
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

type youtubeLinkRequest struct {
	Link string `json:"link"`
}

func (app *app) handleYoutubeLink(w http.ResponseWriter, r *http.Request) {
	setupCorsResponse(&w, r)
	var q youtubeLinkRequest

	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	client := youtube.Client{}

	parsedURL, err := url.Parse(q.Link)
	if err != nil {
		fmt.Printf("Error parsing URL: %s\n", err)
		return
	}

	// Get the value of the "v" query parameter
	videoID := parsedURL.Query().Get("v")
	if videoID == "" {
		fmt.Println("No 'v' query parameter found in the URL")
		return
	}

	fmt.Println(videoID)

	video, err := client.GetVideo(videoID)
	if err != nil {
		panic(err)
	}

	formats := video.Formats.WithAudioChannels() // only get videos with audio
	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		panic(err)
	}
	filepath, err := app.awsHandler.UploadVideoFile(stream, videoID)
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
	var resp jobResponse
	resp.PollUrl = fmt.Sprintf("http://localhost:8085/status?job_id=%s", uuid)
	resp.UUID = uuid
	w.WriteHeader(http.StatusCreated)
	respJson, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to upload file to S3", http.StatusBadRequest)
		return
	}
	w.Write(respJson)

}

func setupCorsResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization")
}
