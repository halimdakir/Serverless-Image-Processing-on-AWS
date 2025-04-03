FROM public.ecr.aws/lambda/go:1

# Copy and build the Go app
COPY main.go ./
RUN go build -o main main.go

# Set the CMD to your handler binary
CMD ["main"]
