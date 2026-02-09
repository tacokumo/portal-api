FROM golang:1.25 AS builder
WORKDIR /src

COPY ./go.mod ./
COPY ./go.sum ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./pkg ./pkg
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM scratch
COPY --from=builder /src/server /server
CMD ["/server"]