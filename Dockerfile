# ========================
# Stage 1: Build & Test  <-- 更新階段名稱 (可選，但更清晰)
# ========================
FROM golang:1.23-alpine AS builder

# 設定環境變數 (通常測試不需要 release 模式，但這裡保留)
ENV GIN_MODE=release
# ENV CONFIG_NAME=config-docker # 這個通常測試時不需要

WORKDIR /app

# 預先載入模組以利用 Docker 快取
COPY go.mod go.sum ./
RUN go mod download

# 複製所有原始碼
COPY . .

# --- 新增：執行單元測試 ---
# 在這裡執行測試。如果任何測試失敗，`go test` 會返回非零結束代碼，
# 這會導致 Docker build 在此步驟失敗停止，不會繼續建置映像檔。
# 使用 ./... 來遞迴執行所有子目錄下的測試。
# -v 參數可以顯示更詳細的測試輸出 (每個測試的 PASS/FAIL 狀態)。
RUN go test -v ./internal/...

# --- 原有的建置步驟 ---
# 只有在上面的 go test 成功 (結束代碼為 0) 後，才會執行這個建置指令。
# 建立 Go 應用 binary（輸出在 ./hr-app）
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o hr-app ./cmd/server

# ========================
# Stage 2: Run
# ========================
FROM alpine:latest

WORKDIR /app

# 從 builder 複製 binary 到 /app
COPY --from=builder /app/hr-app .

# EXPOSE port
EXPOSE 8080

# 預設啟動指令
CMD ["./hr-app"]