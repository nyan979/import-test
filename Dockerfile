FROM golang:1.18.1-alpine3.15 AS build

COPY ./ar2-import/apis/import /app/ar2-import/apis/import
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app
RUN pwd
ARG CGO_ENABLED=0
RUN go mod download

WORKDIR /app/ar2-import/apis/import
RUN go build -o /app/server

FROM alpine:3.15

WORKDIR /app
COPY --from=build /app/server /app/server

CMD ["./server"]