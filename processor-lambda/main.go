package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/disintegration/imaging"
)

var (
	s3Client         *s3.S3
	rekognitionClient *rekognition.Rekognition
	resultBucket     = "RESULT_BUCKET_NAME"
)

func init() {
	sess := session.Must(session.NewSession())
	s3Client = s3.New(sess)
	rekognitionClient = rekognition.New(sess)
}

func handler(ctx context.Context, s3Event events.S3Event) (map[string]interface{}, error) {
	results := []map[string]interface{}{}

	for _, record := range s3Event.Records {
		start := time.Now()
		var memStatsBefore runtime.MemStats
		runtime.ReadMemStats(&memStatsBefore)

		bucket := record.S3.Bucket.Name
		key := record.S3.Object.Key
		log.Printf("Processing %s from bucket %s", key, bucket)

		// Download image from S3
		obj, err := s3Client.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Printf("Failed to get object: %v", err)
			continue
		}
		defer obj.Body.Close()

		// Decode and process image
		img, _, err := image.Decode(obj.Body)
		if err != nil {
			log.Printf("Image decode error: %v", err)
			continue
		}

		resized := imaging.Resize(img, img.Bounds().Dx()/2, 0, imaging.Lanczos)
		grayscale := imaging.Grayscale(resized)

		var buffer bytes.Buffer
		if err := png.Encode(&buffer, grayscale); err != nil {
			log.Printf("PNG encode failed: %v", err)
			continue
		}
		processedBytes := buffer.Bytes()

		// Rekognition - detect labels
		labelResp, err := rekognitionClient.DetectLabels(&rekognition.DetectLabelsInput{
			Image: &rekognition.Image{
				Bytes: processedBytes,
			},
			MaxLabels: aws.Int64(5),
		})
		var labels []string
		if err == nil {
			for _, l := range labelResp.Labels {
				labels = append(labels, *l.Name)
			}
		}

		// Rekognition - detect text
		textResp, err := rekognitionClient.DetectText(&rekognition.DetectTextInput{
			Image: &rekognition.Image{
				Bytes: processedBytes,
			},
		})
		var texts []string
		if err == nil {
			for _, t := range textResp.TextDetections {
				if t.DetectedText != nil {
					texts = append(texts, *t.DetectedText)
				}
			}
		}

		// Performance metrics
		var memStatsAfter runtime.MemStats
		runtime.ReadMemStats(&memStatsAfter)
		memoryUsedMB := float64(memStatsAfter.Alloc-memStatsBefore.Alloc) / 1024 / 1024
		executionTime := time.Since(start).Seconds()

		// Upload processed image
		processedKey := "processed/" + key
		_, err = s3Client.PutObject(&s3.PutObjectInput{
			Bucket:      aws.String(resultBucket),
			Key:         aws.String(processedKey),
			Body:        bytes.NewReader(processedBytes),
			ContentType: aws.String("image/png"),
		})
		if err != nil {
			log.Printf("Upload failed: %v", err)
			continue
		}

		// Create and upload JSON metadata
		payload := map[string]interface{}{
			"message":               "Image processed successfully",
			"execution_time":        executionTime,
			"memory_used_MB":        memoryUsedMB,
			"detected_objects":      labels,
			"extracted_text":        texts,
			"processed_image_s3_url": fmt.Sprintf("s3://%s/%s", resultBucket, processedKey),
		}
		jsonBytes, _ := json.Marshal(payload)
		jsonKey := processedKey + ".json"

		_, err = s3Client.PutObject(&s3.PutObjectInput{
			Bucket:      aws.String(resultBucket),
			Key:         aws.String(jsonKey),
			Body:        bytes.NewReader(jsonBytes),
			ContentType: aws.String("application/json"),
		})
		if err != nil {
			log.Printf("Failed to upload JSON: %v", err)
			continue
		}

		results = append(results, payload)
	}

	return map[string]interface{}{
		"statusCode": 200,
		"body":       results,
	}, nil
}

func main() {
	lambda.Start(handler)
}
