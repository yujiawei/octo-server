# QUICKSTART — OCTO Server

Run a full OCTO stack (server + WuKongIM + MySQL + Redis + MinIO) in
~5 minutes with Docker Compose.

## Prerequisites

- Docker Engine 20+
- Docker Compose v2 (`docker compose`, not `docker-compose`)
- ~2 GB free RAM
- Ports 82, 8090, 5100, 5200, 9000 available

## 1. Clone

```bash
git clone https://github.com/Mininglamp-OSS/octo-server.git
cd octo-server/docker/tsdd
```

## 2. Prepare environment

```bash
cp .env.example .env
```

Edit `.env` and set real values. **Important**: the same MySQL password
must also be set in `configs/tsdd.yaml` under `db.mysqlAddr` — they are
not auto-synced.

Set `MYSQL_ROOT_PASSWORD` to your chosen password, then also update
the `root:<password>` portion in `configs/tsdd.yaml`.

Generate `OCTO_MASTER_KEY` (32 hex chars; encrypts bot private keys at rest):
```bash
echo "OCTO_MASTER_KEY=$(openssl rand -hex 16)" >> .env
```

Generate a secure WuKongIM manager token
```

Generate a WuKongIM manager token:
```bash
openssl rand -hex 16   # paste result into configs/tsdd.yaml wukongIM.managerToken
```

## 3. Build octo-server image

```bash
cd ../..
docker build -t octo-server:local -f Dockerfile .
cd docker/tsdd
```

## 4. (Optional) Build octo-web image

Clone and build [octo-web](https://github.com/Mininglamp-OSS/octo-web):
```bash
git clone https://github.com/Mininglamp-OSS/octo-web.git /tmp/octo-web
cd /tmp/octo-web
docker build -t octo-web:local .
cd -
```

If you skip this step, comment out the `tangsengdaodaoweb` service
in `docker-compose.yaml`.

## 5. Start the stack

```bash
docker compose up -d
```

Wait ~30s for services to initialize. Check health:
```bash
docker compose ps
curl http://localhost:8090/v1/ping
```

## 6. Register your first user

Open http://localhost:82 in your browser (requires octo-web).
Alternatively, use the API directly:

```bash
curl -X POST http://localhost:8090/v1/user/register \
  -H "Content-Type: application/json" \
  -d '{"phone":"+8613800000000","password":"test1234","name":"Admin"}'
```

## 7. Connect an AI Agent

Install the daemon CLI:
```bash
go install github.com/Mininglamp-OSS/octo-daemon-cli@latest
```

In OCTO, send `/daemon` to BotFather to receive your start command.

## Troubleshooting

- **Port conflicts**: Edit the `ports:` block in `docker-compose.yaml`.
- **WuKongIM unhealthy**: Check `configs/wk.yaml` and `logs/wk/`.
- **Go build fails with "missing go.sum entry for octo-lib"**:
  See [BUILDING.md](./BUILDING.md) for the cross-repo `replace` workaround.

## How initial schema works

The MySQL container auto-loads `docker/tsdd/init/00-init-schema.sql` on
first startup via Docker's standard `/docker-entrypoint-initdb.d/` hook.
This seeds both the table schema and the `gorp_migrations` history, so
octo-server's built-in migration engine sees a fully-initialized
database and skips all historical migrations.

When you upgrade octo-server later, only truly-new migrations run.

To regenerate this snapshot from your production schema:
```bash
scripts/refresh-init-schema.sh  # (maintainers only)
```

## Stop & reset

```bash
docker compose down            # stop containers, keep data
docker compose down -v         # stop + delete all volumes (DESTRUCTIVE)
```

