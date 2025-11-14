# Releasing Guide

Use this guide to ship consistent, verifiable releases.

## 1. Cut a Release Branch

```bash
git checkout -b release/vX.Y.Z
```

## 2. Verify Quality Gates

1. Run `go test ./... -coverprofile=cover.out -covermode=atomic` and ensure coverage ≥ 80%.
2. Run GolangCI-Lint locally (`golangci-lint run ./...`) to catch regressions early.
3. (Optional) Execute integration tests with Docker: `./tests/integration/run.sh up && go test ./tests/integration -tags=integration`.
4. Update `docs/CHANGELOG.md` (create if missing) with highlights, fixes, and breaking changes.

## 3. Update Version Metadata

- Bump module/service version references (Docker tags, Helm chart values, etc.).
- Commit with a conventional message, e.g. `chore: prepare release vX.Y.Z`.

## 4. Tag & Push

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin release/vX.Y.Z
git push origin vX.Y.Z
```

## 5. Publish Container Images

```bash
docker build -f docker/Dockerfile.api -t ghcr.io/your-org/go-caatsm-api:vX.Y.Z .
docker build -f docker/Dockerfile.worker -t ghcr.io/your-org/go-caatsm-worker:vX.Y.Z .
docker push ghcr.io/your-org/go-caatsm-api:vX.Y.Z
docker push ghcr.io/your-org/go-caatsm-worker:vX.Y.Z
```

Optionally mark `:latest` after successful production rollout:

```bash
docker tag ghcr.io/your-org/go-caatsm-api:vX.Y.Z ghcr.io/your-org/go-caatsm-api:latest
docker push ghcr.io/your-org/go-caatsm-api:latest
```

## 6. Draft GitHub Release

1. Create a release against tag `vX.Y.Z`.
2. Paste changelog highlights, breaking changes, and upgrade notes (link to `docs/UPGRADE.md`).
3. Attach artifacts (coverage report, SBOM, docker-compose bundle) if needed.

## 7. Post-Release

- Monitor CI/CD pipelines, JetStream backlog, and alert channels.
- If hotfixes are required, branch from the release tag (`git checkout -b hotfix/vX.Y.Z+1 vX.Y.Z`) and repeat this workflow.

