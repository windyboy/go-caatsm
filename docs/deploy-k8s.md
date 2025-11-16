# Kubernetes Deployment Example

The following manifest shows the essential pieces needed to run `caatsm` on Kubernetes with native config management and probes.

## 1. ConfigMap and Secret

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: caatsm-config
data:
  config.toml: |
    # trimmed for brevity; see configs/config.prod.toml
    [nats]
    url = "nats://nats.jetstream.svc:4222"
    stream = "TELEGRAM"
    consumer = "telegram-consumer"

    [postgres]
    # default overridden by CAATSM_POSTGRES_URL in env
    url = "postgres://caatsm:password@timescale.svc:5432/aviation?sslmode=disable"

    [monitoring]
    addr = ":2112"
    enable_metrics = true
    enable_health = true
---
apiVersion: v1
kind: Secret
metadata:
  name: caatsm-secrets
type: Opaque
stringData:
  CAATSM_POSTGRES_URL: postgres://caatsm:super-secret@timescale.svc:5432/aviation?sslmode=disable
```

## 2. Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: caatsm
spec:
  replicas: 1
  selector:
    matchLabels:
      app: caatsm
  template:
    metadata:
      labels:
        app: caatsm
    spec:
      containers:
        - name: caatsm
          image: ghcr.io/<org>/caatsm:latest
          args: ["listen", "--config", "/etc/caatsm/config.toml"]
          envFrom:
            - secretRef:
                name: caatsm-secrets
          env:
            - name: GO_ENV
              value: prod
          ports:
            - name: monitoring
              containerPort: 2112
          volumeMounts:
            - name: config
              mountPath: /etc/caatsm
          livenessProbe:
            httpGet:
              path: /livez
              port: monitoring
            initialDelaySeconds: 10
            periodSeconds: 15
          readinessProbe:
            httpGet:
              path: /readyz
              port: monitoring
            initialDelaySeconds: 5
            periodSeconds: 15
      volumes:
        - name: config
          configMap:
            name: caatsm-config
```

## 3. Service and Scraping

```yaml
apiVersion: v1
kind: Service
metadata:
  name: caatsm-metrics
  labels:
    app: caatsm
spec:
  selector:
    app: caatsm
  ports:
    - name: http
      port: 2112
      targetPort: monitoring
      protocol: TCP
```

Point Prometheus at the service above (or annotate it if you use `prometheus-operator`). The `/readyz` probe surfaces upstream connectivity issues, while `/livez` is used solely for liveness.

