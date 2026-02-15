FROM golang:1.25 AS build
WORKDIR /app

# Deps don't change often, so keep layers separate
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /health-checker .

FROM gcr.io/distroless/static
COPY --from=build /health-checker /health-checker
EXPOSE 8989
ENTRYPOINT ["/health-checker"]