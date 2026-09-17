# ============================================
# Stage 1：构建
# ============================================
FROM golang:1.25-alpine AS builder
WORKDIR /app

# 【改动】由构建参数注入代理，不再硬编码GOPROXY，二选一
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV NO_PROXY=${NO_PROXY}
# 公有模块用goproxy，私有模块自动走上面socks代理
ENV GOPROXY=https://goproxy.cn,direct

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# 编译3个二进制，保持不变
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/indexer ./cmd/indexer
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/consumer ./cmd/consumer

# ============================================
# Stage 2：运行 alpine:3.19（3.19阿里云源稳定，保留sed）
# ============================================
FROM alpine:3.19
WORKDIR /app

# 运行层 alpine3.19 保留阿里云源，没问题
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/api /app/api
COPY --from=builder /app/indexer /app/indexer
COPY --from=builder /app/consumer /app/consumer

ENV TZ=Asia/Shanghai
CMD ["/app/api"]
