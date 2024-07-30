# Use the official Golang image as the base image
FROM golang:1.22.3-alpine

# Install necessary dependencies
RUN apk add --no-cache git

# Set the working directory inside the container
WORKDIR /postgo

# Copy the go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go application
RUN go build

# Expose the port the app runs on (80 in this case)
EXPOSE 80

# Command to run the executable
CMD ["./postgo"]
