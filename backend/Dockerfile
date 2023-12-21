FROM golang:1.21 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build

FROM debian:12-slim
COPY --from=builder /app/elm-backend /usr/local/bin/elm-backend
CMD [ "/usr/local/bin/elm-backend" ]
EXPOSE 80
