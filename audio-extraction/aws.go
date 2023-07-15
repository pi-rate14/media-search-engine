package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type AWS struct {
	bucket  string
	session *session.Session
}

func NewAWS(region, accessKeyID, secretAccessKey string) (*AWS, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		// Add your AWS access key ID and secret access key here
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, err
	}
	return &AWS{
		bucket:  "media-search-engine-bucket",
		session: sess,
	}, nil
}

func (awsHandler *AWS) downloadFile(key string, filename string) error {

	// Get the size of the file.
	svc := s3.New(awsHandler.session)
	obj, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(awsHandler.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to get object size. %v", err)
	}
	size := *obj.ContentLength
	partSize := int64(1024 * 1024 * 5) // 5 MB

	// Calculate the number of parts required to download the file.
	numParts := size / partSize
	if size%partSize != 0 {
		numParts++
	}

	// Download each part of the file and store it on disk.
	file, err := os.Create(fmt.Sprintf("cache/%s", filename))
	if err != nil {
		return fmt.Errorf("failed to create file. %v", err)
	}
	defer file.Close()

	for i := int64(0); i < numParts; i++ {
		start := i * partSize
		end := (i + 1) * partSize
		if end > size {
			end = size
		}

		// Download the part of the file.
		rangeHeader := "bytes=" + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end-1, 10)
		resp, err := svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(awsHandler.bucket),
			Key:    aws.String(key),
			Range:  aws.String(rangeHeader),
		})
		if err != nil {
			return fmt.Errorf("failed to download part. %v", err)
		}
		defer resp.Body.Close()

		// Write the part of the file to disk.
		_, err = file.Seek(start, 0)
		if err != nil {
			return fmt.Errorf("failed to seek file. %v", err)
		}
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			fmt.Println("Failed to write file", err)
			return fmt.Errorf("failed to write file. %v", err)
		}
	}
	fmt.Println("File downloaded successfully")
	return nil
}

func (awsHandler *AWS) uploadAudioFile(uuid string) (string, error) {

	uploader := s3manager.NewUploader(awsHandler.session)

	// Create an S3 object key using the file name
	// key := strings.ReplaceAll((*file).Filename(), " ", "_")

	uploadName := fmt.Sprintf("audios/%s.wav", uuid)
	fmt.Println(uploadName)

	file, err := os.Open(fmt.Sprintf("cache/%s.wav", uuid))
	if err != nil {
		fmt.Println("Failed to open file", err)
		return "", err
	}
	defer file.Close()

	// Create a buffer to read the file into
	buf := bytes.NewBuffer([]byte{})

	// Create a new buffered reader using a 4KB buffer.
	reader := bufio.NewReaderSize(file, 4096)

	for {
		// Read a chunk from the file.
		chunk, err := reader.ReadBytes('\n')

		// Write the chunk to the buffer.
		buf.Write(chunk)

		// Check for EOF or other errors.
		if err != nil {
			if err.Error() == "EOF" {
				break
			} else {
				fmt.Println("Failed to read file", err)
			}
		}
	}

	// Create an S3 upload input
	input := &s3manager.UploadInput{
		Bucket: aws.String(awsHandler.bucket), // Replace with your bucket name
		Key:    aws.String(uploadName),
		Body:   bytes.NewReader(buf.Bytes()),
	}

	// Upload the file to S3
	_, err = uploader.Upload(input)
	if err != nil {
		return "", err
	}

	fmt.Printf("File uploaded to S3: s3://%s/%s\n", *input.Bucket, *input.Key)
	return *input.Key, nil
}
