FROM golang:1.18.1-alpine as builder

WORKDIR /ar2-import

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY lib/ lib/
COPY workflows/ workflows/
COPY apps/ apps/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -installsuffix cgo -o /ar2-import/import ./apps/import

FROM alpine

COPY --from=builder /ar2-import/import /import
COPY ./.env ./.env

CMD ["./import"]