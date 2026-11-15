# up-monitor

A small uptime monitor written in Go.

- Checks a list of URLs concurrently every 10 seconds (one goroutine per URL)
- Only reports state changes (DOWN / RECOVERED), after 3 consecutive failures
- Sends alerts to a Discord webhook
- Stores every check in Postgres
- Serves the latest state of each URL at `GET /status`

## Run it

```
cp .env.example .env     # optionally set DISCORD_WEBHOOK and URLS
docker compose up --build
curl localhost:8080/status
```

## Config (environment variables)

| Variable          | Meaning                                         |
|-------------------|-------------------------------------------------|
| `DATABASE_URL`    | Postgres connection string (set by compose)     |
| `DISCORD_WEBHOOK` | Discord webhook URL; alerts are skipped if empty |
| `URLS`            | Comma-separated URLs to monitor                 |

## Roadmap

- [x] Concurrent checks, state-change alerts, Postgres history, status endpoint
- [x] Dockerfile + docker-compose
- [ ] GitHub Actions CI
- [ ] Deploy to a VPS behind HTTPS
- [ ] Prometheus `/metrics` + Grafana
- [ ] Terraform for the server
