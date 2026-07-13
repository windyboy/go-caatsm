# CAATSM — Full Project Review

**Project:** `caatsm` — Civil Aviation Authority Telegram Message Processor
**Reviewer:** Forge
**Date:** 2026-07-13
**Branch reviewed:** `feature/parser-packages` (latest by commit date)
**Scope:** Full source review — architecture, packages, functions, code, and documentation.

> **Organization:** This review is structured top-down — from the largest architectural decisions down to line-level code and documentation accuracy.
>
> **Update log:**
> - *Initial pass* — architecture → package → function → code → docs.
> - *Follow-up pass (§9)* — examined the runtime/data-flow paths the first pass skipped (processor, publisher, consumer concurrency, regex timeout, weather parser) and surfaced additional data-loss risks plus corrections to two earlier findings (§4.1 likelihood, §2.6 fix). Also corrected the message-type count (no `PLN` type; four unparsed types, not five).

---

## 1. Executive Summary

CAATSM is a Go service implementing Clean Architecture: it ingests aviation/AFTN telegrams from NATS JetStream, parses them (aviation + weather sub-parsers), persists to PostgreSQL/TimescaleDB, and republishes structured JSON. The code is generally readable, uses Wire for DI, OpenTelemetry + Prometheus for observability, and has a broad Ginkgo test suite.

The first pass surfaced **one critical build/reproducibility blocker**, **two data-correctness risks**, and a **large amount of dead/unwired code** (notably an entire schedule parser and four domain message types that are never parsed). A follow-up pass over the runtime paths surfaced **additional data-loss risks on the publish path**: a failed publish leaves inconsistent cross-table state and, due to over-narrow transient classification, is permanently dropped; the NATS dedup header is ineffective (random UUID); and a duplicate-detected insert causes re-publish. Several documented behaviors do not match the implementation.

### Severity at a glance

| Level | Severity | Finding |
|-------|----------|---------|
| Package | 🔴 Critical | `go.sum` missing → build/test fail out of the box |
| Function | 🔴 High | Idempotency race; no unique constraint on `(message_id, date_time)` |
| Function | 🔴 Med-High | Publish failure → inconsistent `telegrams`(parsed)+`telegrams_raw`(error)+DLQ state |
| Function | 🔴 Med-High | Narrow transient check → publish failures permanently lost |
| Architecture | 🟠 Medium | DLQ publish failure still ACKs original → silent loss |
| Package | 🟠 Medium | `schedule` parser package never wired into pipeline |
| Package | 🟠 Medium | Domain types `ALN`/`CHG`/`CPL`/`SITA` have no parser (`pln.go` = schedule helpers, no `PLN` type) |
| Architecture | 🟠 Medium | `Parser` interface lives in `adapter`, not `port` (doc mismatch) |
| Code | 🟡 Low | `interface{}`/`any` in public interfaces (violates "avoid `any`") |
| Function | 🟡 Low | OTel recorder mostly a no-op |
| Architecture | 🟡 Low | Deprecated `nats.JetStreamContext`; global zap logger |
| Package | 🟡 Low | `serial reader` metric naming misleading |
| Function | 🟡 Low | `EffectiveSubscriptionTopic` legacy-only |
| Docs | 🟡 Low | Multiple doc/implementation mismatches |

---

## 2. Architecture-Level Findings

### 2.1 Clean Architecture: `Parser` interface in the wrong layer
The central `Parser` abstraction is defined in `internal/adapter/parser/parser.go:6-9` (the **adapter** layer), not in `internal/port` as the architecture intends. `internal/port` contains only `publisher.go`, `repository.go`, `weather_parser.go`.

`CLAUDE.md:187` states *"Each parser implements the `Parser` interface from `internal/port/parser.go`"* — that file does not exist.

**Impact:** `internal/app` depends on `internal/adapter/parser` for the interface rather than the `port` layer, weakening the dependency-inversion the project claims.
**Fix:** Move `Parser` into `internal/port` (or document the deviation) and correct `CLAUDE.md`.

