FROM --platform=linux/amd64 golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main .

# Production stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Creating webdata directory in the CURRENT working directory
RUN mkdir -p webdata

# 설정은 전부 환경변수로 받는다. 예전에는 .env.yml 을 이미지에 구워 넣었는데,
# 그러면 DB 비밀번호가 이미지에 박히고 값을 바꿀 때마다 다시 빌드해야 했다.
# config.Init 은 .env.yml 이 없으면 조용히 건너뛰고 환경변수만 쓴다.
# gym_management/back 도 같은 방식이다.
#
# 필수: PORT, DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASS, CORS
# (PORT 를 빠뜨리면 80 으로 떨어져 VIRTUAL_PORT 와 어긋난다)
ENV APP_MODE=production

CMD ["./main"]