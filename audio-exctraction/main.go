package main

import (
	"fmt"
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
	// inputFile := "5255c113-d905-d63a-db4f-b5b658c7d459.mp4"
	// outputFile := "output.wav"
	// chunkCount := 10

	// // Calculate the duration of the input video
	// durationCmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputFile)
	// durationOutput, err := durationCmd.Output()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// duration, err := strconv.ParseFloat(strings.TrimSpace(string(durationOutput)), 64)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // Calculate the duration of each chunk
	// chunkDuration, err := strconv.ParseFloat(fmt.Sprintf("%.2f", (float64(duration)/float64(chunkCount))), 64)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // Create a WaitGroup to wait for all audio extraction goroutines to finish
	// var wg sync.WaitGroup

	// // Extract the audio from each chunk in parallel
	// for i := 0; i < chunkCount; i++ {
	// 	var startTime float64 = float64(i) * chunkDuration
	// 	outputFilename := fmt.Sprintf("cache/chunk_%d.wav", i)

	// 	wg.Add(1)
	// 	go func(chunk int) {
	// 		defer wg.Done()

	// 		// Extract the audio from the chunk
	// 		cmd := exec.Command("ffmpeg", "-i", inputFile, "-ss", fmt.Sprintf("%f", startTime), "-t", fmt.Sprintf("%f", chunkDuration), "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "2", outputFilename)
	// 		err := cmd.Run()
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		log.Printf("Audio extracted from chunk %d\n", chunk)
	// 	}(i)
	// }

	// // Wait for all audio extraction goroutines to finish
	// wg.Wait()

	// // Concatenate the audio from all chunks
	// listFilename := joinChunkFilenames(chunkCount)
	// concatCmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFilename, "-c", "copy", outputFile)
	// err = concatCmd.Run()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // clear the chunks
	// clearCache(chunkCount)

	// log.Println("All audio chunks concatenated successfully")

	app := NewConfig()
	app.listenForJobs()
	// awsHandler.downloadFile("717617ec-a696-d85a-fef7-88febfb5597d_Spider_web.mp4")
	// listenForVideoChunks()
}

func joinChunkFilenames(inputFileName string, chunkCount int) string {
	// result := ""
	// for i := 0; i < chunkCount; i++ {
	// 	result += fmt.Sprintf("cache/chunk_%d.wav|", i)
	// }
	// return result[:len(result)-1]
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
