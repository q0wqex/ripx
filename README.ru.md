[English](README.md) | [Русский](README.ru.md)

# Обзор

Минимальный сервис хостинга изображений.

Только стандартная библиотека Go.

Хранение в файловой системе.

Без базы данных.

Анонимные сессии на основе cookie.

Предназначен для работы за обратным прокси-сервером.

# Зависимости

- Docker
- Docker Compose

# Установка

```bash
git clone https://github.com/project-absolute/ripx.git && cd ripx
```

# Запуск сервиса

```bash
docker compose up -d
```

Сервис работает с использованием предварительно собранного Docker образа из GHCR. Локальная сборка не выполняется.

# Обновление сервиса

```bash
docker compose pull && docker compose up -d
```

Сервис обновляется путём загрузки нового образа. Данные в /data сохраняются.

# Контейнерный образ

- Образ: ghcr.io/project-absolute/ripx:[main / v.*.*.* / dev]

# Пример обратного прокси-сервера

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

# Примечания

- Анонимные cookie
- Нет учётных записей пользователей
- Нет API
- Предназначен для собственного хостинга
