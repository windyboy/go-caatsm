# Secret Management Guide

This document describes best practices for managing secrets and sensitive configuration in the CAATSM application.

## Current Approach

The application currently supports secrets via environment variables with the `CAATSM_` prefix:

```bash
export CAATSM_POSTGRES_URL="postgres://user:password@localhost:5432/aviation"
export CAATSM_NATS_AUTH_TOKEN="your-token"
export CAATSM_NATS_AUTH_PASSWORD="secure-password"
```

**Security considerations:**
- Environment variables are visible to all processes on the system
- Secrets may be logged in process lists or shell history
- No automatic rotation or expiration
- Manual management required

## Recommended Secret Management Systems

For production deployments, use a dedicated secret management system:

### Option 1: HashiCorp Vault (Recommended)

[Vault](https://www.vaultproject.io/) provides secure secret storage with dynamic secrets, encryption, and access control.

#### Setup

1. **Install Vault:**
   ```bash
   # Download and install Vault
   wget https://releases.hashicorp.com/vault/1.15.0/vault_1.15.0_linux_amd64.zip
   unzip vault_1.15.0_linux_amd64.zip
   sudo mv vault /usr/local/bin/
   ```

2. **Start Vault (dev mode for testing):**
   ```bash
   vault server -dev
   ```

3. **Store secrets:**
   ```bash
   export VAULT_ADDR='http://127.0.0.1:8200'
   vault kv put secret/caatsm \
     postgres_url="postgres://user:pass@db:5432/aviation" \
     nats_token="your-token" \
     nats_password="secure-password"
   ```

#### Integration

Create a wrapper script or init container to fetch secrets from Vault:

```bash
#!/bin/bash
# fetch-secrets.sh

export VAULT_ADDR="${VAULT_ADDR:-http://vault:8200}"
export VAULT_TOKEN="${VAULT_TOKEN}"

# Fetch secrets from Vault
vault kv get -format=json secret/caatsm | jq -r '.data.data | to_entries | .[] | "export CAATSM_\(.key | ascii_upcase | gsub("-"; "_"))=\(.value)"' > /tmp/secrets.env

# Source secrets
source /tmp/secrets.env

# Start application
exec ./bin/receiver listen
```

#### Kubernetes Integration

Use Vault Agent Sidecar or Vault Secrets Operator:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: caatsm-receiver
spec:
  containers:
  - name: vault-agent
    image: vault:latest
    command: ["/bin/sh", "-c"]
    args:
      - |
        vault agent -config=/vault/config/agent.hcl
  - name: caatsm-receiver
    image: caatsm/receiver:latest
    envFrom:
    - secretRef:
        name: caatsm-secrets
```

### Option 2: AWS Secrets Manager

For AWS deployments, use [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/) for centralized secret management.

#### Setup

1. **Store secrets:**
   ```bash
   aws secretsmanager create-secret \
     --name caatsm/production \
     --secret-string '{
       "postgres_url": "postgres://user:pass@db:5432/aviation",
       "nats_token": "your-token",
       "nats_password": "secure-password"
     }'
   ```

2. **Retrieve secrets:**
   ```bash
   aws secretsmanager get-secret-value \
     --secret-id caatsm/production \
     --query SecretString \
     --output text | jq -r 'to_entries | .[] | "export CAATSM_\(.key | ascii_upcase | gsub("-"; "_"))=\(.value)"'
   ```

#### Integration

Use AWS SDK or CLI in init container:

```bash
#!/bin/bash
# fetch-aws-secrets.sh

SECRET_JSON=$(aws secretsmanager get-secret-value \
  --secret-id caatsm/production \
  --query SecretString \
  --output text)

echo "$SECRET_JSON" | jq -r 'to_entries | .[] | "export CAATSM_\(.key | ascii_upcase | gsub("-"; "_"))=\(.value)"' > /tmp/secrets.env

source /tmp/secrets.env
exec ./bin/receiver listen
```

### Option 3: Kubernetes Secrets

For Kubernetes deployments, use [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/).

#### Setup

1. **Create secret:**
   ```bash
   kubectl create secret generic caatsm-secrets \
     --from-literal=postgres-url="postgres://user:pass@db:5432/aviation" \
     --from-literal=nats-token="your-token" \
     --from-literal=nats-password="secure-password"
   ```

2. **Use in deployment:**
   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: caatsm-receiver
   spec:
     template:
       spec:
         containers:
         - name: receiver
           image: caatsm/receiver:latest
           env:
           - name: CAATSM_POSTGRES_URL
             valueFrom:
               secretKeyRef:
                 name: caatsm-secrets
                 key: postgres-url
           - name: CAATSM_NATS_AUTH_TOKEN
             valueFrom:
               secretKeyRef:
                 name: caatsm-secrets
                 key: nats-token
   ```

#### Best Practices

