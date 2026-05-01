FROM ubuntu:latest
LABEL authors="federico"

ENTRYPOINT ["top", "-b"]