FROM golang:1.24.1 AS build
WORKDIR /app
COPY . .

ARG VERSION
ARG COMMIT
ARG DATE

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.commit=$COMMIT -X main.version=$VERSION -X main.date=$DATE" github.com/planetscale/cli/cmd/pscale

FROM ubuntu:noble
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates mysql-client openssh-client \
    && rm -rf /var/lib/apt/lists/*
ENV LANG=C.utf8
EXPOSE 3306

WORKDIR /app
COPY --from=build /app/pscale /usr/bin
ENTRYPOINT ["/usr/bin/pscale"]
