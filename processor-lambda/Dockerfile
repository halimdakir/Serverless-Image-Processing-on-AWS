FROM public.ecr.aws/lambda/go:1

COPY main.go ./
RUN go build -o main main.go

CMD ["main"]
