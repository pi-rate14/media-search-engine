package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
)

func (app *app) listenForJobs() error {
	fmt.Println("Listening for jobs...")
	// Create a Kafka consumer
	consumer, err := sarama.NewConsumer([]string{"localhost:9092"}, nil)
	if err != nil {
		return err
	}
	defer consumer.Close()

	// Subscribe to the Kafka topic
	partitionConsumer, err := consumer.ConsumePartition("JOBS", 0, sarama.OffsetNewest)
	if err != nil {
		return err
	}
	defer partitionConsumer.Close()

	for {
		select {
		case message := <-partitionConsumer.Messages():
			fmt.Println("Found job!")

			// Parse the headers from the Kafka message
			headers := message.Headers
			fileNameHeader := headers[0]
			folderHeader := headers[1]

			// If the message doesn't have the expected headers, skip it
			if fileNameHeader == nil || folderHeader == nil {
				continue
			}

			fileName := string(fileNameHeader.Value)
			folder := string(folderHeader.Value)
			uuid := string(message.Value)

			// Write the chunk data to the file
			downloadKey := fmt.Sprintf("%s/%s-_-%s", folder, uuid, fileName)
			err := app.awsHandler.downloadFile(downloadKey, fmt.Sprintf("%s-_-%s", uuid, fileName))
			if err != nil {
				panic(err)
			}
			err = app.handleProcessing(fmt.Sprintf("%s-_-%s", uuid, fileName), uuid)
			if err != nil {
				panic(err)
			}

		case err := <-partitionConsumer.Errors():
			panic(err)
		}
	}
}

func (app *app) handleProcessing(inputFileName string, uuid string) error {
	inputFile := fmt.Sprintf("cache/%s", inputFileName)
	outputFile := fmt.Sprintf("cache/%s.wav", uuid)
	chunkCount := 10

	// Calculate the duration of the input video
	fmt.Println(inputFile)
	durationCmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputFile)
	durationOutput, err := durationCmd.Output()
	if err != nil {
		return fmt.Errorf("ffmpeg couldn't give duration of input: %v", err)
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(durationOutput)), 64)
	if err != nil {
		return fmt.Errorf("couldn't parse duration as float: %v", err)
	}

	// Calculate the duration of each chunk
	chunkDuration, err := strconv.ParseFloat(fmt.Sprintf("%.2f", (float64(duration)/float64(chunkCount))), 64)
	if err != nil {
		return fmt.Errorf("couldn't parse chunk duration as float: %v", err)
	}

	// Create a WaitGroup to wait for all audio extraction goroutines to finish
	var wg sync.WaitGroup

	// Extract the audio from each chunk in parallel
	for i := 0; i < chunkCount; i++ {
		var startTime float64 = float64(i) * chunkDuration
		outputFilename := fmt.Sprintf("cache/chunk_%s_%d.wav", inputFileName, i)

		wg.Add(1)
		go func(chunk int) {
			defer wg.Done()

			// Extract the audio from the chunk
			cmd := exec.Command("ffmpeg", "-i", inputFile, "-ss", fmt.Sprintf("%f", startTime), "-t", fmt.Sprintf("%f", chunkDuration), "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "2", outputFilename)
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
			log.Printf("Audio extracted from chunk %d\n", chunk)
		}(i)
	}

	// Wait for all audio extraction goroutines to finish
	wg.Wait()

	// Concatenate the audio from all chunks
	listFilename := joinChunkFilenames(inputFileName, chunkCount)
	concatCmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFilename, "-c", "copy", outputFile)
	err = concatCmd.Run()
	if err != nil {
		return fmt.Errorf("couldn't concat chanks: %v", err)
	}

	// clear the chunks
	err = clearCache(inputFileName, chunkCount)
	if err != nil {
		return err
	}

	app.awsHandler.uploadAudioFile(uuid)

	message := &sarama.ProducerMessage{
		Topic: "TRANSCRIPTION_JOBS",
		Value: sarama.ByteEncoder(uuid),
	}
	fmt.Println(message)
	app.producer.Input() <- message

	// log.Println("All audio chunks concatenated successfully")
	return nil
}

func clearCache(inputFileName string, chunkCount int) error {
	for i := 0; i < chunkCount; i++ {
		err := os.Remove(fmt.Sprintf("cache/chunk_%s_%d.wav", inputFileName, i))
		if err != nil {
			return fmt.Errorf("couldn't clear cache: %v", err)
		}
	}
	os.Remove(fmt.Sprintf("cache/%s", inputFileName))
	return nil
}
