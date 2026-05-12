FROM golang:alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM gcr.io/distroless/static-debian13:nonroot

LABEL org.opencontainers.image.source="https://github.com/FrostWalk/pipeline-alert"
LABEL org.opencontainers.image.description="GitLab pipeline failure alert server"

COPY --from=build /server /server

EXPOSE 8080
ENV HOST=0.0.0.0
ENV PORT=8080
ENV GIN_MODE=release

ENTRYPOINT ["/server"]
