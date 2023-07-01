# syntax=docker/dockerfile:experimental
FROM golang:1.18 AS build

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /build
COPY . /build

RUN --mount=type=cache,target=/root/.cache/go-build \
  go build -o bin/demo-webhook .

FROM scratch AS runtime

COPY --from=build /build/bin/demo-webhook /usr/local/bin/

CMD ["demo-webhook"]
