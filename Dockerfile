# syntax=docker/dockerfile:1
FROM rust:trixie AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        libgtk-4-dev \
        libgtk4-layer-shell-dev \
        libpoppler-glib-dev \
        libwayland-dev \
        pkg-config \
        protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY Cargo.toml Cargo.lock rust-toolchain.toml build.rs ./
COPY src ./src
COPY resources ./resources

RUN cargo build --locked --release \
    && install -Dm755 target/release/walker /out/walker \
    && strip /out/walker \
    && chown -R 1000:1000 "$CARGO_HOME"

FROM debian:trixie-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        adwaita-icon-theme \
        ca-certificates \
        fonts-dejavu-core \
        gstreamer1.0-libav \
        gstreamer1.0-plugins-base \
        gstreamer1.0-plugins-good \
        libgtk-4-1 \
        libgtk-4-media-gstreamer \
        libgtk4-layer-shell0 \
        libpoppler-glib8t64 \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --uid 1000 walker

COPY --from=builder /out/walker /usr/local/bin/walker
COPY LICENSE /usr/local/share/licenses/walker/LICENSE

ENV HOME=/home/walker GDK_BACKEND=wayland
USER walker
WORKDIR /home/walker
ENTRYPOINT ["walker"]
