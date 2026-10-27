# Builds the server into a small image. The host sets PORT and DATABASE_URL.
#   docker build -t game-backend .
#   docker run -p 8080:8080 -e DATABASE_URL=... game-backend

# Build: compile a static binary (migrations are embedded in it).
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# Run: just the binary, as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
