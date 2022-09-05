FROM golang:1.19-bullseye AS build
WORKDIR /app/
COPY go.mod go.sum /app/
RUN go mod download
COPY *.go /app/
RUN CGO_ENABLED=0 go build

FROM gcr.io/distroless/static-debian11:latest
LABEL maintainer "Setuu <setuu@neigepluie.net>"
WORKDIR /app/
COPY *.html.tpl /app/
COPY --from=build /app/xiv-loot-manager /app/
CMD ["/app/xiv-loot-manager"]
