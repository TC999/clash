FROM --platform=${BUILDPLATFORM} whatwewant/builder-go:v1.21-1 as builder

RUN wget -O /Country.mmdb https://github.com/doreamon-design/clash/releases/download/v2.0.8/Country.mmdb

WORKDIR /build

COPY go.mod ./

COPY go.sum ./

RUN go mod download

COPY . .

ARG TARGETOS

ARG TARGETARCH

RUN CGO_ENABLED=0 \
  GOOS=${TARGETOS} \
  GOARCH=${TARGETARCH} \
  go build \
  -trimpath \
  -ldflags '-w -s -buildid=' \
  -v -o /clash ./cmd/clash

FROM whatwewant/builder-node:v20-1 as builder-ui

ADD https://github.com/doreamon-design/clash-dashboard /build

WORKDIR /build

RUN yarn

RUN yarn build

FROM whatwewant/alpine:v3.17-1

ENV TZ=Asia/Shanghai

LABEL org.opencontainers.image.source="https://github.com/doreamon-design/clash"

ENV CLASH_OVERRIDE_EXTERNAL_UI_DIR=/etc/clash/ui

ENV CLASH_CONFIG_FILE=/etc/clash/config.yaml

ENV CLASH_HOME_DIR=/etc/clash

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder-ui /build/dist /etc/clash/ui

COPY --from=builder /Country.mmdb /etc/clash/

COPY --from=builder /clash /usr/bin

CMD clash
