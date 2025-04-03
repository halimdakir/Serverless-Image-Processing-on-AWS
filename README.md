```markdown
# AWS Lambda Image Uploader & Processor

This repository includes two AWS Lambda functions written in Go and deployed using Docker containers:

1. **Image Uploader Lambda**  
   - Triggered via HTTP (API Gateway)
   - Accepts base64-encoded image via POST request
   - Uploads the image to an **input S3 bucket**

2. **Image Processor Lambda**  
   - Triggered by S3 `ObjectCreated` event
   - Resizes and grayscales the image
   - Uses Amazon Rekognition for object detection and OCR
   - Saves the processed image and JSON metadata to a **result S3 bucket**


---

## Architecture Overview

The following diagram provides a high-level view of how the system components interact from upload to processing:



---

## Prerequisites

- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
- [Docker](https://docs.docker.com/get-docker/)
- IAM Role (`lambda-execution-role`) with necessary permissions for:
  - `s3:GetObject`, `s3:PutObject`, `s3:ListBucket`
  - `logs:*`
  - `rekognition:*` (for processor only)

> Replace all `{{PLACEHOLDER}}` values with your actual AWS settings.

---

## Folder Structure

``` bash
/processor-lambda/
    Dockerfile               # Docker build file for processor Lambda
    go.mod                   # Go module file
    main.go                  # Lambda code: resize, grayscale, Rekognition
    
/results/                # Local test outputs (before/after images + JSON)
    bird.png
    bird (original).png
    bird (processed).png
    bird.json
    formula1.png
    formula1 (original).png
    formula1 (processed).png
    formula1.json

/uploader-lambda/
    Dockerfile               # Docker build file for uploader Lambda
    go.mod                   # Go module file
    main.go                  # Lambda code: decode + upload to S3

benchmark_test.sh        # ApacheBench benchmarking script
convert_image_to_base64.py  # Script to create payload.json from an image
formula1.png             # Sample test image
payload.json             # Test input for uploader Lambda
queries_cloudwatch.pl    # Saved CloudWatch queries (optional)
/README.md                 # Project documentation
```


## Deployment Steps

### 1. Build the Docker Image

```bash
# Uploader
cd uploader-lambda
docker build -t image-uploader .

# Processor
cd ../processor-lambda
docker build -t image-processor .
```

---

### 2. Authenticate with ECR

```bash
aws ecr get-login-password --region {{REGION}} | docker login --username AWS --password-stdin {{ACCOUNT_ID}}.dkr.ecr.{{REGION}}.amazonaws.com
```

---

### 3. Create ECR Repositories

```bash
aws ecr create-repository --repository-name image-uploader --region {{REGION}}
aws ecr create-repository --repository-name image-processor --region {{REGION}}
```

---

### 4. Tag and Push Docker Images

```bash
# Uploader
docker tag image-uploader:latest {{ACCOUNT_ID}}.dkr.ecr.{{REGION}}.amazonaws.com/image-uploader:latest
docker push {{ACCOUNT_ID}}.dkr.ecr.{{REGION}}.amazonaws.com/image-uploader:latest

# Processor
docker tag image-processor:latest {{ACCOUNT_ID}}.dkr.ecr.{{REGION}}.amazonaws.com/image-processor:latest
docker push {{ACCOUNT_ID}}.dkr.ecr.{{REGION}}.amazonaws.com/image-processor:latest
```

---

### 5. Create Lambda Functions

```bash
# Uploader
aws lambda create-function \
  --function-name image-uploader-lambda \
  --package-type Image \
  --code ImageUri={{ACCOUNT_ID}}.dkr.ecr.{{REGION}}.amazonaws.com/image-uploader:latest \
  --role arn:aws:iam::{{ACCOUNT_ID}}:role/lambda-execution-role \
  --region {{REGION}}

# Processor
aws lambda create-function \
  --function-name image-processor-lambda \
  --package-type Image \
  --code ImageUri={{ACCOUNT_ID}}.dkr.ecr.{{REGION}}.amazonaws.com/image-processor:latest \
  --role arn:aws:iam::{{ACCOUNT_ID}}:role/lambda-execution-role \
  --region {{REGION}}
```

---

### 6. S3 Trigger Setup (for Processor)

```bash
aws s3api put-bucket-notification-configuration \
  --bucket {{INPUT_BUCKET_NAME}} \
  --notification-configuration '{
    "LambdaFunctionConfigurations": [
      {
        "LambdaFunctionArn": "arn:aws:lambda:{{REGION}}:{{ACCOUNT_ID}}:function:image-processor-lambda",
        "Events": ["s3:ObjectCreated:*"]
      }
    ]
  }'
```

---

## Sample IAM Role Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "s3:PutObject",
      "Resource": [
        "arn:aws:s3:::{{INPUT_BUCKET_NAME}}/*",
        "arn:aws:s3:::{{RESULT_BUCKET_NAME}}/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:ListBucket"],
      "Resource": [
        "arn:aws:s3:::{{INPUT_BUCKET_NAME}}",
        "arn:aws:s3:::{{INPUT_BUCKET_NAME}}/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "rekognition:DetectLabels",
        "rekognition:DetectText"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    }
  ]
}
```

---

## Testing
#### Sending Payload to the Uploader Lambda

To test the uploader Lambda, you need a valid `payload.json` with a base64-encoded image.  
Use the provided helper script (`convert_image_to_base64.py`) or the example below to generate it:

```bash
python generate_payload.py
```

> This script reads an image file (e.g., `formula1.png`), encodes it in base64, and creates `payload.json`.

Then test using:
```bash
curl -X POST https://{{API_GATEWAY_ENDPOINT}} \
     -H "Content-Type: application/json" \
     -d @payload.json
```

---

### Performance Benchmarking & Cloud Monitoring

#### Load Testing with ApacheBench (ab)

You can test the scalability of the uploader API using the provided benchmarking script (`benchmark.sh`). It simulates increasing and decreasing workloads.

To run:
```bash
chmod +x benchmark.sh
./benchmark.sh
```

This will:
- Hit the API Gateway with multiple concurrency levels
- Save results in files like `result_scale_out_100.txt`

#### CloudWatch Log Insights (Lambda Monitoring)

Use these queries in the [CloudWatch Logs Insights](https://console.aws.amazon.com/cloudwatch/home#logs-insights:) console for detailed Lambda metrics:

- **Total Invocations**
  ```sql
  fields @timestamp, @message
  | filter @message like /REPORT RequestId/
  | stats count(*) as totalInvocations
  ```

- **Timeouts**
  ```sql
  fields @timestamp, @message
  | filter @message like /Task timed out/
  | stats count(*) as timeouts
  ```

- **Cold Starts**
  ```sql
  fields @timestamp, @message
  | filter @message like /INIT_START/
  | stats count(*) as coldStarts
  ```

- **Performance Stats (Duration & Memory)**
  ```sql
  fields @timestamp, @message
  | filter @message like /REPORT RequestId/
  | parse @message /Duration: (?<duration>[0-9.]+) ms.*Max Memory Used: (?<maxMem>\d+) MB/
  | stats
      avg(duration) as avgDurationMs,
      min(duration) as minDurationMs,
      max(duration) as maxDurationMs,
      avg(maxMem) as avgMemoryMB,
      min(maxMem) as minMemoryMB,
      max(maxMem) as maxMemoryMB
  ```

---

## Summary

This solution demonstrates a fully containerized, event-driven image processing pipeline using AWS Lambda, S3, and Rekognition — written in Go, deployed with Docker, and benchmarked for scalability.
