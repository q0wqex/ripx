# Ripx

[![Stars](https://img.shields.io/github/stars/project-absolute/ripx.svg?style=social)](https://github.com/project-absolute/ripx/stargazers)
[![Docker Image](https://img.shields.io/badge/docker-ghcr.io-blue?logo=docker)](https://github.com/project-absolute/ripx/pkgs/container/ripx)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

[🇷🇺 Русский](README.md) | [🇺🇸 English](README_EN.md)

## Description

A lightweight and fast image hosting service written in Go. Created as a simple alternative for those seeking minimalism and speed.

> *Ripx — when Yapx went RIP, but the habit of simple hosting remained.*

## Features

- **🖼️ Image Support**: Upload and view JPEG, PNG, GIF, WebP.
- **📁 Albums**: Organize images into albums.
- **🧹 Auto-Cleanup**: Automatic removal of old files (default 60 days).
- **🚀 Ultra Fast**: Minimal dependencies, using cached templates.
- **🐳 Docker Ready**: Full support for containerization and one-command deployment.
- **📱 Responsive UI**: Adaptive interface for mobile and desktop.

## API

The server runs on port `8000` by default.

| Endpoint | Method | Description |
|----------|---------|-------------|
| `/` | GET | Main page / Album view |
| `/upload` | POST | Upload an image |
| `/create-album` | POST | Create a new album |
| `/delete-image` | POST | Delete a specific image |
| `/delete-album` | POST | Delete an entire album |
| `/delete-user` | POST | Delete user and all their data |
| `/changelog` | GET | View change history |

## Configuration

Core parameters are defined in `app/config.go`:

| Variable | Default Value | Description |
|----------|---------------|-------------|
| `MaxFileSize` | `10 * 1024 * 1024` (10MB) | Maximum upload file size |
| `CleanupDuration` | `60 days` | File storage duration before deletion |
| `CleanupInterval` | `24 hours` | Frequency of old file checks |
| `DataPath` | `/data` | Path to image storage directory |
| `ServerAddr` | `0.0.0.0:8000` | Server address and port |

## Setup Instructions

### 1. Quick Start (Docker)

Start the service with a single command (creates data folder, downloads config, and starts container):

```bash
mkdir -p ripx/data && cd ripx && curl -O https://raw.githubusercontent.com/project-absolute/ripx/main/docker-compose.yml && docker-compose up -d
```

### 2. Manual Installation

If you want to build from source:

1. Clone the repository:
   ```bash
   git clone https://github.com/project-absolute/ripx.git && cd ripx
   ```
2. Build and run:
   ```bash
   cd app && go build -o ripx && ./ripx
   ```

## Reverse Proxy Configuration

Nginx is recommended as a reverse proxy for security and request body limits.

<details>
<summary>Nginx Configuration Example</summary>

```nginx
location / {
    proxy_pass http://127.0.0.1:8000;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # Disable buffering for instant content delivery
    proxy_buffering off;
    client_max_body_size 10M;
}
```

</details>

## Project Structure

- `/app` — Go server source code.
- `/app/templates` — HTML templates and static files (JS/CSS).
- `/data` — Image storage (created automatically).
- `docker-compose.yml` — Docker deployment file.
- `changelog.md` — Project history.
