# ใช้ alpine ตัวล่าสุดที่มักจะ update go เป็นเวอร์ชันใหม่เสมอ
FROM golang:alpine

# ติดตั้ง git (จำเป็นสำหรับ go install)
RUN apk add --no-cache git

WORKDIR /app

# คราวนี้คุณสามารถใช้ @latest ของ Air ได้เลย เพราะ Go version ในเครื่องจะเป็นตัวใหม่ล่าสุด
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
# ถ้า go.mod เป็น 1.25.3 ตัวเครื่องใน Docker นี้จะรองรับทันที
RUN go mod download

COPY . .

CMD ["air"]