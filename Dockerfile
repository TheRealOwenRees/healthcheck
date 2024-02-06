FROM golang:1.21.6-alpine3.19
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY cmd/main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/healthcheck main.go
CMD ["bin/healthcheck"]
