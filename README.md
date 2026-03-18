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

## Local testing with Prometheus and Grafana

The `example/` directory contains a compose stack that runs the exporter alongside Prometheus and Grafana so you can test everything locally without a Grafana Cloud account.

1. Set up the env file with just your GitHub token:

```bash
cd example
cp .env.example .env
# edit .env — only GITHUB_TOKEN and GITHUB_REPOS are needed
```

2. Start the stack:

```bash
docker compose up --build
```

3. Open Grafana at http://localhost:3000 (no login required) and explore the `github_pr_open` metric via the pre-configured Prometheus datasource. Prometheus UI is at http://localhost:9090.
