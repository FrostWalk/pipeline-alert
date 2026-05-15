FROM gcr.io/distroless/static-debian13:nonroot

# GoReleaser dockers_v2 context: prebuilt binary at ${TARGETPLATFORM}/server (e.g. linux/amd64/server).
ARG TARGETPLATFORM

LABEL org.opencontainers.image.source="https://github.com/FrostWalk/pipeline-alert"
LABEL org.opencontainers.image.description="GitLab pipeline failure alert server"

COPY ${TARGETPLATFORM}/server /server

EXPOSE 8080
ENV HOST=0.0.0.0
ENV PORT=8080
ENV GIN_MODE=release

ENTRYPOINT ["/server"]
