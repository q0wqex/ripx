[English](README.md) | [Русский](README.ru.md)

# Overview

Minimal image hosting service.

Go standard library only.

Filesystem storage.

No database.

Anonymous cookie-based sessions.

Intended to run behind a reverse proxy.

# Dependencies

- Docker
- Docker Compose


# Installation

```bash
git clone https://github.com/project-absolute/ripx.git && cd ripx
```

# Running and the Service

```bash
docker compose up -d
```

The service runs using a prebuilt Docker image from GHCR. No local build is performed.

# Updating the Service

```bash
docker compose pull && docker compose up -d
```

The service is updated by pulling a newer image. Data in /data is preserved.

# Container Image

- Image: ghcr.io/project-absolute/ripx:[main / v.*.*.* / dev]

# Reverse Proxy Example

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8000;

        # Allow large uploads (10MB limit)
        client_max_body_size 10M;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

# Notes

- Anonymous cookies
- No user accounts
- No API
- Intended for self-hosted use
