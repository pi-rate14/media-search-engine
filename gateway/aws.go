package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/hashicorp/go-uuid"
)

type AWS struct {
	region          string
	accessKeyID     string
	secretAccessKey string
	bucket          string
	session         *session.Session
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

func (awsHandler *AWS) UploadVideoFile(file io.Reader, fileName string) (string, error) {

	uploader := s3manager.NewUploader(awsHandler.session)

	// Create an S3 object key using the file name
	// key := strings.ReplaceAll((*file).Filename(), " ", "_")
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return "", err
	}
	uploadName := fmt.Sprintf("videos/%s-_-%s", uuid, fileName)

	// Create a buffer to read the file into
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, file)
	if err != nil {
		return "", err
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

const chunkSize = 5 * 1024 * 1024 // 5 MB

func (awsHandler *AWS) UploadLargeVideoFile(filepath, key string) error {
	// Create a new session with your AWS credentials
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsHandler.region),
		Credentials: credentials.NewStaticCredentials(awsHandler.accessKeyID, awsHandler.secretAccessKey, ""),
	})
	if err != nil {
		return err
	}

	// Open the video file
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a new S3 service client
	svc := s3.New(sess)

	// Initialize the multipart upload
	uploadInput := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(awsHandler.bucket),
		Key:    aws.String(key),
	}
	resp, err := svc.CreateMultipartUpload(uploadInput)
	if err != nil {
		return err
	}
	uploadID := resp.UploadId
	defer func() {
		// Abort the upload if it fails or encounters an error
		svc.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
			Bucket:   aws.String(awsHandler.bucket),
			Key:      aws.String(key),
			UploadId: uploadID,
		})
	}()

	// Read the file in chunks and upload each chunk as a separate part
	var parts []*s3.CompletedPart
	buffer := make([]byte, chunkSize)
	for partNumber := 1; ; partNumber++ {
		// Read the next chunk from the file
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Upload the chunk as a separate part
		uploadPartInput := &s3.UploadPartInput{
			Bucket:        aws.String(awsHandler.bucket),
			Key:           aws.String(key),
			PartNumber:    aws.Int64(int64(partNumber)),
			UploadId:      uploadID,
			Body:          bytes.NewReader(buffer[:bytesRead]),
			ContentLength: aws.Int64(int64(bytesRead)),
		}
		resp, err := svc.UploadPart(uploadPartInput)
		if err != nil {
			return err
		}

		// Save the completed part
		completedPart := &s3.CompletedPart{
			ETag:       resp.ETag,
			PartNumber: aws.Int64(int64(partNumber)),
		}
		parts = append(parts, completedPart)
	}

	// Complete the multipart upload
	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(awsHandler.bucket),
		Key:      aws.String(key),
		UploadId: uploadID,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
	}
	_, err = svc.CompleteMultipartUpload(completeInput)
	if err != nil {
		return err
	}

	return nil
}