- **Encrypt at rest**: Enable encryption for etcd (Kubernetes backend)
- **RBAC**: Restrict access to secrets using Role-Based Access Control
- **External Secrets Operator**: Use [External Secrets Operator](https://external-secrets.io/) for integration with external secret stores

### Option 4: Docker Secrets

For Docker Swarm deployments, use [Docker Secrets](https://docs.docker.com/engine/swarm/secrets/).

#### Setup

1. **Create secret:**
   ```bash
   echo "your-secret-value" | docker secret create caatsm_nats_token -
   ```

2. **Use in service:**
   ```yaml
   version: '3.8'
   services:
     receiver:
       image: caatsm/receiver:latest
       secrets:
         - caatsm_nats_token
       environment:
         - CAATSM_NATS_AUTH_TOKEN_FILE=/run/secrets/caatsm_nats_token
   ```

## Secret Rotation

### Manual Rotation

1. **Update secret** in secret management system
2. **Restart application** to pick up new secret
3. **Verify** application is working correctly
4. **Remove old secret** after verification

### Automated Rotation

For AWS Secrets Manager, enable automatic rotation:

```bash
aws secretsmanager rotate-secret \
  --secret-id caatsm/production \
  --rotation-lambda-arn arn:aws:lambda:region:account:function:rotate-secret
```

For Vault, use dynamic secrets or scheduled rotation policies.

## Security Best Practices

### 1. Principle of Least Privilege

- **Minimal access**: Grant only necessary permissions
- **Service accounts**: Use dedicated service accounts for applications
- **Secret scoping**: Limit secrets to specific services/environments

### 2. Encryption

- **Encryption at rest**: Ensure secrets are encrypted in storage
- **Encryption in transit**: Use TLS for secret retrieval
- **Key management**: Use proper key management (HSM, KMS, etc.)

### 3. Audit and Monitoring

- **Audit logs**: Enable audit logging for secret access
- **Monitoring**: Monitor secret access patterns
- **Alerts**: Set up alerts for unusual access patterns

### 4. Secret Lifecycle

- **Rotation**: Rotate secrets regularly (e.g., every 90 days)
- **Expiration**: Set expiration dates for secrets
- **Revocation**: Have a process for revoking compromised secrets

### 5. Development vs Production

- **Separate stores**: Use different secret stores for dev/staging/prod
- **No production secrets in code**: Never commit production secrets
- **Local development**: Use local secret files or dev vault instance

## Configuration Examples

### Environment Variables (Current)

```bash
# Development
export CAATSM_POSTGRES_URL="postgres://user:pass@localhost:5432/aviation?sslmode=disable"
export CAATSM_NATS_URL="nats://localhost:4222"
export CAATSM_NATS_AUTH_TOKEN="dev-token"

# Production (via secret management)
# Secrets loaded from Vault/AWS/K8s before application start
```

### Configuration File (Not Recommended for Secrets)

```toml
# config.prod.toml
# DO NOT store secrets in config files
# Use environment variables or secret management instead

[postgres]
# URL should come from CAATSM_POSTGRES_URL env var
url = ""  # Empty, will be overridden by env var

[nats.auth]
# Token should come from CAATSM_NATS_AUTH_TOKEN env var
token = ""  # Empty, will be overridden by env var
```

## Secret Injection Patterns

### Pattern 1: Init Container (Kubernetes)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: caatsm-receiver
spec:
  initContainers:
  - name: fetch-secrets
    image: vault:latest
    command: ["/bin/sh", "-c"]
    args:
      - |
        vault kv get -format=json secret/caatsm | \
        jq -r '.data.data | to_entries | .[] | "\(.key | ascii_upcase | gsub("-"; "_"))=\(.value)"' > \
        /shared/secrets.env
    volumeMounts:
    - name: shared-secrets
      mountPath: /shared
  containers:
  - name: receiver
    image: caatsm/receiver:latest
    envFrom:
    - configMapRef:
        name: caatsm-config
    env:
    - name: CAATSM_SECRETS_FILE
      value: /shared/secrets.env
    volumeMounts:
    - name: shared-secrets
      mountPath: /shared
  volumes:
  - name: shared-secrets
    emptyDir: {}
```

### Pattern 2: Sidecar Container

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: caatsm-receiver
spec:
  containers:
  - name: vault-agent
    image: vault:latest
    command: ["vault", "agent", "-config=/vault/config/agent.hcl"]
    volumeMounts:
    - name: vault-config
      mountPath: /vault/config
  - name: receiver
    image: caatsm/receiver:latest
    envFrom:
    - secretRef:
        name: caatsm-secrets
  volumes:
  - name: vault-config
    configMap:
      name: vault-agent-config
```

### Pattern 3: Application-Level Integration

For applications that need to fetch secrets at runtime:

```go
// Example: Fetch secrets from Vault at startup
func loadSecretsFromVault() error {
    client, err := vault.NewClient(vault.DefaultConfig())
    if err != nil {
        return err
    }
    
    secret, err := client.Logical().Read("secret/data/caatsm")
    if err != nil {
        return err
    }
    
    // Set environment variables
    for k, v := range secret.Data["data"].(map[string]interface{}) {
        os.Setenv("CAATSM_"+strings.ToUpper(k), v.(string))
    }
    
    return nil
}
```

## Troubleshooting

### Secret Not Found

**Symptoms:**
- Application fails to start
- Connection errors to database/NATS

**Solutions:**
- Verify secret exists in secret store
- Check secret name/path is correct
- Verify application has permissions to access secret
- Check secret format (JSON, plain text, etc.)

### Secret Access Denied

**Symptoms:**
- Authentication errors when fetching secrets
- Permission denied errors

**Solutions:**
- Verify IAM roles/service accounts have correct permissions
- Check Vault policies or AWS IAM policies
- Verify authentication tokens/credentials are valid

### Secret Rotation Issues

**Symptoms:**
- Application fails after secret rotation
- Connection errors after rotation

**Solutions:**
- Implement graceful secret reloading
- Use connection pooling with automatic reconnection
- Test rotation process in staging first

## References

- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)
- [AWS Secrets Manager Documentation](https://docs.aws.amazon.com/secretsmanager/)
- [Kubernetes Secrets Documentation](https://kubernetes.io/docs/concepts/configuration/secret/)
- [External Secrets Operator](https://external-secrets.io/)
- [12-Factor App: Config](https://12factor.net/config)

