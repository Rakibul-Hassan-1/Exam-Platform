FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/api ./api
EXPOSE 8080
ENTRYPOINT ["./api"]
