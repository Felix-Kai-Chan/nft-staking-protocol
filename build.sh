#!/bin/bash

# 获取 Mac 局域网 IP
MAC_IP=$(ipconfig getifaddr en0)

docker-compose build \
  --build-arg HTTP_PROXY=socks5://${MAC_IP}:7897 \
  --build-arg HTTPS_PROXY=socks5://${MAC_IP}:7897 \
  --build-arg NO_PROXY=localhost,127.0.0.1 \
  "$@"