### 2.2 Composite parser only wires aviation + weather
The composite parser combines exactly two sub-parsers:
- `internal/adapter/parser/composite.go:13-40` — `CompositeParser{aviationParser, weatherParser}`.
- `pkg/di/wire.go:37-67` (`runtimeSet`) does not include the schedule parser.

The `internal/adapter/parser/schedule` package is never imported or invoked anywhere (only referenced in `CLAUDE.md`). So "flight schedule messages" parsing, described in `CLAUDE.md:132,182`, does **not** happen.

**Impact:** An entire parser package (~700 LOC incl. tests) is dead code, and a documented capability is false.
**Fix:** Wire it into the composite (with a `CanParse` guard and a `port.Parser`-compatible signature) or remove it.

### 2.3 DLQ / stream design: mixed concerns + silent loss
`ensureInfrastructure` builds a **single** stream with `WorkQueuePolicy` whose subject list includes the input subscription topic **plus** the publisher output topic **and** the DLQ subject:
- `internal/infra/nats/consumer.go:278-301` (`:289` `WorkQueuePolicy`; `:279-284` appends publisher + DLQ subjects).

The input stream therefore doubles as output + DLQ storage. More importantly, when a permanent failure occurs the consumer routes to DLQ and **unconditionally ACKs** the original, ignoring the DLQ publish result:
- `internal/infra/nats/consumer.go:220-227` — `_ = msg.Ack()` after `RouteToDLQ`.
- `internal/infra/nats/dlq_handler.go:57-68` — `RouteToDLQ` returns an error that is discarded by the caller.

**Impact:** If the DLQ publish fails (e.g., JetStream down), the poison message is still ACKed and **lost** — only a `caatsm_dlq_publish_failures_total` metric increments.
**Fix:** On DLQ publish failure, NAK/retry instead of ACK.
**Note:** This is one half of a broader ACK-after-failure pattern — the publish path has the same flaw on a different error class (see §9.4).

### 2.4 Cross-cutting: global logger vs. dependency injection
`internal/infra/log/logger.go:143` calls `zap.ReplaceGlobals(logger)` "for backward compatibility". Two packages then use the global logger instead of an injected one, violating the "no global state / propagate context" rule in `CLAUDE.md:259-263`:
- `internal/adapter/parser/schedule/schedule.go:51,115,188` → `zap.S()`.
- `internal/domain/sita.go:99` → `zap.S()`.

**Fix:** Pass `*zap.Logger` (or a context-scoped logger) into these packages and drop `zap.ReplaceGlobals`.

### 2.5 Deprecated NATS JetStream API
The code uses the legacy `nats.JetStreamContext` (`internal/infra/nats/jetstream.go:75`), consumed in `consumer.go:25` and `publisher.go:18`. `nats.go` v1.47 still supports it but it is superseded by the `jetstream` package; future upgrades may drop it.

### 2.6 Health endpoint design
`/healthz` and `/readyz` both route to `handleHealth` (`internal/infra/monitoring/server.go:52-59`), so `/healthz` is not a pure liveness check (it pings Postgres/NATS). Using it as a k8s liveness probe could cause restarts during dependency blips.
**Fix:** A correct liveness handler `/livez` (`handleLive`) already exists at `server.go:52`. Point liveness probes there, or relabel `/healthz` to match `/livez` behavior — do **not** add a new handler.

---

## 3. Package-Level Findings

### 3.1 🔴 `go.sum` is missing (build/reproducibility)
`go.mod` is committed, but **`go.sum` does not exist** on disk and is not tracked by git:
```
$ ls go.sum
ls: go.sum: No such file or directory
$ git ls-files go.sum        # (empty)
```
A clean `go build ./...` fails immediately:
```
internal/infra/config/config.go:9:2: missing go.sum entry for module providing package github.com/knadh/koanf/parsers/toml
```
**Impact:** No reproducible builds; `make build`/`make test`/CI cannot run without `go mod download`. Dependency versions/integrity are unpinned (supply-chain concern). Confirmed there is **no `vendor/` directory**, so there is no fallback.
**Fix:** `go mod tidy` and commit `go.sum`; add a CI check (`go mod verify`).

