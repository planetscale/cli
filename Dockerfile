FROM golang:1.27.3@sha256:e519d3b0a5b285e5aa53ab9eef02bc6bbb9cb04608389b743a136a9ce75f0d1e AS build
WORKDIR /app
COPY . .

ARG VERSION
ARG COMMIT
ARG DATE

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.commit=$COMMIT -X main.version=$VERSION -X main.date=$DATE" github.com/planetscale/cli/cmd/pscale

FROM ubuntu:noble@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852
ARG MYSQL_VERSION=8.4.13.1
# Oracle MySQL client from the official MySQL APT repo (amd64 only, matching the amd64 build above).
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates curl gpg openssh-client; \
    curl -fsSL https://repo.mysql.com/RPM-GPG-KEY-mysql-2025 -o /tmp/mysql.key; \
    gpg --show-keys --with-colons /tmp/mysql.key | grep -qx 'fpr:::::::::BCA43417C3B485DD128EC6D4B7B3B788A8D3785C:'; \
    gpg --dearmor -o /usr/share/keyrings/mysql.gpg /tmp/mysql.key; \
    echo 'deb [signed-by=/usr/share/keyrings/mysql.gpg] https://repo.mysql.com/apt/ubuntu noble mysql-8.4-lts' > /etc/apt/sources.list.d/mysql.list; \
    apt-get update; \
    apt-get install -y --no-install-recommends "mysql-community-client-core=${MYSQL_VERSION}-1ubuntu24.04"; \
    rm /etc/apt/sources.list.d/mysql.list; \
    apt-get purge -y --auto-remove curl gpg; \
    rm -rf /var/lib/apt/lists/* /tmp/* /usr/share/keyrings/mysql.gpg /root/.gnupg
ENV LANG=C.utf8
EXPOSE 3306

WORKDIR /app
COPY --from=build /app/pscale /usr/bin
ENTRYPOINT ["/usr/bin/pscale"]
