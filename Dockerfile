# Use the official Golang image from the Docker Hub as the base image
FROM golang:1.23-alpine as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go module files to the container
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod tidy

# Copy the Go application into the container
COPY . .

# Build the Go binary
#RUN GOOS=linux GOARCH=amd64 go build -o main .
RUN GOOS=linux GOARCH=amd64 go build -o main main.go


# Use the official Amazon Linux image for Lambda functions
FROM public.ecr.aws/lambda/go:1

# Copy the pre-built binary from the builder stage
#COPY --from=builder . .
COPY --from=builder /app/main .

# Set the CMD to your Lambda function handler
#CMD [ "main" ]
CMD ["./main"]