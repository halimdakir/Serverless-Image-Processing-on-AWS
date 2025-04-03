package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	s3Client   *s3.S3
	bucketName = "INPUT_BUCKET_NAME"
)

func init() {
	sess := session.Must(session.NewSession())
	s3Client = s3.New(sess)
}

type RequestBody struct {
	ImageBase64 string `json:"image_base64"`
	FileName    string `json:"file_name"`
}

func handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Event received: %+v", event)

	if event.Body == "" {
		return response(400, `{"error": "Missing request body"}`), nil
	}

	body := event.Body
	if event.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return response(400, fmt.Sprintf(`{"error": "Failed to decode base64 body: %v"}`, err)), nil
		}
		body = string(decoded)
	}

	var payload RequestBody
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return response(400, fmt.Sprintf(`{"error": "Invalid JSON: %v"}`, err)), nil
	}

	if payload.ImageBase64 == "" {
		return response(400, `{"error": "Missing 'image_base64' field"}`), nil
	}

	fileName := payload.FileName
	if strings.TrimSpace(fileName) == "" {
		fileName = "uploaded-image.png"
	}

	imageData, err := base64.StdEncoding.DecodeString(payload.ImageBase64)
	if err != nil {
		return response(400, fmt.Sprintf(`{"error": "Failed to decode image: %v"}`, err)), nil
	}

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String("images/" + fileName),
		Body:        strings.NewReader(string(imageData)),
		ContentType: aws.String("image/png"),
	})
	if err != nil {
		return response(500, fmt.Sprintf(`{"error": "Failed to upload to S3: %v"}`, err)), nil
	}

	return response(200, `{"message": "Image uploaded to S3 successfully!"}`), nil
}

func response(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       body,
	}
}

func main() {
	lambda.Start(handler)
}
