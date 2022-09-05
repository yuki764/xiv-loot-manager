FROM golang:1.19-bullseye AS builder
WORKDIR /app/
COPY go.mod go.sum /app/
RUN go mod download
COPY *.go /app/
RUN go build

FROM gcr.io/distroless/base-debian11:latest
LABEL maintainer "Setuu <setuu@neigepluie.net>"
WORKDIR /app/
COPY *.html.tpl /app/
COPY --from=builder /app/xiv-loot-manager /app/
CMD ["/app/xiv-loot-manager"]
