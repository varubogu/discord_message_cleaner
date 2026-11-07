# build
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

# production image
FROM gcr.io/distroless/static-debian12:nonroot AS production-image
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /out/bot /bot
USER nonroot:nonroot
ENTRYPOINT ["/bot"]
