FROM --platform=${BUILDPLATFORM} whatwewant/builder-go:v1.21-1 as builder

RUN wget -O /Country.mmdb https://github.com/doreamon-design/clash/releases/download/v2.0.8/Country.mmdb

WORKDIR /build

COPY go.mod ./

COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 \
  go build \
  -trimpath \
  -ldflags '-w -s -buildid=' \
  -v -o /clash ./cmd/clash

FROM whatwewant/alpine:v3.17-1

LABEL org.opencontainers.image.source="https://github.com/doreamon-design/clash"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /Country.mmdb /root/.config/clash/

COPY --from=builder /clash /usr/bin

CMD clash
