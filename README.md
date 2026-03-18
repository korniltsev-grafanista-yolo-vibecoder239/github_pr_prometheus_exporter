# github_pr_prometheus_exporter

Monitors open GitHub pull requests and pushes metrics to Grafana Cloud via Prometheus remote write.

## Running locally

1. Copy the example env file and fill in your secrets:

```bash
cp .env.example .env
# edit .env with your values
```

2. Start the exporter:

```bash
docker compose up --build
```

This builds the image and runs the poll loop. Metrics are pushed to your configured Grafana Cloud endpoint on each interval.

To run in the background:

```bash
docker compose up --build -d
docker compose logs -f
```

To stop:

```bash
docker compose down
```
