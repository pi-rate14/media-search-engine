package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	partitionConsumer, err := consumer.ConsumePartition("TRANSCRIPTION_JOBS", 0, sarama.OffsetNewest)
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

			// Write the chunk data to the file
			downloadKey := fmt.Sprintf("audios/%s.wav", uuid)
			err := app.awsHandler.downloadFile(downloadKey, fmt.Sprintf("%s.wav", uuid))
			if err != nil {
				panic(err)
			}
			err = app.handleProcessing(uuid)
			if err != nil {
				panic(err)
			}

		case err := <-partitionConsumer.Errors():
			panic(err)
		}
	}
}

func (app *app) handleProcessing(uuid string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// construct the path to the bin directory
	binPath := filepath.Join(wd, "whisper-repo")

	// change the working directory to the bin directory
	err = os.Chdir(binPath)
	if err != nil {
		return err
	}

	// construct the path to the executable
	execPath := filepath.Join(binPath, "main")

	log.Println("transcribing audio...")
	// pass flags and parameters to the executable
	cmd := exec.Command(execPath, "-m", "models/ggml-medium.en.bin", "-f", fmt.Sprintf("../cache/%s.wav", uuid), "-t", "8", "-ml", "125", "-pp", "true")

	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	// change the working directory back to the initial directory
	err = os.Chdir(wd)
	if err != nil {
		log.Fatal(err)
	}

	transcriptionOutput := strings.Split(string(output), "\n")
	for _, each := range transcriptionOutput {
		fmt.Println(each)
	}
	SaveToCSV(transcriptionOutput, fmt.Sprintf("cache/%s.csv", uuid))

	err = readCsvFile(fmt.Sprintf("cache/%s.csv", uuid))
	if err != nil {
		fmt.Printf("invalid csv: %v", err)
	}

	app.awsHandler.uploadTranscriptFile(uuid)

	message := &sarama.ProducerMessage{
		Topic: "EMBEDDING_JOBS",
		Value: sarama.ByteEncoder(uuid),
	}
	fmt.Println(message)
	app.producer.Input() <- message

	log.Println("EMBEDDING_JOB created.")
	return nil

}

func readCsvFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	_, err = csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("unable to parse file as CSV for "+filePath, err)
	}

	return nil
}
