ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# DaisyUI doesn't work with Tailwind standalone cli :(
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
  && apt-get install -y nodejs

COPY . .

RUN npm ci
RUN npm run build-minify

RUN go build -v -o /run-app .


FROM debian:bookworm

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates
RUN update-ca-certificates

RUN apt-get install -y curl
# Latest releases available at https://github.com/aptible/supercronic/releases
ENV SUPERCRONIC_URL=https://github.com/aptible/supercronic/releases/download/v0.2.31/supercronic-linux-amd64 \
  SUPERCRONIC=supercronic-linux-amd64 \
  SUPERCRONIC_SHA1SUM=fb4242e9d28528a76b70d878dbf69fe8d94ba7d2

RUN curl -fsSLO "$SUPERCRONIC_URL" \
  && echo "${SUPERCRONIC_SHA1SUM}  ${SUPERCRONIC}" | sha1sum -c - \
  && chmod +x "$SUPERCRONIC" \
  && mv "$SUPERCRONIC" "/usr/local/bin/${SUPERCRONIC}" \
  && ln -s "/usr/local/bin/${SUPERCRONIC}" /usr/local/bin/supercronic

COPY tasks/crontab /etc/cron.d/app-tasks
COPY tasks/reset_monthly_usage.sh /usr/local/bin/

COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app"]
