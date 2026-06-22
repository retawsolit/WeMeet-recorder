FROM golang:1.24 AS builder

ARG TARGETPLATFORM
ARG TARGETARCH

WORKDIR /go/src/app

COPY main.go go.mod go.sum ./
RUN go mod download

COPY helpers/ helpers/
COPY pkg/ pkg/
COPY version/ version/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH GO111MODULE=on \
    go build -trimpath -ldflags '-w -s -buildid=' \
    -a -o WeMeet-recorder main.go

# =======================

FROM ubuntu:24.04

ARG TARGETPLATFORM

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl wget gnupg unzip \
    fonts-noto fonts-liberation \
    pulseaudio ffmpeg xvfb \
    libnspr4 libnss3 libu2f-udev xdg-utils \
    libvulkan1 libdrm2 libgbm1 libasound2t64 \
    libxdamage1 libxfixes3 libxrandr2 libxkbcommon0 \
    libgtk-3-0 libatk-bridge2.0-0 libatk1.0-0 \
    libcups2 libxcomposite1 libx11-xcb1 libxcb-dri3-0 \
    libxshmfence1 ca-certificates \
 && rm -rf /var/lib/apt/lists/*

COPY --from=livekit/chrome-installer:125.0.6422.141 /chrome-installer /chrome-installer
RUN /chrome-installer/install-chrome "$TARGETPLATFORM"

# cleanup
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

# copy binary
COPY --from=builder /go/src/app/WeMeet-recorder /usr/bin/WeMeet-recorder
COPY docker-build/entrypoint.sh /
RUN chmod +x /entrypoint.sh

# ===== USER MODE (QUAN TRỌNG) =====
RUN useradd -m -s /bin/bash wemeet

RUN mkdir -p /home/wemeet/.config/pulse /tmp/xdg /app && \
    chown -R wemeet:wemeet /home/wemeet /tmp/xdg /app

USER wemeet

ENV HOME=/home/wemeet
ENV XDG_RUNTIME_DIR=/tmp/xdg
ENV PATH=${PATH}:/chrome

ENTRYPOINT ["/entrypoint.sh"]