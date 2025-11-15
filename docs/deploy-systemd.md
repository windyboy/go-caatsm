# Systemd Deployment Example

This example targets a single host running the `caatsm` binary under systemd with minimal moving parts.

## 1. Install the Binary

```
sudo install -m 755 bin/receiver /usr/local/bin/caatsm
sudo install -d /etc/caatsm/configs
sudo cp configs/config.dev.toml /etc/caatsm/configs/config.prod.toml
```

Adjust the config file to point at your production NATS cluster, TimescaleDB endpoint, and telemetry collector.

## 2. Environment File

Create `/etc/caatsm/caatsm.env` to hold secrets or overrides (systemd keeps file permissions intact):

```
CAATSM_NATS_URL=nats://nats.prod.svc.cluster.local:4222
CAATSM_POSTGRES_URL=postgres://caatsm:***@tsdb.prod:5432/aviation?sslmode=require
CAATSM_LOG_LEVEL=info
GO_ENV=prod
```

## 3. systemd Unit

`/etc/systemd/system/caatsm.service`

```
[Unit]
Description=CAATSM Telegram Processor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/caatsm/caatsm.env
WorkingDirectory=/etc/caatsm
ExecStart=/usr/local/bin/caatsm listen
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

Reload and start:

```
sudo systemctl daemon-reload
sudo systemctl enable --now caatsm
```

## 4. Observability Hooks

- Expose `monitoring.addr = ":2112"` (default) and add firewall rules so Prometheus can scrape `http://host:2112/metrics`.
- systemd watchdogs can use `curl -sf http://127.0.0.1:2112/healthz`.

With these three files (binary, config, env) the service becomes repeatable and easy to operate.

