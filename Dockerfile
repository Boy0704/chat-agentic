FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o agent-service ./cmd/server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata python3 py3-pip

WORKDIR /app
COPY --from=builder /app/agent-service .

# Install Python dependencies declared by custom skills
COPY custom-skills/requirements.txt ./custom-skills/requirements.txt
RUN pip3 install --no-cache-dir -r custom-skills/requirements.txt

RUN mkdir -p /app/data
EXPOSE 8080
CMD ["./agent-service", "-config", "config.yaml"]
