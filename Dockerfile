# --- build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Build.
COPY . .
RUN go run github.com/a-h/templ/cmd/templ@v0.3.960 generate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/ghstats ./cmd/server

# --- runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ghstats /ghstats
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/ghstats"]
