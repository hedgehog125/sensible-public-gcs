FROM golang:1.21.6-bookworm

RUN apt-get update

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY ./ ./

# Build
RUN go build main.go

# Run
CMD ["./main"]
#CMD sleep infinity