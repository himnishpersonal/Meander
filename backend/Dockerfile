FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/meander-api ./cmd/api
RUN mkdir -p /out/artifacts

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build --chown=65532:65532 /out/meander-api /app/meander-api
COPY --from=build --chown=65532:65532 /src/fixtures /app/fixtures
COPY --from=build --chown=65532:65532 /out/artifacts /app/output

ENV PORT=8080
ENV MEANDER_FIXTURES=/app/fixtures
ENV MEANDER_OUTPUT=/app/output
EXPOSE 8080

ENTRYPOINT ["/app/meander-api"]
