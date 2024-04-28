FROM golang:1.22 AS build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/server .
COPY --from=build /app/.env .
ENTRYPOINT [ "./server" ]