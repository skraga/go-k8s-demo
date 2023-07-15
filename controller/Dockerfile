# syntax=docker/dockerfile:experimental
FROM golang:1.20 AS build

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /build
COPY . /build

RUN --mount=type=cache,target=/root/.cache/go-build \
  go build -ldflags '-s -w' -o bin/demo-controller .

FROM scratch AS runtime

COPY --from=build /build/bin/demo-controller /usr/local/bin/
USER 65532:65532
CMD ["demo-controller"]
