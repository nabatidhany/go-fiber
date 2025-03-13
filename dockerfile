# Gunakan Alpine untuk image yang lebih ringan
FROM golang:1.23.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set environment untuk build
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set folder kerja dalam container
WORKDIR /app

# Copy module files terlebih dahulu (untuk caching)
COPY go.mod go.sum ./  
RUN go mod download

# Copy seluruh file project
COPY . .

# Build aplikasi dengan optimasi ukuran binary
RUN go build -ldflags="-s -w" -o main .

# Gunakan Alpine untuk runtime yang lebih ringan
FROM alpine:latest  

# Install `tzdata` di tahap runtime
RUN apk add --no-cache tzdata  

# Set folder kerja dalam container
WORKDIR /root/

# Set zona waktu ke Asia/Jakarta
ENV TZ=Asia/Jakarta

# Copy binary dari tahap sebelumnya
COPY --from=builder /app/main .

# Expose port untuk Fiber (default 3000)
EXPOSE 3000

# Jalankan aplikasi
CMD ["./main"]