### 3.2 `schedule` package is dead
See §2.2 — never imported or invoked; only referenced by docs.

### 3.3 `port` package is missing `parser.go`
`ls internal/port` → `publisher.go`, `repository.go`, `weather_parser.go`. The `Parser` interface the docs point to does not exist here (it's in `adapter/parser`). See §2.1.

### 3.4 `mapper` package: dead interface + `FromDBRow`
- `internal/adapter/mapper/mapper.go:6-12` defines a `Mapper` interface that is **never referenced** anywhere in production code (the repository uses the concrete `*mapper.TelegramMapper` — `internal/infra/postgres/repository.go:25`).
- `FromDBRow` (`internal/adapter/mapper/telegram.go:73`) is only called from tests.

**Fix:** Remove the unused `Mapper` interface and `FromDBRow`, or actually use them.

### 3.5 `domain` package: four message types are dead (correction)
The domain layer defines **nine aviation message types**, but the aviation parser registry registers only five:
- Registry: `internal/adapter/parser/aviation/registry.go:290-296` → `ARR, DEP, CNL, DLA, FPL`.
- Constants: `internal/adapter/parser/aviation/constants.go:12-16`.

`ALN` (`domain/aln.go`), `CHG` (`domain/chg.go`), `CPL` (`domain/cpl.go`), and `SITA` (`domain/sita.go`) are referenced **only in their own `_test.go` files** — no parser produces them, so any such inbound message fails with `"invalid message type"`. `SITA` is a fundamentally different (free-text dispatch-release) format with no parser at all.

**Correction:** `domain/pln.go` does **not** define a `PLN` type — it defines `ScheduleLine` and `WayPoint` structs (`pln.go:5,42`), which are schedule helpers tied to the dead `schedule` parser, not an aviation message type. So the unparsed set is **four** types (`ALN`, `CHG`, `CPL`, `SITA`), not five, and the earlier "ten message types" count was wrong.

**Fix:** Implement + register a parser per type, or delete the unused models/tests. Add a test asserting every domain message type is reachable from the registry.

### 3.6 `metrics` package: misleading "serial reader" naming
`internal/infra/nats/monitor.go:98-99,138` records `caatsm_message_gap_seconds`, `caatsm_serial_reader_healthy`, `caatsm_message_sequence_gap_total`. The "serial reader" terminology is legacy — this is a NATS JetStream consumer, not a serial port. Functionally used, but the naming will confuse operators. (Metric names defined in `internal/infra/metrics/metrics.go:36,92,177,187,332`.)

---

## 4. Function-Level Findings

### 4.1 🔴 Idempotency race / missing unique constraint
`Repository.InsertOne` guards duplicates with a read-then-write:
- `internal/infra/postgres/repository.go:60-79` — `messageExists()` does `SELECT 1 ... WHERE message_id=$1 AND date_time=$2`.
- `internal/infra/postgres/repository.go:99` — `INSERT ... ON CONFLICT (uuid, received_at) DO NOTHING`.

But the DDL only defines `PRIMARY KEY (uuid, received_at)` (`internal/infra/postgres/telegrams.ddl:23`) — **no unique constraint on `(message_id, date_time)`**. Since `uuid` is freshly generated per parse (`aviation.go:202` / `mapper.go:31`), `ON CONFLICT (uuid, received_at)` can never fire for a duplicate business message. The only guard is the racy `SELECT`.

**Impact:** Under concurrency (multiple consumer replicas, or a redelivery that re-parses), two messages sharing `(message_id, date_time)` can both pass the `SELECT` and both insert → **duplicate rows**.
**Likelihood note (follow-up):** The consumer processes messages **sequentially** — `processBatch` loops synchronously over fetched messages (`consumer.go:163-173`), no per-message goroutine, via `PullSubscribe`+`Fetch` (`consumer.go:116,135`). So the race requires **either multiple consumer replicas** or **a `Handle()` exceeding `ackWait` (30s, `consumer.go:70`) causing NATS redelivery** (see §9.6) — not intra-process concurrency.
**Edge case:** The pre-check only runs `if msg.MessageID != "" && msg.DateTime != ""` (`repository.go:60`); messages with empty business keys won't be deduplicated even after the fix below.
**Fix:** Add a **partial** `UNIQUE (message_id, date_time) WHERE message_id <> '' AND date_time <> ''` to the DDL, switch the conflict target to it, and return a sentinel `ErrDuplicate` on conflict (do not return `nil`). Apply similarly to `telegrams_raw` — but note `telegrams_raw` (DDL `telegrams.ddl:36-44`) has **no `message_id`/`date_time` columns**, so the same unique-key fix cannot apply directly: add a `content_hash` unique index there, or accept no dedup for the raw audit table.

### 4.2 DLQ publish error ignored → ACK
See §2.3 — `RouteToDLQ` error is discarded by the caller in `consumer.go:220-227`.

### 4.3 `EffectiveSubscriptionTopic` is legacy-only
`internal/infra/config/config.go:443-451` reads **only** the legacy `subscription.topic`; there is no modern `nats.subscription_topic` field. The CLI `--subject` flag writes to the legacy block (`cmd/main/main.go:252-254`). The README's `[subscription] topic` works, but the modern config shape is inconsistent.

### 4.4 OTel recorder is mostly a no-op
`internal/infra/telemetry/telemetry.go` defines a full `Recorder` interface, but `otelRecorder` implements only `RecordProcessingResult` and `RecordPublishFailure` meaningfully. These are empty: `RecordFailure` (`:302`), `RecordMessageHandled` (`:307`), `RecordRetry` (`:312`), `RecordDLQMessage` (`:315`), `RecordDLQPublishFailure` (`:318`), `RecordJSAPICall` (`:321`), `RecordAFTNValidationError` (`:324`).

**Impact:** If you rely on OpenTelemetry (not Prometheus) for dashboards/alerts, you get almost no metrics (failures, retries, DLQ, consumer lag, AFTN errors all missing).
**Fix:** Implement the missing methods or document the Prometheus-only scope.

### 4.5 Dead validation/sanitization functions
Defined in `internal/adapter/parser/aviation/validation.go` — `ValidateBodySize` (`:53`), `ValidateTokenCount` (`:65`), `SanitizeErrorForClient` (`:114`) — are referenced **only by their own tests**. `Parse` only calls `ValidateInputSize` (`aviation.go:165`), so the 1500-char body cap and 500-token cap are never enforced (DoS hardening is partially inert).
**Fix:** Call them in the parse path, or remove them.

### 4.6 `InsertBatch` is never called in production
Defined in `internal/infra/postgres/repository.go:143` and required by the `port.Repository` interface, but the processor only calls `InsertOne`/`InsertRaw` (`internal/app/processor.go:184,247`). It also lacks the idempotency check `InsertOne` has. Benchmarks/tests exercise it; no live path does.
**Fix:** Wire it (with dedup) or drop it from the interface.

### 4.7 `DLQHandler.ValidateDLQ` is never called
`internal/infra/nats/dlq_handler.go:34` is implemented but the consumer never invokes it.

### 4.8 `SITA.Validate` builds an error awkwardly
`internal/domain/sita.go:101-106` does `err := "invalid send_time format"` (a string) then `fmt.Errorf("%s", err)`. It works, but should use `errors.New`/a wrapped error. (Also uses `zap.S()` at `:99` — see §2.4.)

---

## 5. Code-Level Findings (line-level)

- **`interface{}`/`any` in public interfaces** (violates `AGENTS.md:18` "avoid `any`"):
  - `port.Publisher.Publish(message interface{})` — `internal/port/publisher.go:6`.
  - `ParsedTelegram.BodyData interface{}` — `internal/adapter/dto/telegram.go:36`.
  - `nats.Publisher.Publish(message any)` — `internal/infra/nats/publisher.go:41` (concrete impl type-switches on `*dto.ParsedTelegram` at `:62-71`).
  - `domain.SITA.BodyData interface{}` — `internal/domain/sita.go:76` (not previously listed).
  - *Fix:* use a concrete `Publish(msg *dto.ParsedTelegram) error` or a small `Publishable` interface.

- **Redundant no-op** — `internal/infra/config/config.go:316` sets `cfg.Telemetry.Endpoint = ""` (already `""`).

- **FPL regex fragility** — `internal/adapter/parser/aviation/constants.go:113` uses a greedy `(?P<surve>.*)` and complex multiline anchors; minor format variations (extra spaces/line breaks) fail to match.

- **Unused `processor` variable** — `cmd/main/main.go:239-240` initializes `processor` then discards it with `_ = processor`; the consumer already holds its own reference.

- **CLI flag vs. validation contradiction** — `--nats-mode` help says *"jetstream or core"* (`cmd/main/main.go:71-74`), but `config.Validate()` rejects anything other than `jetstream`/`""` (`config.go:353-356`). Passing `--nats-mode core` yields a hard config error.

- **Global `zap.S()` usage** — `internal/adapter/parser/schedule/schedule.go:51,115,188` and `internal/domain/sita.go:99` (see §2.4).

---

## 6. Documentation Findings

| Doc claim | Reality | Ref |
|-----------|---------|-----|
| `Parser` interface in `internal/port/parser.go` | File does not exist; interface is in `internal/adapter/parser/parser.go` | `CLAUDE.md:187` |
| Schedule parser processes flight schedule messages | Package exists but is never wired | `CLAUDE.md:132,182` |
| Ten aviation message types parsed | Nine defined; registered: ARR/DEP/CNL/DLA/FPL; unparsed: ALN/CHG/CPL/SITA (`pln.go` = `ScheduleLine`/`WayPoint` helpers, no `PLN` type) | `registry.go:290-296`, `pln.go:5,42` |
| `config.dev.toml` exists | ✅ Confirmed present | `configs/` |
| `docs/prod-guide.md`, `nats.md`, `observability.md`, `migrations.md`, `performance.md`, `weather-parser.md` | ✅ All present | `docs/` |

`README.md` and `AGENTS.md` are otherwise accurate and useful. `CLAUDE.md` also omits several existing docs (`architecture-ha.md`, `deploy-k8s.md`, `deploy-systemd.md`, `secret-management.md`, `otel-best-practices.md`, `dev-guide.md`).

---

## 7. Prioritized Recommendations

1. **[Critical]** Commit `go.sum` (`go mod tidy`); add CI verification. *(§3.1)*
2. **[High]** Add partial `UNIQUE (message_id, date_time) WHERE non-empty`; switch `ON CONFLICT` to it; return `ErrDuplicate` on conflict (don't return `nil`). *(§4.1)*
3. **[Med-High]** Fix publish-failure state: don't persist as `parsed` before publish succeeds; avoid triple-storage across `telegrams`/`telegrams_raw`/DLQ. *(§9.1)*
4. **[Med-High]** Broaden transient publish-error classification (timeout / 503 / connection-closed / deadline), not just `ErrNoResponders`, so they NAK/retry instead of DLQ+ACK. *(§9.4)*
5. **[Medium]** Don't ACK when DLQ publish fails — NAK/retry instead. *(§2.3)*
6. **[Medium]** Wire or remove the schedule parser; register or delete ALN/CHG/CPL/SITA (`pln.go` = schedule helpers). *(§2.2, §3.5)*
7. **[Medium]** Make NATS dedup deterministic: derive `Nats-Msg-Id` from the business key (`message_id`+`date_time`+`category`), not the random per-parse `Uuid`. *(§9.3)*
8. **[Medium]** Fix duplicate handling: when `InsertOne` detects a dup, return a sentinel error (or skip publish) instead of `nil` so the processor doesn't re-publish. *(§9.2)*
9. **[Low]** Move `Parser` to `port`; fix `CLAUDE.md`. *(§2.1)*
10. **[Low]** Remove `interface{}` in `Publisher`/`BodyData` (incl. `sita.go:76`) with concrete types. *(§5)*
11. **[Low]** Implement the missing OTel recorder methods or document the Prometheus-only scope. *(§4.4)*
12. **[Low]** Remove `zap.ReplaceGlobals`; inject loggers into `schedule`/`sita`. *(§2.4)*
13. **[Low]** Delete dead code: `Mapper` interface, `FromDBRow`, `InsertBatch` (or wire it), `ValidateBodySize`/`ValidateTokenCount`/`SanitizeErrorForClient` (or call them), `DLQHandler.ValidateDLQ`, unused `processor` var, redundant no-op. *(§3.4, §4.5–4.7, §5)* — also fix `SITA.Validate` (§4.8) and harden the FPL regex (§5).
14. **[Low]** Reconcile CLI `--nats-mode` help with validation; point liveness probes at `/livez` (or relabel `/healthz`). *(§2.6, §5)*
15. **[Low-Med]** Mitigate `MatchWithTimeout` goroutine leak: bound total concurrent regex work or cap in-flight parses. *(§9.5)*
16. **[Low]** Add a per-message context timeout (`< ackWait`) in `processMsg`; bound `ProvideDB` ping with a timeout. *(§9.6, §9.8)*
17. **[Low]** Migrate from legacy `nats.JetStreamContext` to the `jetstream` package API (`jetstream.New`). *(§2.5)*
18. **[Low]** Rename "serial reader" metrics to consumer-oriented names (`caatsm_consumer_healthy`, etc.) and update Help text. *(§3.6)*
19. **[Low]** Unify subscription-topic config: read a modern `nats.subscription_topic` field with legacy `Subscription.Topic` as fallback; validate at least one is set. *(§4.3)*

---

## 8. What's Done Well

- Clean layering with Wire DI; `wire_gen.go` correctly excluded from manual edits.
- ReDoS **latency** protection via `MatchWithTimeout` on the aviation regex path (`internal/adapter/parser/aviation/aviation.go:90-101`) — *caveat:* the matching goroutine is not cancellable, so it bounds caller latency but not CPU/goroutine exhaustion under attack (see §9.5).
- Clear error taxonomy (parsed / header_error / body_error / aftn_error) and permanent-vs-transient retry strategy (`internal/app/errors.go`, `internal/infra/nats/consumer.go:211-257`).
- Sensitive-data hygiene in logs (postgres URL sanitized — `internal/infra/postgres/db.go:15-35`; DLQ payload avoids leaking credentials).
- Broad Ginkgo test coverage including benchmarks and integration tests (`test/integration`) — *caveat:* dead code (schedule, ALN/CHG/CPL/SITA, `InsertBatch`, `ValidateBodySize`) has passing tests, so coverage % is inflated and the idempotency race is not concurrency-tested.
- Thoughtful config validation and sane defaults (`internal/infra/config/config.go`).

---

## 9. Issues Missed in the Initial Review (Follow-up)

> Added after a deeper pass over the runtime/data-flow paths the first pass did not examine: `processor.go` (full `Handle` pipeline), `publisher.go`, `consumer.go` (concurrency model), `regex_timeout.go`, `db.go`, and the weather parser entry point. All claims below were verified against source.

### 9.1 [Med-High] Publish failure leaves inconsistent cross-table state
The pipeline is **insert-then-publish**:
- `internal/app/processor.go:184` — `InsertOne` succeeds → row written to `aviation.telegrams` with `status=parsed`.
- `processor.go:201` — `Publish` fails.
- `processor.go:214` — `persistRaw` writes the *same* message to `aviation.telegrams_raw` with an error status.
- `processor.go:227` — returns `Permanent` → consumer DLQ + ACK (`consumer.go:220-227`).

**Result:** one logical message lands in **three** places — `telegrams` (status `parsed`, though never dispatched), `telegrams_raw` (status error), and the DLQ. A `telegrams` row marked `parsed` that actually failed to dispatch is semantically wrong and could be re-dispatched by any downstream reader of `need_dispatch`/`dispatched_at`. The initial review's §2.3 noted DLQ loss but missed this state inconsistency.

### 9.2 [Medium] Duplicate-detected insert returns nil → processor re-publishes
`internal/infra/postgres/repository.go:72-78` returns **`nil`** (no error) when a duplicate is detected via the racy `SELECT`. Back in `processor.go:184-201`, a `nil` error is treated as success, so the processor **publishes the message again**. Thus the existing dedup check, when it *does* fire on a redelivery, causes a **duplicate publication** rather than suppressing it. This is a logic bug compounding §4.1 and was entirely missed initially.

### 9.3 [Medium] `Nats-Msg-Id` dedup header is ineffective (random UUID)
`internal/infra/nats/publisher.go:62-71` sets the JetStream deduplication header `Nats-Msg-Id` to `parsed.Uuid`. But `parsed.Uuid` is **freshly generated on every parse** (`aviation.go:202`, `composite.go:51`). On redelivery the message is re-parsed → new UUID → the dedup header is never the same → JetStream never deduplicates. The deliberate dedup feature is cosmetic. This is a separate dimension of the §4.1 idempotency problem (DB side vs. NATS-publish side).

### 9.4 [Med-High] Publish transient-error classification too narrow → data loss
`processor.go:218-223` and `publisher.go:77` treat **only** `nats.ErrNoResponders` as transient. Any other transient JetStream failure during publish (timeout, 503, slow consumer, leadership change) is classified `Permanent` → DLQ + ACK → **message permanently lost**. This is a data-loss path *parallel to and worse than* the §2.3 DLQ finding, and the initial review missed it entirely.

### 9.5 [Low-Med] `MatchWithTimeout` goroutine leak under ReDoS
`internal/adapter/parser/aviation/regex_timeout.go:42` spawns a goroutine running `re.FindStringSubmatch` with no cancellation (Go's `regexp` cannot be interrupted). On timeout the caller returns, but the goroutine keeps burning CPU until the regex finishes. The "ReDoS protection" praised in §8 bounds **latency** but not **resource exhaustion** — a burst of malicious inputs piles up leaked goroutines/CPU.

### 9.6 [Low] No per-message context timeout → redelivery race
`consumer.go:191` calls `Handle` with the loop context and no deadline. `ackWait` defaults to 30s (`consumer.go:70`). A `Handle` slower than `ackWait` triggers NATS redelivery → the same message is reprocessed concurrently (new UUID, same business key) → the §4.1 race and the §9.2 re-publish bug. (Also relevant to §4.1 likelihood.)

### 9.7 [Low] Weather parser: best-effort, lower risk than feared
Contrary to the first pass's silence, the weather path was read (`provider.go`, `classifier.go`, `metar_parser.go`). It is **best-effort with warnings** (unrecognized tokens → `Warnings`, not crashes), and `CanParse` requires a trailing `=` (`provider.go:25`) which aviation messages (ending `NNNN`) don't have — so weather-first ordering in `composite.go:30` is safe. Minor smells only: `CanParse` discards `reportType` (`provider.go:29`), and `Parse` re-classifies instead of reusing `CanParse`'s result. Not a major gap, but it was unexamined.

### 9.8 [Low] `ProvideDB` pings with `context.Background()`
`internal/infra/postgres/db.go:39,70` — no timeout on pool creation/Ping; a slow DB blocks startup indefinitely.
