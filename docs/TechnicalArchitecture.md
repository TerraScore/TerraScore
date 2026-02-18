# LANDINTEL — Technical Architecture & Implementation Blueprint

## Version 4.0 | February 2026 | CONFIDENTIAL

### Design Principle: Build the Minimum That Works, Then Iterate

> We are a startup with 5 engineers. Every piece of infrastructure we add is something we must maintain, debug at 2 AM, and pay for. This document is designed for **shipping fast**, not for impressing VCs with architecture diagrams.

---

# Table of Contents

1. [Tech Stack Decision](#1-tech-stack-decision)
2. [What We Removed (and Why)](#2-what-we-removed-and-why)
3. [System Architecture](#3-system-architecture)
4. [Data Architecture](#4-data-architecture)
5. [Implementation Phases](#5-implementation-phases)

---

# 1. Tech Stack Decision

## 1.1 Why Go (Not Python, Not Node)

The core of LandIntel is a **real-time job allocation engine** — matching agents to parcels under time pressure, with thousands of concurrent location updates. This is a concurrency problem, not a CRUD problem.

| Factor | Python | Node.js | Go | Our Pick |
|--------|--------|---------|-----|----------|
| **Concurrency** | GIL — no true parallelism | Single-threaded event loop | Goroutines — native parallel execution | **Go** |
| **Memory per process** | 80-150MB | 60-100MB | 10-20MB | **Go** (cheaper infra) |
| **Cold start** | 3-8s | 1-3s | <100ms | **Go** (faster deploys) |
| **Deployment** | venv + system deps (GDAL) | node_modules bloat | Single static binary | **Go** |
| **Type safety** | Hints only (not enforced) | TypeScript helps but `any` creep | Compiler enforced | **Go** |
| **Matching engine perf** | Sequential under GIL | Blocks event loop on CPU work | Parallel scoring across cores | **Go** |
| **WebSocket (agent tracking)** | asyncio — limited | Socket.IO — good to ~10K/proc | gorilla — 100K+/proc | **Go** |

### Where Python Stays

Python enters the codebase **only in Phase 2** when we actually need ML:

| Component | Why Python | When |
|-----------|-----------|------|
| Risk Scoring (XGBoost) | ML ecosystem is Python-only | Phase 2, Sprint 10 (not before) |
| Legal Scraping (Scrapy) | Best scraping framework | Phase 2, Sprint 9 (not before) |

**Phase 1 has zero Python.** Risk scoring in Phase 1 is rule-based, implemented in Go.

## 1.2 Final Stack

```
┌──────────────────────────────────────────────────────────────┐
│                    LANDINTEL TECH STACK                       │
│                    (Startup-Optimized)                        │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  BACKEND                    FRONTEND                         │
│  ───────                    ────────                         │
│  Go 1.22+                   Next.js 14 (App Router)          │
│  Chi router                 TailwindCSS                      │
│  sqlc (SQL → type-safe Go)  Mapbox GL JS                     │
│  pgx (PostgreSQL driver)    Zustand (state)                  │
│  gorilla/websocket          React Query (server state)       │
│  go-redis                                                    │
│                             MOBILE (Agent Only)              │
│  AUTH + API GATEWAY         ─────────────────                │
│  ───────────────            React Native (Expo)              │
│  Keycloak (identity/RBAC)   Expo Camera + Location           │
│  Kong (API gateway)         SQLite (offline)                 │
│                             Zustand + React Query            │
│  DATABASE                                                    │
│  ────────                   INFRASTRUCTURE                   │
│  PostgreSQL 16 + PostGIS    ──────────────                   │
│  Redis 7                    AWS (ap-south-1)                 │
│                             ECS Fargate (containers)         │
│  STORAGE                    ALB (load balancer)              │
│  ───────                    RDS PostgreSQL                   │
│  AWS S3                     ElastiCache Redis                │
│                             S3                               │
│  EXTERNAL SERVICES          CloudWatch (monitoring)          │
│  ─────────────────          Terraform (IaC)                  │
│  SendGrid (email)           GitHub Actions (CI/CD)           │
│  MSG91 (SMS + OTP)                                           │
│  Firebase Cloud Messaging                                    │
│  Razorpay (payments)                                         │
│  Mapbox (geocoding)                                          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

# 2. What We Removed (and Why)

This section documents everything from v3.0 that was **over-engineered** for a startup. We can add these back when the need is proven.

| Removed | Was In v3 | Why It's Overkill | Replacement | Add Back When |
|---------|-----------|-------------------|-------------|---------------|
| **8 Microservices** | Auth, Land, Job, Agent, Survey, Report, Billing, Notification as separate services | 5 engineers cannot maintain 8 services. Distributed debugging, 8 Dockerfiles, 8 CI pipelines, inter-service auth, network failures. Each service adds ~2 weeks of boilerplate. | **Modular monolith**: single Go binary, clean internal packages. Same code organization, zero network overhead. Keycloak + Kong stay as standalone infrastructure (they're battle-tested, not custom code we maintain). | When a specific module needs independent scaling (likely Job Engine at 500+ concurrent jobs). |
| **gRPC (internal)** | Proto files, codegen, gRPC servers between services | It's a monolith now — internal calls are function calls. Zero network overhead. gRPC adds proto compilation step, generated code maintenance. | **Direct function calls** between Go packages. | When we split into actual microservices. |
| **RabbitMQ** | AmazonMQ RabbitMQ ($50-130/mo) | Another managed service. At our scale (<100 events/min), it's a sledgehammer for a nail. | **Two options used together**: (1) Redis Pub/Sub for real-time events (agent location, WebSocket push), (2) PostgreSQL-backed job queue (`jobs` table with status + polling) for durable work items. Go channels for in-process events. | When event throughput exceeds what Redis Pub/Sub handles (~100K msg/sec — we won't hit this for years). |
| **Elasticsearch** | Separate ES cluster for search + logging | Another cluster to manage. We'll have <10K records in Year 1. PostgreSQL full-text search handles millions. | **PostgreSQL**: `pg_trgm` extension for fuzzy search, `tsvector` for full-text. CloudWatch for logs. | When search queries exceed PostgreSQL capabilities (100K+ documents with complex faceting). |
| **EKS (Kubernetes)** | Full EKS cluster ($73/mo control plane + node management) | Kubernetes is an operating system for infrastructure teams. A 5-person startup shouldn't be debugging pod scheduling, RBAC, ingress controllers, and helm chart templating. | **ECS Fargate**: serverless containers. Define task, set CPU/memory, deploy. No nodes to manage. ALB for routing. Auto-scales. | When we need advanced scheduling (GPU nodes, custom resource types) or have a dedicated DevOps person. |
| **Terragrunt** | Complex multi-environment Terragrunt setup | Additional abstraction layer over Terraform. Adds complexity, debugging difficulty. | **Simple Terraform** with workspaces for dev/staging/prod. One `main.tf` per environment. | When we have 10+ environments or complex module dependencies. |
| **Prometheus + Grafana** | Self-hosted monitoring stack on K8s | Two more services to deploy and maintain on K8s (which we also removed). | **AWS CloudWatch**: built-in metrics from ECS, RDS, Redis. Custom metrics via CloudWatch SDK in Go. CloudWatch Alarms for alerts. Free tier covers us. | When CloudWatch costs exceed self-hosted, or we need complex dashboards. |
| **Separate WebSocket Gateway** | Dedicated Go service for WS connections | Unnecessary at our scale. Adds deployment complexity, another service to monitor. | **Embedded in monolith**: gorilla/websocket handlers in the same Go binary. Redis Pub/Sub for multi-instance message fanout. | When WebSocket connections exceed 50K (need dedicated scaling). |
| **OpenAPI Codegen** | Proto files → Go stubs → TS client generation | Build pipeline complexity. Codegen maintenance. At 5 engineers, just write the types. | **Manual TypeScript API client** with axios. Shared type definitions in `web/lib/types.ts`. Keep API surface small and stable. | When API surface grows beyond 50+ endpoints and type drift becomes a real problem. |
| **Python in Phase 1** | Risk Engine (XGBoost) + Legal Scraping (Scrapy) | Two languages in Phase 1 = double the tooling (Docker base images, linting, CI, debugging). Risk scoring is rule-based until we have enough data for ML. Legal scraping is Phase 2. | **Go rule-based scoring**: weighted formula from survey data. Simple, debuggable, fast. | Phase 2: when we have 200+ labeled surveys for ML training, and need legal crawlers. |
| **Landowner Mobile App (Phase 1)** | React Native app for landowners alongside agent app | Two mobile apps to build and maintain simultaneously. Landowners check reports occasionally; they don't need a native app yet. | **Responsive Next.js web app** (mobile-friendly). PWA with push notifications via FCM web SDK. | Phase 2-3: when mobile app retention data justifies the investment. |
| **Stripe** | International payment gateway | No international users in Phase 1. Bangalore/Hyderabad focus. | **Razorpay only**. | When we have NRI customers wanting USD payments. |
| **WhatsApp (Gupshup)** | WhatsApp Business API integration | Additional vendor, template approvals, compliance. Email + SMS + push cover Phase 1. | **Email (SendGrid) + SMS (MSG91) + Push (FCM)**. | Phase 2: when users request it (WhatsApp has high engagement in India). |
| **Mixpanel** | Product analytics platform | Premature analytics. Focus on building, not measuring funnels. | **Simple event logging to PostgreSQL** `analytics_events` table. Query with SQL when needed. | Phase 2: when we need funnel analysis, cohort retention, A/B testing. |
| **Intercom / Freshdesk** | Customer support platform | No support volume in Phase 1. | **Email support** + in-app feedback form that sends to a shared inbox. | When support tickets exceed 20/week. |
| **BrightData** | Residential proxy network | No scraping in Phase 1. | None needed. | Phase 2: when legal scraping needs proxy rotation. |
| **CloudFront CDN** | Content delivery network | <1000 users. S3 direct + Next.js hosting is fine. | **S3 presigned URLs** for media. Vercel/Next.js for web static assets. | When media delivery latency becomes an issue (1000+ users). |
| **AWS WAF + Shield** | Web application firewall | Expensive. Basic security in Go middleware is sufficient at low traffic. | **Go middleware**: rate limiting, input validation, CORS, security headers. | When facing real DDoS or attack vectors at scale. |
| **spaCy (NER)** | Named entity recognition for legal matching | ML-based NER is overkill when we have structured data (survey numbers, names). | **Fuzzy string matching**: Levenshtein distance + exact match on survey numbers. | When unstructured legal text parsing becomes the bottleneck. |
| **Multiple DB schemas with per-service users** | `land.*`, `jobs.*`, `agents.*` etc. | It's a monolith. Single schema with clear table prefixes is fine. | **Single `public` schema**. Tables named clearly: `users`, `parcels`, `agents`, `survey_jobs`, etc. | When we split into microservices and need schema-level isolation. |

### What This Saves

| Category | v3.0 Monthly Cost | v4.0 Monthly Cost | Savings |
|----------|-------------------|-------------------|---------|
| EKS Control Plane | $73 | $0 (ECS Fargate) | $73 |
| EKS Worker Nodes (dev) | $100-160 | ~$50 (Fargate tasks) | $80 |
| AmazonMQ (RabbitMQ) | $50 | $0 (Redis Pub/Sub) | $50 |
| Elasticsearch | $80+ | $0 (PostgreSQL) | $80 |
| Keycloak | $35 (separate RDS) | ~$20 (Fargate task, shares main RDS) | $15 |
| Kong | (was on EKS) | ~$15 (Fargate task, DB-less mode) | — |
| **Total Dev Infra** | **~$485/mo** | **~$215/mo** | **~$270/mo saved** |

More importantly: **hundreds of engineering hours saved** not debugging Kubernetes, RabbitMQ, and Elasticsearch. Keycloak and Kong stay because they're battle-tested infrastructure — not custom code we wrote and must maintain.

---

# 3. System Architecture

## 3.1 The Modular Monolith

One Go binary. Clean package boundaries. Keycloak for identity. Kong for gateway. Can be split later if needed.

```
                    ┌──────────────────────────────┐
                    │          CLIENTS              │
                    │                               │
                    │  Landowner Web  Agent Mobile   │
                    │  (Next.js)     (React Native) │
                    │                               │
                    │  Admin Dashboard              │
                    │  (Next.js /admin routes)      │
                    └──────────┬────────────────────┘
                               │
                    ┌──────────▼────────────────────┐
                    │     AWS ALB (Load Balancer)    │
                    │     ─ SSL termination          │
                    │     ─ Health checks            │
                    └──────────┬────────────────────┘
                               │
                    ┌──────────▼────────────────────┐
                    │     KONG API GATEWAY           │
                    │     (Fargate, DB-less mode)    │
                    │                               │
                    │  ─ JWT validation (Keycloak)   │
                    │  ─ Rate limiting (per consumer)│
                    │  ─ Request routing → Go app    │
                    │  ─ CORS, request-size-limit    │
                    │  ─ Request ID injection        │
                    │  ─ API versioning (/v1/)       │
                    └──────┬───────────┬────────────┘
                           │           │
              ┌────────────▼──┐        │
              │  KEYCLOAK      │        │
              │  (Fargate)     │        │
              │                │        │
              │  ─ Realm:      │        │
              │    landintel   │        │
              │  ─ Clients:    │        │
              │    web-app     │        │
              │    agent-app   │        │
              │    admin-app   │        │
              │  ─ Roles:      │        │
              │    landowner   │        │
              │    agent       │        │
              │    admin       │        │
              │  ─ OTP via     │        │
              │    MSG91 SPI   │        │
              │  ─ JWT issuer  │        │
              └───────────────┘        │
                                       │
                    ┌──────────────────▼────────────┐
                    │                               │
                    │   GO MODULAR MONOLITH         │
                    │   (Single Binary on Fargate)  │
                    │                               │
                    │   ┌───────────────────────┐   │
                    │   │   Chi Router           │   │
                    │   │   ─ /v1/parcels/*      │   │
                    │   │   ─ /v1/agents/*       │   │
                    │   │   ─ /v1/jobs/*         │   │
                    │   │   ─ /v1/surveys/*      │   │
                    │   │   ─ /v1/admin/*        │   │
                    │   │   ─ /ws (WebSocket)    │   │
                    │   └───────────────────────┘   │
                    │                               │
                    │   ┌─────────┐ ┌─────────────┐ │
                    │   │  Auth   │ │    Land     │ │
                    │   │ Module  │ │   Module    │ │
                    │   │         │ │             │ │
                    │   │JWT issue│ │Parcel CRUD  │ │
                    │   │OTP/MSG91│ │Boundary     │ │
                    │   │Role RBAC│ │PostGIS      │ │
                    │   │Sessions │ │Subscription │ │
                    │   └─────────┘ └─────────────┘ │
                    │                               │
                    │   ┌─────────┐ ┌─────────────┐ │
                    │   │  Job    │ │   Agent     │ │
                    │   │ Engine  │ │   Module    │ │
                    │   │         │ │             │ │
                    │   │Scheduler│ │Profile/KYC  │ │
                    │   │Matching │ │Location     │ │
                    │   │Dispatch │ │Availability │ │
                    │   │Cascade  │ │Performance  │ │
                    │   │StateMach│ │Tiers        │ │
                    │   └─────────┘ └─────────────┘ │
                    │                               │
                    │   ┌─────────┐ ┌─────────────┐ │
                    │   │ Survey  │ │   Report    │ │
                    │   │ Module  │ │   Module    │ │
                    │   │         │ │             │ │
                    │   │Checklist│ │Auto QA      │ │
                    │   │Media    │ │PDF generate │ │
                    │   │Geofence │ │Risk scoring │ │
                    │   │GPS trace│ │(rule-based) │ │
                    │   └─────────┘ └─────────────┘ │
                    │                               │
                    │   ┌─────────┐ ┌─────────────┐ │
                    │   │Billing  │ │Notification │ │
                    │   │ Module  │ │  Module     │ │
                    │   │         │ │             │ │
                    │   │Razorpay │ │Email/SGrid  │ │
                    │   │Subscript│ │SMS/MSG91    │ │
                    │   │Payouts  │ │Push/FCM     │ │
                    │   │Wallet   │ │In-app (WS)  │ │
                    │   └─────────┘ └─────────────┘ │
                    │                               │
                    │   ┌─────────────────────────┐ │
                    │   │   Background Workers    │ │
                    │   │   (goroutines + ticker)  │ │
                    │   │                         │ │
                    │   │  ─ Job scheduler (hourly)│ │
                    │   │  ─ Offer timeout checker │ │
                    │   │  ─ Location flush to PG  │ │
                    │   │  ─ Payout batch (weekly) │ │
                    │   │  ─ Report generation     │ │
                    │   │  ─ QA pipeline           │ │
                    │   └─────────────────────────┘ │
                    │                               │
                    └───┬──────────┬──────────┬─────┘
                        │          │          │
                   ┌────▼───┐ ┌───▼────┐ ┌───▼───┐
                   │PostgreSQL│ │ Redis  │ │  S3   │
                   │+ PostGIS│ │        │ │       │
                   │         │ │Loc cache│ │Photos │
                   │All data │ │Pub/Sub │ │Videos │
                   │Full-text│ │Rate lim│ │PDFs   │
                   │search   │ │Sessions│ │Docs   │
                   └─────────┘ └────────┘ └───────┘
```

### Why Modular Monolith > Microservices (For Us, Right Now)

| Concern | Microservices (v3) | Modular Monolith (v4) |
|---------|-------------------|-----------------------|
| **Inter-module calls** | HTTP/gRPC over network (~1-5ms per call, can fail) | Direct function calls (~1μs, cannot fail from network) |
| **Transactions** | Distributed transactions (sagas, eventual consistency) | Single database transaction. `BEGIN; ... COMMIT;` Just works. |
| **Debugging** | Distributed tracing (Jaeger), correlate logs across 8 services | Single log stream. Stack traces go straight to the bug. |
| **Deployment** | 8 Docker builds, 8 CI pipelines, 8 health checks, rolling deploys for each | 1 Docker build, 1 CI pipeline, 1 deploy. Done. |
| **Local dev** | docker-compose with 8+ services + infra | `go run ./cmd/server` + PostgreSQL + Redis. That's it. |
| **Refactoring** | Change an interface = update proto, regenerate, deploy both services | Change a function signature = compiler tells you every caller to fix. |
| **Latency** | Job matching: Agent lookup (network) + skill check (network) + score (local) + save (network) | Job matching: Agent lookup (function) + skill check (function) + score (function) + save (function). Entire flow in one process. |

### Can We Split Later?

Yes. The modular monolith is **designed for future extraction**:

```
landintel/
├── internal/
│   ├── auth/          ← Can become Auth Service
│   ├── land/          ← Can become Land Service
│   ├── job/           ← Can become Job Engine (most likely first split)
│   ├── agent/         ← Can become Agent Service
│   ├── survey/        ← Can become Survey Service
│   ├── report/        ← Can become Report Service
│   ├── billing/       ← Can become Billing Service
│   └── notification/  ← Can become Notification Service
```

Each module:
- Has its own **repository interface** (database queries)
- Communicates with other modules through **Go interfaces** (not direct struct access)
- Owns its own **database tables** (can be extracted to separate DB later)
- Publishes **events via an internal event bus** (Go channels now, Redis Pub/Sub later, RabbitMQ even later)

**Split rule:** Extract a module into a separate service ONLY when it has a **different scaling profile** from the rest. Example: if the Job Engine needs 4x more CPU than everything else, split it.

## 3.2 Internal Event System

Instead of RabbitMQ, we use a simple layered approach:

```
LAYER 1: In-Process Events (Go channels)
─────────────────────────────────────────
For: Module-to-module communication within the same process.
Implementation: Simple Go event bus using channels.

  eventbus.Publish("survey.submitted", SurveySubmittedEvent{...})
  eventbus.Subscribe("survey.submitted", reportModule.HandleSurveySubmitted)

Works because: It's a monolith. All modules are in the same process.
Guarantees: Synchronous delivery. If handler fails, caller knows.

LAYER 2: Redis Pub/Sub (for real-time)
──────────────────────────────────────
For: Pushing updates to WebSocket clients (agent tracking, job status).
Implementation: Go publishes to Redis channel, WebSocket handlers subscribe.

  redis.Publish("job:123:status", statusUpdateJSON)
  // WebSocket handler subscribes and forwards to connected client

Works because: Redis is already in the stack for caching.
Guarantees: At-most-once. Messages not persisted (fine for real-time UI).

LAYER 3: PostgreSQL Job Queue (for durable work)
──────────────────────────────────────────────────
For: Work that must not be lost — report generation, payout processing,
     email delivery, QA pipeline.
Implementation: `task_queue` table with status + polling worker.

  INSERT INTO task_queue (type, payload, status) VALUES ('generate_report', '{}', 'pending');
  // Background goroutine polls: SELECT ... WHERE status = 'pending' FOR UPDATE SKIP LOCKED

Works because: PostgreSQL is ACID. If server crashes, pending tasks survive.
Guarantees: At-least-once delivery. Exactly what we need for critical work.
```

### Task Queue Table

```sql
CREATE TABLE task_queue (
    id          BIGSERIAL PRIMARY KEY,
    task_type   VARCHAR(50) NOT NULL,        -- generate_report | send_email | process_payout | run_qa
    payload     JSONB NOT NULL,              -- task-specific data
    status      VARCHAR(20) DEFAULT 'pending',
        -- pending | processing | completed | failed | dead_letter
    priority    INTEGER DEFAULT 0,           -- higher = process first
    attempts    INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    last_error  TEXT,
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),  -- for delayed tasks
    started_at  TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_task_queue_pending ON task_queue(status, priority DESC, scheduled_at)
    WHERE status = 'pending';
```

**Worker goroutine:**

```
func (w *Worker) Run(ctx context.Context):
  ticker := time.NewTicker(1 * time.Second)
  for:
    select:
      case <-ticker.C:
        task = db.QueryRow(`
          UPDATE task_queue
          SET status = 'processing', started_at = NOW(), attempts = attempts + 1
          WHERE id = (
            SELECT id FROM task_queue
            WHERE status = 'pending' AND scheduled_at <= NOW()
            ORDER BY priority DESC, created_at ASC
            LIMIT 1
            FOR UPDATE SKIP LOCKED
          )
          RETURNING *
        `)
        if task != nil:
          err = w.handle(task)
          if err:
            if task.Attempts >= task.MaxAttempts:
              markDeadLetter(task, err)
            else:
              markFailed(task, err)  // will be retried
          else:
            markCompleted(task)
```

## 3.3 Communication Patterns (Simplified)

```
CLIENT → SERVER: REST/JSON via ALB → Kong → Go chi router
  ─ Kong validates JWT (Keycloak JWKS) before request reaches Go
  ─ Kong applies rate limiting (per consumer, stored in Redis)
  ─ Kong injects X-Request-ID, strips sensitive headers
  ─ Go middleware reads validated claims, injects UserContext

MODULE → MODULE: Direct Go function calls
  ─ job.Engine calls agent.FindNearby() directly
  ─ survey.Module calls land.GetParcelBoundary() directly
  ─ No network, no serialization, no failure modes

REAL-TIME → CLIENT: WebSocket (gorilla/websocket)
  ─ Agent location → Redis Pub/Sub → WebSocket → client
  ─ Job status change → Redis Pub/Sub → WebSocket → client

BACKGROUND WORK: PostgreSQL task queue + goroutine workers
  ─ Report generation (after QA pass)
  ─ Email/SMS/Push delivery
  ─ Weekly payout batch
  ─ Scheduled survey job creation

SCHEDULED: Go ticker (in-process cron)
  ─ Job scheduler: every hour, check parcels needing surveys
  ─ Offer timeout: every minute, expire stale offers
  ─ Location flush: every 5 min, Redis → PostGIS
  ─ Payout batch: every Monday 6 AM IST
```

## 3.4 Infrastructure (ECS Fargate)

```
┌─────────────────────────────────────────────────────────────┐
│                    AWS (ap-south-1)                          │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                     VPC                               │   │
│  │                                                       │   │
│  │  Public Subnets                                       │   │
│  │  ┌─────────────────────────────────────────────────┐  │   │
│  │  │  ALB (Application Load Balancer)                │  │   │
│  │  │  ─ HTTPS listener (ACM certificate)             │  │   │
│  │  │  ─ /api/* → Go Fargate tasks                    │  │   │
│  │  │  ─ /ws   → Go Fargate tasks (sticky sessions)   │  │   │
│  │  └─────────────────────────────────────────────────┘  │   │
│  │                                                       │   │
│  │  Private Subnets                                      │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │   │
│  │  │ ECS: Kong    │  │ ECS: Go App  │  │ECS:Keycloak│  │   │
│  │  │ (API GW)     │  │ (monolith)   │  │(Identity)  │  │   │
│  │  │ 0.25vCPU     │  │ 0.5 vCPU     │  │0.5 vCPU    │  │   │
│  │  │ 0.5GB RAM    │  │ 1GB RAM      │  │1GB RAM     │  │   │
│  │  │ DB-less mode │  │ auto-scaled  │  │shared RDS  │  │   │
│  │  └──────────────┘  └──────────────┘  └────────────┘  │   │
│  │                                                       │   │
│  │  Database Subnets                                     │   │
│  │  ┌──────────────────┐  ┌───────────────────────┐     │   │
│  │  │ RDS PostgreSQL   │  │ ElastiCache Redis     │     │   │
│  │  │ 16 + PostGIS     │  │ 7                     │     │   │
│  │  │ db.t4g.medium    │  │ cache.t4g.micro       │     │   │
│  │  │ (dev)            │  │ (dev)                 │     │   │
│  │  └──────────────────┘  └───────────────────────┘     │   │
│  │                                                       │   │
│  └───────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌──────────────────┐  ┌──────────────────────────────┐     │
│  │ S3               │  │ ECR (Container Registry)     │     │
│  │ survey-media     │  │ landintel-api                │     │
│  │ reports          │  └──────────────────────────────┘     │
│  └──────────────────┘                                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘

Dev Cost:    ~$136/mo (RDS $15 + Redis $13 + Fargate $50 + ALB $20 + NAT $35 + S3 $2 + ECR $1)
Staging:     ~$136/mo (same as dev)
Production:  ~$600-800/mo (larger RDS, more Fargate tasks, multi-AZ Redis, 2x Keycloak)
```

### Why ECS Fargate Over EKS

| Factor | EKS | ECS Fargate |
|--------|-----|-------------|
| Control plane cost | $73/mo (always-on) | $0 (included) |
| Node management | You manage EC2 instances, AMI updates, scaling | Zero — AWS manages everything |
| Deployment | Helm charts, kubectl, rolling updates | Task definition update → deploy. Done. |
| Learning curve | Weeks to master K8s concepts | Hours to learn ECS concepts |
| Debugging | kubectl logs, exec into pods, check events | CloudWatch logs, ECS console |
| When to upgrade | Never (for us at current scale) | When we need K8s-specific features (custom operators, service mesh) |

## 3.5 Project Structure

```
landintel/
├── cmd/
│   └── server/
│       └── main.go                 # Entrypoint: wire modules, start HTTP server
│
├── internal/                       # All business logic (not importable externally)
│   ├── auth/
│   │   ├── handler.go              # HTTP handlers: proxy to Keycloak token endpoint
│   │   ├── service.go              # Business logic: Keycloak admin API calls
│   │   ├── repository.go           # Database queries (users table — profile data)
│   │   ├── keycloak.go             # Keycloak Admin REST API client (create user, assign roles, trigger OTP)
│   │   ├── middleware.go           # Auth middleware: validate Keycloak JWT (public key), extract claims, inject UserContext
│   │   └── otp.go                  # MSG91 OTP via Keycloak custom SPI / authenticator
│   │
│   ├── land/
│   │   ├── handler.go              # HTTP: parcel CRUD, boundary, subscriptions
│   │   ├── service.go              # Validation, PostGIS operations, subscription logic
│   │   ├── repository.go           # sqlc-generated queries
│   │   └── events.go               # Publish: parcel.registered
│   │
│   ├── job/
│   │   ├── handler.go              # HTTP: job status, admin manual assignment
│   │   ├── engine.go               # Matching algorithm, scoring, candidate ranking
│   │   ├── dispatcher.go           # Offer dispatch, cascade logic, timeout handling
│   │   ├── scheduler.go            # Cron: check parcels needing surveys
│   │   ├── statemachine.go         # Job status transitions (enforced)
│   │   └── repository.go           # sqlc queries for survey_jobs, job_offers
│   │
│   ├── agent/
│   │   ├── handler.go              # HTTP: register, profile, location, availability, earnings
│   │   ├── service.go              # KYC, tier promotion, performance calculation
│   │   ├── location.go             # Location tracking: Redis write + PostGIS flush
│   │   ├── repository.go           # sqlc queries for agents
│   │   └── matching.go             # FindNearbyAgents (PostGIS spatial query)
│   │
│   ├── survey/
│   │   ├── handler.go              # HTTP: submit survey, upload media, get checklist
│   │   ├── service.go              # Geofence verification, completeness check
│   │   ├── media.go                # S3 presigned URL generation, metadata recording
│   │   ├── checklist.go            # Template management, step validation
│   │   └── repository.go           # sqlc queries for responses, media
│   │
│   ├── report/
│   │   ├── handler.go              # HTTP: get reports, download PDF
│   │   ├── qa.go                   # Automated QA pipeline (all checks)
│   │   ├── generator.go            # HTML template → PDF (chromedp)
│   │   ├── risk.go                 # Rule-based risk scoring (Phase 1)
│   │   └── repository.go           # sqlc queries for risk_scores
│   │
│   ├── billing/
│   │   ├── handler.go              # HTTP: subscription management, payment webhooks
│   │   ├── razorpay.go             # Razorpay subscription + payout integration
│   │   ├── wallet.go               # Agent wallet operations
│   │   ├── payout.go               # Weekly batch payout logic
│   │   └── repository.go           # sqlc queries for transactions, payouts
│   │
│   ├── notification/
│   │   ├── service.go              # Routing engine: which channels for which event
│   │   ├── email.go                # SendGrid integration
│   │   ├── sms.go                  # MSG91 integration
│   │   ├── push.go                 # Firebase Cloud Messaging
│   │   └── templates.go            # Notification templates
│   │
│   └── platform/                   # Cross-cutting concerns
│       ├── config.go               # Env-based config (viper)
│       ├── database.go             # pgx pool setup
│       ├── redis.go                # go-redis client
│       ├── s3.go                   # S3 client + presigned URLs
│       ├── eventbus.go             # In-process event bus (Go channels)
│       ├── taskqueue.go            # PostgreSQL-backed task queue
│       ├── websocket.go            # WebSocket hub + rooms
│       ├── middleware.go           # Shared: request ID, logging, recovery
│       ├── errors.go               # Standard error types
│       └── geo.go                  # GeoJSON helpers
│
├── db/
│   ├── migrations/                 # golang-migrate SQL files
│   │   ├── 001_create_users.up.sql
│   │   ├── 001_create_users.down.sql
│   │   ├── 002_create_parcels.up.sql
│   │   └── ...
│   ├── queries/                    # sqlc query files (SQL → Go codegen)
│   │   ├── users.sql
│   │   ├── parcels.sql
│   │   ├── agents.sql
│   │   ├── jobs.sql
│   │   ├── surveys.sql
│   │   └── billing.sql
│   └── sqlc.yaml                   # sqlc configuration
│
├── web/                            # Next.js (landowner dashboard + admin)
│   ├── app/
│   ├── components/
│   └── package.json
│
├── mobile/                         # React Native (agent app only in Phase 1)
│   ├── app/                        # Expo Router
│   ├── components/
│   └── package.json
│
├── infra/                          # Terraform + service configs
│   ├── main.tf                     # All AWS resources
│   ├── variables.tf
│   ├── outputs.tf
│   ├── terraform.tfvars.dev
│   ├── terraform.tfvars.staging
│   ├── terraform.tfvars.prod
│   ├── keycloak/
│   │   └── landintel-realm.json    # Realm export: clients, roles, mappers, OTP flow
│   └── kong/
│       └── kong.yml                # Declarative config: routes, services, plugins
│
├── Dockerfile                      # Single Dockerfile for Go monolith
├── docker-compose.yml              # Local dev: PostgreSQL + Redis
├── Makefile                        # build, test, migrate, run, deploy
├── .github/workflows/
│   └── deploy.yml                  # CI/CD: test → build → push → deploy
└── go.mod
```

## 3.6 API Design (All Routes, One Router)

```go
// cmd/server/main.go — simplified

func main() {
    cfg := config.Load()
    db := database.Connect(cfg.DatabaseURL)
    rdb := redis.Connect(cfg.RedisURL)
    s3 := s3.NewClient(cfg)

    // Initialize modules
    authMod := auth.New(db, rdb, cfg)
    landMod := land.New(db, cfg)
    agentMod := agent.New(db, rdb, cfg)
    jobMod := job.New(db, rdb, agentMod, notifMod, cfg)
    surveyMod := survey.New(db, s3, cfg)
    reportMod := report.New(db, s3, cfg)
    billingMod := billing.New(db, cfg)
    notifMod := notification.New(db, cfg)
    wsMod := websocket.New(rdb)

    // Router
    r := chi.NewRouter()

    // Global middleware (Kong handles JWT validation + rate limiting + CORS)
    // Go middleware handles: request ID propagation, logging, panic recovery,
    // and extracting Keycloak claims from validated JWT
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(authMod.ExtractKeycloakClaims) // reads JWT claims (already validated by Kong)

    // Auth routes (Go orchestrates OTP + Keycloak Admin API)
    r.Post("/v1/auth/register", authMod.Register)        // create user in Keycloak + send OTP
    r.Post("/v1/auth/verify-otp", authMod.VerifyOTP)      // verify OTP → get Keycloak JWT
    r.Post("/v1/auth/login", authMod.Login)                // send OTP for existing user
    r.Post("/v1/auth/refresh", authMod.RefreshToken)       // proxy to Keycloak token endpoint
    r.Post("/v1/agents/register", agentMod.Register)       // create agent in Keycloak + DB
    r.Post("/v1/billing/webhooks/razorpay", billingMod.RazorpayWebhook)

    // Authenticated routes (landowner)
    r.Route("/v1/parcels", func(r chi.Router) {
        r.Use(authMod.RequireAuth)
        r.Use(authMod.RequireRole("landowner"))
        r.Post("/", landMod.CreateParcel)
        r.Get("/", landMod.ListParcels)
        r.Get("/{id}", landMod.GetParcel)
        r.Put("/{id}/boundary", landMod.UpdateBoundary)
        r.Post("/{id}/request-visit", landMod.RequestOnDemandVisit)
        r.Get("/{id}/surveys", surveyMod.ListSurveys)
        r.Get("/{id}/risk-score", reportMod.GetRiskScore)
        r.Get("/{id}/reports", reportMod.ListReports)
        r.Get("/{id}/reports/{reportId}/download", reportMod.DownloadReport)
    })

    // Authenticated routes (agent)
    r.Route("/v1/agents/me", func(r chi.Router) {
        r.Use(authMod.RequireAuth)
        r.Use(authMod.RequireRole("agent"))
        r.Get("/", agentMod.GetProfile)
        r.Put("/profile", agentMod.UpdateProfile)
        r.Post("/kyc", agentMod.SubmitKYC)
        r.Post("/location", agentMod.UpdateLocation)
        r.Put("/availability", agentMod.ToggleAvailability)
        r.Get("/jobs", jobMod.ListAgentJobs)
        r.Post("/jobs/{id}/accept", jobMod.AcceptOffer)
        r.Post("/jobs/{id}/decline", jobMod.DeclineOffer)
        r.Post("/jobs/{id}/arrive", jobMod.MarkArrival)
        r.Post("/jobs/{id}/start", surveyMod.StartSurvey)
        r.Post("/jobs/{id}/submit", surveyMod.SubmitSurvey)
        r.Post("/jobs/{id}/media", surveyMod.UploadMedia)
        r.Get("/earnings", billingMod.GetAgentEarnings)
    })

    // Admin routes
    r.Route("/v1/admin", func(r chi.Router) {
        r.Use(authMod.RequireAuth)
        r.Use(authMod.RequireRole("admin"))
        r.Get("/jobs/unassigned", jobMod.ListUnassignedJobs)
        r.Post("/jobs/{id}/assign", jobMod.ManualAssign)
        r.Get("/qa/pending", reportMod.ListPendingQA)
        r.Post("/qa/{id}/review", reportMod.SubmitQAReview)
        r.Get("/agents", agentMod.ListAgents)
        r.Put("/agents/{id}/status", agentMod.UpdateAgentStatus)
    })

    // Landowner alerts
    r.Route("/v1/alerts", func(r chi.Router) {
        r.Use(authMod.RequireAuth)
        r.Get("/", notifMod.ListAlerts)
        r.Put("/{id}/read", notifMod.MarkRead)
    })

    // WebSocket
    r.Get("/ws", wsMod.HandleUpgrade)

    // Subscription management
    r.Route("/v1/subscriptions", func(r chi.Router) {
        r.Use(authMod.RequireAuth)
        r.Post("/", billingMod.CreateSubscription)
        r.Get("/", billingMod.GetSubscription)
        r.Put("/{id}/plan", billingMod.ChangePlan)
        r.Delete("/{id}", billingMod.CancelSubscription)
    })

    // Start background workers
    go jobMod.StartScheduler(ctx)       // hourly job creation
    go jobMod.StartOfferTimeoutChecker(ctx) // minute-by-minute offer expiry
    go agentMod.StartLocationFlusher(ctx)   // 5-min Redis → PostGIS
    go taskqueue.StartWorkers(ctx, 4)       // 4 concurrent task workers

    // Serve
    http.ListenAndServe(":8080", r)
}
```

---

# 4. Data Architecture

## 4.1 Complete Database Schema

Single `public` schema. Clean table names. No over-normalization.

### Users + Parcels + Subscriptions

```sql
-- ═══════════════════════════════════════
-- USERS (Landowners)
-- ═══════════════════════════════════════

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone           VARCHAR(15) NOT NULL UNIQUE,
    email           VARCHAR(255),
    full_name       VARCHAR(200) NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'landowner',
        -- landowner | admin | ops
    avatar_url      TEXT,

    state_code      VARCHAR(10),
    district_code   VARCHAR(10),
    city            VARCHAR(100),

    status          VARCHAR(20) DEFAULT 'active',
    phone_verified  BOOLEAN DEFAULT TRUE,
    language        VARCHAR(5) DEFAULT 'en',

    notification_prefs JSONB DEFAULT '{"email": true, "sms": true, "push": true}',

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_phone ON users(phone);


-- ═══════════════════════════════════════
-- PARCELS
-- ═══════════════════════════════════════

CREATE TABLE parcels (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    label           VARCHAR(200),
    survey_number   VARCHAR(50),
    village         VARCHAR(100),
    taluk           VARCHAR(100),
    district        VARCHAR(100) NOT NULL,
    state           VARCHAR(100) NOT NULL,
    state_code      VARCHAR(10) NOT NULL,
    pin_code        VARCHAR(6),

    boundary        GEOMETRY(POLYGON, 4326) NOT NULL,
    centroid        GEOMETRY(POINT, 4326) GENERATED ALWAYS AS (ST_Centroid(boundary)) STORED,
    area_sqm        REAL GENERATED ALWAYS AS (ST_Area(boundary::geography)) STORED,

    land_type       VARCHAR(30),
    registered_area_sqm REAL,
    title_deed_s3_key TEXT,

    status          VARCHAR(20) DEFAULT 'active',
    monitoring_since TIMESTAMPTZ DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_parcels_user ON parcels(user_id);
CREATE INDEX idx_parcels_boundary ON parcels USING GIST(boundary);
CREATE INDEX idx_parcels_centroid ON parcels USING GIST(centroid);


-- ═══════════════════════════════════════
-- SUBSCRIPTIONS
-- ═══════════════════════════════════════

CREATE TABLE subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    parcel_id       UUID NOT NULL REFERENCES parcels(id),

    plan            VARCHAR(20) NOT NULL,        -- basic | pro | premium
    status          VARCHAR(20) DEFAULT 'active',
    amount_per_cycle NUMERIC(10,2) NOT NULL,

    razorpay_subscription_id VARCHAR(100),
    current_period_start TIMESTAMPTZ,
    current_period_end   TIMESTAMPTZ,

    visits_used_this_period INTEGER DEFAULT 0,
    on_demand_visits_remaining INTEGER DEFAULT 0,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subs_active ON subscriptions(status) WHERE status = 'active';
```

### Agents

```sql
-- ═══════════════════════════════════════
-- AGENTS
-- ═══════════════════════════════════════

CREATE TABLE agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    full_name       VARCHAR(200) NOT NULL,
    phone           VARCHAR(15) NOT NULL UNIQUE,
    email           VARCHAR(255),
    date_of_birth   DATE,

    aadhaar_hash    VARCHAR(64),
    aadhaar_verified BOOLEAN DEFAULT FALSE,

    home_location   GEOMETRY(POINT, 4326),
    last_known_location GEOMETRY(POINT, 4326),
    last_location_at TIMESTAMPTZ,
    preferred_radius_km INTEGER DEFAULT 25,
    state_code      VARCHAR(10),
    district_code   VARCHAR(10),

    status          VARCHAR(25) DEFAULT 'pending_verification',
        -- pending_verification | training | active | suspended | deactivated
    tier            VARCHAR(20) DEFAULT 'basic',
        -- basic | experienced (25+) | senior (100+) | expert (250+)
    vehicle_type    VARCHAR(20),

    total_jobs_completed INTEGER DEFAULT 0,
    avg_rating      NUMERIC(3,2) DEFAULT 0.00,
    completion_rate NUMERIC(5,4) DEFAULT 1.0000,
    qa_pass_rate    NUMERIC(5,4) DEFAULT 1.0000,
    last_job_completed_at TIMESTAMPTZ,

    bank_account_enc TEXT,
    bank_ifsc       VARCHAR(11),
    upi_id          VARCHAR(100),
    wallet_balance  NUMERIC(10,2) DEFAULT 0.00,

    certifications  TEXT[] DEFAULT ARRAY['basic_survey'],

    fcm_token       TEXT,
    device_id       VARCHAR(100),
    app_version     VARCHAR(20),

    is_online       BOOLEAN DEFAULT FALSE,
    available_days  TEXT[] DEFAULT ARRAY['mon','tue','wed','thu','fri','sat'],
    available_start TIME DEFAULT '08:00',
    available_end   TIME DEFAULT '18:00',

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agents_location ON agents USING GIST(last_known_location);
CREATE INDEX idx_agents_matching ON agents(status, is_online, tier)
    WHERE status = 'active' AND is_online = TRUE;
```

### Jobs + Offers

```sql
-- ═══════════════════════════════════════
-- SURVEY JOBS
-- ═══════════════════════════════════════

CREATE TABLE survey_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id       UUID NOT NULL REFERENCES parcels(id),
    subscription_id UUID REFERENCES subscriptions(id),
    user_id         UUID NOT NULL,

    survey_type     VARCHAR(30) NOT NULL,
    priority        VARCHAR(10) DEFAULT 'normal',
    deadline        TIMESTAMPTZ NOT NULL,
    trigger         VARCHAR(20) DEFAULT 'scheduled',

    status          VARCHAR(25) DEFAULT 'pending_assignment',
        -- pending_assignment | offered | assigned | agent_en_route
        -- | agent_on_site | in_progress | submitted
        -- | completed | failed_qa | cancelled | unassigned

    assigned_agent_id UUID REFERENCES agents(id),
    assigned_at     TIMESTAMPTZ,
    cascade_round   INTEGER DEFAULT 0,
    total_offers_sent INTEGER DEFAULT 0,

    agent_arrived_at TIMESTAMPTZ,
    survey_started_at TIMESTAMPTZ,
    survey_submitted_at TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,

    arrival_location GEOMETRY(POINT, 4326),
    arrival_distance_m REAL,

    base_payout     NUMERIC(8,2),
    distance_bonus  NUMERIC(8,2) DEFAULT 0,
    urgency_bonus   NUMERIC(8,2) DEFAULT 0,
    total_payout    NUMERIC(8,2),
    payout_status   VARCHAR(20) DEFAULT 'pending',

    landowner_rating NUMERIC(2,1),
    qa_score        NUMERIC(5,4),
    qa_status       VARCHAR(20) DEFAULT 'pending',
    qa_notes        TEXT,

    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_jobs_status ON survey_jobs(status);
CREATE INDEX idx_jobs_parcel ON survey_jobs(parcel_id, created_at DESC);
CREATE INDEX idx_jobs_agent ON survey_jobs(assigned_agent_id, status);
CREATE INDEX idx_jobs_pending ON survey_jobs(deadline) WHERE status IN ('pending_assignment', 'offered');


-- ═══════════════════════════════════════
-- JOB OFFERS
-- ═══════════════════════════════════════

CREATE TABLE job_offers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES survey_jobs(id),
    agent_id        UUID NOT NULL REFERENCES agents(id),
    cascade_round   INTEGER NOT NULL,
    offer_rank      INTEGER NOT NULL,

    distance_km     REAL,
    match_score     NUMERIC(5,4),

    status          VARCHAR(20) DEFAULT 'sent',
        -- sent | accepted | declined | expired
    sent_at         TIMESTAMPTZ DEFAULT NOW(),
    responded_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,
    decline_reason  VARCHAR(100)
);

CREATE INDEX idx_offers_job ON job_offers(job_id);
CREATE INDEX idx_offers_agent ON job_offers(agent_id, status);
```

### Surveys + Media

```sql
-- ═══════════════════════════════════════
-- CHECKLIST TEMPLATES
-- ═══════════════════════════════════════

CREATE TABLE checklist_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    survey_type VARCHAR(30) NOT NULL,
    version     INTEGER DEFAULT 1,
    is_active   BOOLEAN DEFAULT TRUE,
    steps       JSONB NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);


-- ═══════════════════════════════════════
-- SURVEY RESPONSES
-- ═══════════════════════════════════════

CREATE TABLE survey_responses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id      UUID NOT NULL REFERENCES survey_jobs(id),
    agent_id    UUID NOT NULL REFERENCES agents(id),
    template_id UUID REFERENCES checklist_templates(id),

    responses   JSONB NOT NULL,
    gps_trail   GEOMETRY(LINESTRING, 4326),
    device_info JSONB,

    started_at  TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ DEFAULT NOW(),
    duration_minutes REAL
);

CREATE INDEX idx_responses_job ON survey_responses(job_id);


-- ═══════════════════════════════════════
-- SURVEY MEDIA
-- ═══════════════════════════════════════

CREATE TABLE survey_media (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES survey_jobs(id),
    agent_id        UUID NOT NULL REFERENCES agents(id),
    step_id         VARCHAR(100) NOT NULL,

    media_type      VARCHAR(10) NOT NULL,
    s3_key          TEXT NOT NULL,
    file_size_bytes BIGINT,
    duration_sec    INTEGER,

    location        GEOMETRY(POINT, 4326) NOT NULL,
    captured_at     TIMESTAMPTZ NOT NULL,

    file_hash_sha256 VARCHAR(64) NOT NULL,
    device_id       VARCHAR(100),

    within_boundary BOOLEAN,
    duplicate_hash  VARCHAR(64),

    uploaded_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_media_job ON survey_media(job_id);
```

### Billing + Risk + Tasks

```sql
-- ═══════════════════════════════════════
-- TRANSACTIONS (landowner payments)
-- ═══════════════════════════════════════

CREATE TABLE transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    subscription_id UUID,
    type            VARCHAR(20) NOT NULL,
    amount          NUMERIC(10,2) NOT NULL,
    status          VARCHAR(20) DEFAULT 'pending',
    razorpay_payment_id VARCHAR(100),
    razorpay_order_id   VARCHAR(100),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);


-- ═══════════════════════════════════════
-- AGENT PAYOUTS
-- ═══════════════════════════════════════

CREATE TABLE agent_payouts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        UUID NOT NULL REFERENCES agents(id),
    period_start    DATE NOT NULL,
    period_end      DATE NOT NULL,
    total_jobs      INTEGER,
    gross_amount    NUMERIC(10,2),
    platform_commission NUMERIC(10,2),
    tds_deducted    NUMERIC(10,2),
    net_amount      NUMERIC(10,2),
    status          VARCHAR(20) DEFAULT 'pending',
    razorpay_payout_id VARCHAR(100),
    failure_reason  TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);


-- ═══════════════════════════════════════
-- RISK SCORES
-- ═══════════════════════════════════════

CREATE TABLE risk_scores (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id       UUID NOT NULL REFERENCES parcels(id),
    job_id          UUID,

    overall_score   NUMERIC(5,2) NOT NULL,
    risk_level      VARCHAR(10) NOT NULL,

    encroachment_score  NUMERIC(5,2),
    boundary_score      NUMERIC(5,2),
    environmental_score NUMERIC(5,2),
    neighborhood_score  NUMERIC(5,2),

    contributing_factors JSONB,
    computed_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_risk_parcel ON risk_scores(parcel_id, computed_at DESC);


-- ═══════════════════════════════════════
-- TASK QUEUE (replaces RabbitMQ)
-- ═══════════════════════════════════════

CREATE TABLE task_queue (
    id              BIGSERIAL PRIMARY KEY,
    task_type       VARCHAR(50) NOT NULL,
    payload         JSONB NOT NULL,
    status          VARCHAR(20) DEFAULT 'pending',
    priority        INTEGER DEFAULT 0,
    attempts        INTEGER DEFAULT 0,
    max_attempts    INTEGER DEFAULT 3,
    last_error      TEXT,
    scheduled_at    TIMESTAMPTZ DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_task_queue_pending ON task_queue(status, priority DESC, scheduled_at)
    WHERE status = 'pending';


-- ═══════════════════════════════════════
-- ALERTS (in-app notifications)
-- ═══════════════════════════════════════

CREATE TABLE alerts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    type        VARCHAR(50) NOT NULL,
    title       VARCHAR(200) NOT NULL,
    body        TEXT,
    data        JSONB,
    is_read     BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_alerts_user ON alerts(user_id, is_read, created_at DESC);


-- ═══════════════════════════════════════
-- ANALYTICS EVENTS (replaces Mixpanel)
-- ═══════════════════════════════════════

CREATE TABLE analytics_events (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID,
    agent_id    UUID,
    event_type  VARCHAR(100) NOT NULL,
    properties  JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_analytics_type ON analytics_events(event_type, created_at DESC);
```

## 4.2 Event Catalog (Simplified)

Internal events (Go channels) + Redis Pub/Sub (real-time to clients):

| Event | Producer | Consumers | Delivery |
|-------|----------|-----------|----------|
| `parcel.registered` | Land module | Job module (schedule first survey) | In-process (Go channel) |
| `subscription.activated` | Billing module | Job module (start scheduling) | In-process |
| `job.created` | Job scheduler | Job dispatcher (start matching) | In-process |
| `job.status_changed` | Job module | WebSocket hub (push to clients), Notification module | In-process + Redis Pub/Sub |
| `survey.submitted` | Survey module | Report module (trigger QA) | In-process → task queue |
| `survey.qa_passed` | Report module | Billing module (credit wallet), Notification module (send report) | In-process → task queue |
| `survey.qa_failed` | Report module | Job module (re-schedule), Notification module | In-process → task queue |
| `report.generated` | Report module | Notification module (email/SMS/push to landowner) | Task queue |
| `risk.scored` | Report module | Notification module (alert if high risk) | In-process |
| `agent.location` | Agent module | Redis Pub/Sub → WebSocket → landowner tracking | Redis Pub/Sub (real-time) |
| `payout.completed` | Billing module | Notification module (confirm to agent) | Task queue |

---

# 5. Implementation Phases

---

## PHASE 1: Core Loop (Weeks 1-12)

**The only goal: Landowner registers parcel → agent surveys it → landowner gets report.**

Everything else is Phase 2.

---

### Sprint 1 (Weeks 1-2): Infrastructure + Project Setup + Auth

**Goal:** Dev environment running. Go project structured. Auth working.

#### Infrastructure (Terraform)

```
infra/
├── main.tf              # VPC, subnets, ECS cluster, ALB, RDS, Redis, S3, ECR
├── variables.tf
├── outputs.tf
└── terraform.tfvars.dev
```

| Resource | Spec (Dev) | Spec (Prod - future) | Monthly Cost (Dev) |
|----------|-----------|---------------------|-------------------|
| VPC | 2 AZs, public + private subnets, NAT Gateway | 3 AZs | $35 (NAT) |
| ECS Cluster | Fargate, no EC2 | Same | $0 (cluster free) |
| ECS Task: Go Monolith | 1 task, 0.5 vCPU, 1GB RAM | 2-4 tasks, 1 vCPU, 2GB | ~$20 |
| ECS Task: Keycloak | 1 task, 0.5 vCPU, 1GB RAM | 2 tasks (HA), 1 vCPU, 2GB | ~$20 |
| ECS Task: Kong | 1 task, 0.25 vCPU, 0.5GB RAM | 2 tasks (HA), 0.5 vCPU, 1GB | ~$10 |
| ALB | Single ALB, HTTP + HTTPS | Same | $20 |
| RDS PostgreSQL 16 | `db.t4g.micro`, 20GB, PostGIS + Keycloak schema | `db.t4g.medium` multi-AZ | $15 |
| ElastiCache Redis | `cache.t4g.micro`, single node | `cache.t4g.small`, 2 nodes | $13 |
| S3 | 1 bucket with prefixes | Same | ~$2 |
| ECR | 3 repos: `landintel-api`, `keycloak-custom`, `kong-custom` | Same | ~$1 |
| **Total** | | | **~$136/mo** |

#### CI/CD (GitHub Actions)

```yaml
# .github/workflows/deploy.yml
name: Build and Deploy

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgis/postgis:16-3.4
        env:
          POSTGRES_DB: landintel_test
          POSTGRES_PASSWORD: test
        ports: ["5432:5432"]
      redis:
        image: redis:7-alpine
        ports: ["6379:6379"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go vet ./...
      - run: go test ./... -race -cover -coverprofile=coverage.out
      - run: go build -o /dev/null ./cmd/server

  deploy:
    if: github.ref == 'refs/heads/main'
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: ap-south-1
      - uses: aws-actions/amazon-ecr-login@v2
      - name: Build and push
        run: |
          docker build -t landintel-api .
          docker tag landintel-api $ECR_REGISTRY/landintel-api:$GITHUB_SHA
          docker push $ECR_REGISTRY/landintel-api:$GITHUB_SHA
      - name: Deploy to ECS
        run: |
          aws ecs update-service \
            --cluster landintel-dev \
            --service landintel-api \
            --force-new-deployment
```

#### Dockerfile (Single, Multi-stage)

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/server /server
COPY db/migrations /migrations
EXPOSE 8080
CMD ["/server"]
```

#### Local Dev

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgis/postgis:16-3.4
    ports: ["5432:5432"]
    environment:
      POSTGRES_DB: landintel
      POSTGRES_PASSWORD: dev
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  keycloak:
    image: quay.io/keycloak/keycloak:24.0
    ports: ["8180:8080"]
    environment:
      KC_DB: postgres
      KC_DB_URL: jdbc:postgresql://postgres:5432/landintel?currentSchema=keycloak
      KC_DB_USERNAME: postgres
      KC_DB_PASSWORD: dev
      KEYCLOAK_ADMIN: admin
      KEYCLOAK_ADMIN_PASSWORD: admin
    command: start-dev --import-realm
    volumes:
      - ./infra/keycloak/landintel-realm.json:/opt/keycloak/data/import/landintel-realm.json
    depends_on:
      - postgres

  kong:
    image: kong:3.6
    ports:
      - "8000:8000"    # Proxy (clients hit this)
      - "8001:8001"    # Admin API (local only)
    environment:
      KONG_DATABASE: "off"
      KONG_DECLARATIVE_CONFIG: /etc/kong/kong.yml
      KONG_PROXY_LISTEN: "0.0.0.0:8000"
      KONG_ADMIN_LISTEN: "0.0.0.0:8001"
      KONG_LOG_LEVEL: info
    volumes:
      - ./infra/kong/kong.yml:/etc/kong/kong.yml
    depends_on:
      - keycloak

volumes:
  pgdata:
```

```makefile
# Makefile
.PHONY: dev test migrate build

dev:
	docker compose up -d
	@echo "Waiting for Keycloak to start..."
	@sleep 10
	go run ./cmd/server

dev-infra:
	docker compose up -d    # Start PG + Redis + Keycloak + Kong only

test:
	go test ./... -race -cover

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down 1

build:
	docker build -t landintel-api .

sqlc:
	sqlc generate

kong-reload:
	curl -s http://localhost:8001/config -X POST -F config=@infra/kong/kong.yml
```

#### Keycloak Setup

| Task | Details |
|------|---------|
| **Deployment** | Keycloak 24+ on ECS Fargate (1 task, 0.5 vCPU, 1GB). Uses main RDS PostgreSQL (separate `keycloak` schema). Accessible at `auth.landintel.in`. |
| **Realm** | `landintel` realm. Token lifespan: access = 15 min, refresh = 30 days. |
| **Clients** | `web-app` (public, PKCE), `agent-app` (public, PKCE), `admin-app` (public, PKCE), `go-backend` (confidential, service account for Admin API calls). |
| **Roles** | Realm roles: `landowner`, `agent`, `admin`, `ops_team`. Mapped to JWT claims via protocol mapper. |
| **Custom Attributes** | `subscription_plan`, `agent_tier` — added as JWT claim mappers so Go middleware can read them from token without DB call. |
| **OTP Authentication** | Custom Keycloak Authenticator SPI that sends OTP via MSG91 API instead of email. Phone number is the username. Flow: phone → MSG91 OTP → verify → Keycloak issues JWT. |
| **Local Dev** | Keycloak runs in docker-compose alongside PostgreSQL + Redis. Pre-configured realm exported as JSON, auto-imported on startup. |

#### Kong API Gateway Setup

| Task | Details |
|------|---------|
| **Deployment** | Kong 3.x on ECS Fargate (1 task, 0.5 vCPU, 1GB). DB-less mode — declarative config via `kong.yml` mounted from S3 or ConfigMap equivalent. |
| **Route Config** | `/v1/auth/*` → Keycloak (token endpoint proxy). `/v1/parcels/*`, `/v1/agents/*`, `/v1/jobs/*`, `/v1/surveys/*`, `/v1/admin/*` → Go monolith. `/ws` → Go monolith (WebSocket upgrade). |
| **Plugins** | **JWT** (validate Keycloak tokens, JWKS endpoint). **Rate Limiting** (100 req/min basic, 500 pro, 2000 enterprise — stored in Redis). **CORS** (allowed origins). **Request Size Limiting** (10MB for media uploads). **Request Transformer** (inject `X-Request-ID`). |
| **Public Routes** | `/v1/auth/*` and `/v1/billing/webhooks/*` bypass JWT validation (anonymous consumer). |
| **Local Dev** | Kong in docker-compose. `kong.yml` declarative config checked into repo. |

#### Auth Flow (Keycloak-Powered)

```
REGISTRATION (Landowner):
  1. Client → POST /v1/auth/register {phone, full_name}
  2. Go backend → Keycloak Admin API: create user (username=phone, role=landowner)
  3. Go backend → MSG91: send OTP to phone
  4. Go backend → store OTP hash in Redis (TTL 10 min)
  5. Client → POST /v1/auth/verify-otp {phone, otp}
  6. Go backend → verify OTP against Redis
  7. Go backend → Keycloak Admin API: enable user, set email_verified=true
  8. Go backend → Keycloak Token API: exchange credentials → JWT pair
  9. Return {access_token, refresh_token} to client

LOGIN:
  1. Client → POST /v1/auth/login {phone}
  2. Go backend → MSG91: send OTP
  3. Client → POST /v1/auth/verify-otp {phone, otp}
  4. Go backend → verify → Keycloak Token API → JWT pair

AGENT REGISTRATION:
  Same flow but role = "agent", status = "pending_verification"

TOKEN VALIDATION (every API request):
  1. Client sends Authorization: Bearer {access_token}
  2. Kong JWT plugin validates signature against Keycloak JWKS endpoint
     (keys cached, refreshed every 5 min)
  3. If valid → request forwarded to Go monolith with decoded claims in headers
  4. Go auth middleware reads X-Userinfo header (or re-validates token):
     extracts sub, role, plan, tier → injects UserContext into request context
  5. Route-specific middleware checks RequireRole("landowner") / RequireRole("agent")

TOKEN REFRESH:
  1. Client → POST /v1/auth/refresh {refresh_token}
  2. Proxied to Keycloak token endpoint → new access_token
```

**JWT Claims (issued by Keycloak):**

```json
{
  "sub": "user-uuid",
  "realm_access": { "roles": ["landowner"] },
  "subscription_plan": "pro",
  "agent_tier": "experienced",
  "preferred_username": "9876543210",
  "iat": 1708000000,
  "exp": 1708000900,
  "iss": "https://auth.landintel.in/realms/landintel"
}
```

**Sprint 1 Deliverable:** `make dev` starts PostgreSQL + Redis + Keycloak + Kong + Go server. Auth flow working: register → OTP → login → Keycloak JWT. Kong validates tokens and routes to Go app. Terraform deploys dev environment (ECS tasks for Kong, Keycloak, Go). GitHub Actions builds + deploys on push to main.

---

### Sprint 2 (Weeks 3-4): Land Module + Agent Module

**Goal:** Landowners create parcels with map boundaries. Agents register and update location.

#### Land Module

| Endpoint | Implementation Detail |
|----------|----------------------|
| `POST /v1/parcels` | Parse GeoJSON boundary → validate: `ST_IsValid`, area > 50sqm, centroid within India bbox → insert with PostGIS → publish `parcel.registered` |
| `GET /v1/parcels` | List user's parcels with latest risk score (LEFT JOIN LATERAL on risk_scores). Cursor pagination. |
| `GET /v1/parcels/:id` | Full detail: boundary GeoJSON, subscription info, latest survey summary, risk score. Owner-only auth check. |
| `PUT /v1/parcels/:id/boundary` | Validate new boundary → update → log change for audit |
| `POST /v1/parcels/:id/request-visit` | Check subscription → check remaining on-demand visits → create `survey_job` with priority=high, deadline=48hr |

**sqlc query example:**

```sql
-- db/queries/parcels.sql

-- name: CreateParcel :one
INSERT INTO parcels (user_id, label, survey_number, village, taluk, district, state, state_code, pin_code, boundary, land_type, registered_area_sqm)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, ST_GeomFromGeoJSON($10), $11, $12)
RETURNING id, user_id, label, district, state, area_sqm,
          ST_AsGeoJSON(boundary) as boundary_geojson,
          ST_AsGeoJSON(centroid) as centroid_geojson,
          created_at;

-- name: ListParcelsByUser :many
SELECT p.id, p.label, p.district, p.state, p.area_sqm, p.status, p.land_type,
       ST_AsGeoJSON(p.centroid) as centroid_geojson,
       s.plan as subscription_plan,
       s.status as subscription_status,
       (SELECT overall_score FROM risk_scores WHERE parcel_id = p.id ORDER BY computed_at DESC LIMIT 1) as risk_score,
       (SELECT risk_level FROM risk_scores WHERE parcel_id = p.id ORDER BY computed_at DESC LIMIT 1) as risk_level
FROM parcels p
LEFT JOIN subscriptions s ON s.parcel_id = p.id AND s.status = 'active'
WHERE p.user_id = $1 AND p.status = 'active'
ORDER BY p.created_at DESC;

-- name: FindParcelsNeedingSurvey :many
SELECT p.id, p.user_id, s.plan, s.id as subscription_id,
       ST_X(p.centroid) as centroid_lng,
       ST_Y(p.centroid) as centroid_lat
FROM parcels p
JOIN subscriptions s ON s.parcel_id = p.id AND s.status = 'active'
WHERE p.status = 'active'
AND NOT EXISTS (
    SELECT 1 FROM survey_jobs
    WHERE parcel_id = p.id
    AND status IN ('pending_assignment','offered','assigned','agent_en_route','agent_on_site','in_progress','submitted')
);
```

#### Agent Module

| Endpoint | Implementation Detail |
|----------|----------------------|
| `POST /v1/agents/register` | Phone + OTP → create agent (status: pending_verification) → generate agent JWT (role: agent) |
| `PUT /v1/agents/me/profile` | Update name, address, vehicle, radius. Bank details → encrypt before storage. |
| `POST /v1/agents/me/location` | **Hot path (every 60s per agent):** validate lat/lng → `SETEX agent:loc:{id}` in Redis (TTL 5 min) → Redis PUBLISH for real-time tracking. Background goroutine flushes to PostGIS every 5 min. |
| `PUT /v1/agents/me/availability` | Set `is_online` flag. Online → agent enters matching pool. Offline → exits pool, location tracking pauses. |

**Location tracking hot path (optimized):**

```
POST /v1/agents/me/location
  Body: {"lat": 12.97, "lng": 77.59, "accuracy": 8.5}
  Time budget: <10ms (called every 60s per agent)

  1. Validate: lat [-90, 90], lng [-180, 180], accuracy < 100m     ~0.1ms
  2. Redis SETEX "agent:loc:{id}" → JSON with TTL 300s              ~1ms
  3. Redis PUBLISH "agent:{id}:location" → for WebSocket             ~1ms
  4. Return 204 No Content                                           ~0ms
  Total: ~2ms

  Background flusher (every 5 min):
  - Scan Redis keys "agent:loc:*"
  - Batch UPDATE agents SET last_known_location = ..., last_location_at = ...
  - ~50ms for 200 agents
```

**FindNearbyAgents (used by Job Engine):**

```sql
-- name: FindNearbyAgents :many
SELECT id, full_name, avg_rating, completion_rate, tier,
       last_job_completed_at,
       ST_Distance(
           last_known_location::geography,
           ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
       ) / 1000.0 AS distance_km
FROM agents
WHERE status = 'active'
  AND is_online = TRUE
  AND last_location_at > NOW() - INTERVAL '10 minutes'
  AND ST_DWithin(
      last_known_location::geography,
      ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
      $3 * 1000   -- radius in meters
  )
  AND id != ALL($4::uuid[])   -- excluded agent IDs
ORDER BY distance_km ASC
LIMIT 20;
```

**Sprint 2 Deliverable:** Parcels created with validated PostGIS boundaries. Agent registration + profile. Agent location tracking (Redis hot → PostGIS warm). Spatial queries working.

---

### Sprint 3 (Weeks 5-6): Job Allocation Engine

**Goal:** Jobs auto-created, agents matched and offered, cascade on timeout.

#### Job Scheduler

```
Runs as a goroutine with 1-hour ticker.

Every hour:
1. Query FindParcelsNeedingSurvey (parcels with active subscription + no in-flight job)
2. For each parcel:
   a. Calculate if survey is due (last_survey_date + plan_interval <= now + grace_period)
   b. Determine survey_type from plan
   c. Calculate deadline from plan SLA
   d. INSERT survey_job (status: pending_assignment)
   e. Publish "job.created" event on internal event bus
```

#### Matching Algorithm

```
Input:  job (parcel centroid, survey_type, deadline)
Output: ranked list of candidate agents (max 5)

1. Call agent.FindNearbyAgents(centroid, radius, excludedIDs)
   - Round 0: radius = 25km
   - Round 1: radius = 50km
   - Round 2: radius = 100km

2. Filter by certifications:
   - basic_check → any agent
   - detailed_survey → experienced+ tier
   - premium_inspection → senior+ tier

3. Filter by concurrent load:
   - Agent must have < 3 active jobs

4. Filter by rotation:
   - Agent must not have surveyed this parcel in the last cycle

5. Score remaining candidates:
   distance_score  = 1.0 - (distance / max_distance)         × 0.40
   rating_score    = avg_rating / 5.0                         × 0.30
   completion_score = completion_rate                          × 0.20
   freshness_score = min(hours_since_last_job / 48.0, 1.0)   × 0.10

6. Sort by composite score DESC, return top 5
```

#### Cascade Dispatch

```
For each candidate (rank 1 → 5):
  1. Create job_offer record (status: sent, expires_at: now + 30min)
  2. Update job status → offered
  3. Send FCM push notification to agent
  4. Publish to Redis "agent:{id}:offers" (for real-time in-app)
  5. Wait for response:
     - Goroutine subscribes to Redis channel "offer:{id}:response"
     - Timeout after 30 min
  6. On ACCEPT: assign job, notify landowner, done
  7. On DECLINE/TIMEOUT: move to next candidate

After 5 candidates in round → expand radius → repeat (max 3 rounds)
After all rounds exhausted → status = unassigned → alert ops team
```

**Agent accept/decline:**

```
POST /v1/agents/me/jobs/:id/accept
  1. Load offer → verify it's for this agent, not expired
  2. Update offer.status = accepted
  3. Update job: assigned_agent_id, status = assigned
  4. Redis PUBLISH "offer:{id}:response" → "accepted"
  5. Enqueue task: notify_landowner (via task_queue)
  6. Return 200 with job details

POST /v1/agents/me/jobs/:id/decline
  1. Load offer → verify
  2. Update offer.status = declined, decline_reason
  3. Redis PUBLISH "offer:{id}:response" → "declined"
  4. Return 200
```

**Sprint 3 Deliverable:** Jobs auto-created hourly. Matching algorithm finds nearest qualified agent. FCM push notifications sent. Cascade on timeout/decline (3 rounds × 5 agents). Full state machine with enforced transitions.

---

### Sprint 4 (Weeks 7-8): Agent Mobile App (React Native)

**Goal:** Agent app: login → see offers → accept → navigate → complete survey → submit.

#### App Structure

```
mobile/
├── app/                        # Expo Router
│   ├── (auth)/
│   │   └── login.tsx           # Phone + OTP
│   ├── (tabs)/
│   │   ├── home.tsx            # Available job offers
│   │   ├── active.tsx          # Accepted jobs
│   │   ├── earnings.tsx        # Earnings summary
│   │   └── profile.tsx         # Profile + settings
│   └── job/
│       ├── [id]/
│       │   ├── details.tsx     # Job detail before accept
│       │   ├── navigate.tsx    # Navigation to site
│       │   └── survey.tsx      # Survey workflow
│       └── offer/[id].tsx      # Accept/decline offer
│
├── components/
│   ├── survey/
│   │   ├── StepRenderer.tsx    # Renders step by type
│   │   ├── PhotoCapture.tsx    # In-app camera + geotag
│   │   ├── VideoCapture.tsx    # Video recording
│   │   ├── ChecklistStep.tsx   # Yes/No/NA questions
│   │   ├── GPSTraceStep.tsx    # Boundary walk recording
│   │   └── GeofenceGate.tsx    # Blocks survey until arrival
│   └── common/
│       ├── JobCard.tsx         # Offer card in feed
│       └── OfflineBanner.tsx   # Network status
│
├── hooks/
│   ├── useLocation.ts         # Background location
│   ├── useJobOffers.ts        # WebSocket job feed
│   └── useOfflineSync.ts      # SQLite queue + sync
│
├── services/
│   ├── api.ts                 # API client (axios)
│   ├── auth.ts                # Token storage + refresh
│   └── sync.ts                # Offline sync engine
│
└── stores/
    ├── authStore.ts           # Zustand
    └── surveyStore.ts         # In-progress survey data
```

#### Key Flows

**Geofence arrival:**
```
1. Agent taps "Navigate" → Google Maps deep link
2. App tracks location every 15s (foreground)
3. When haversine(current, parcel_centroid) < 100m:
   - Check Android mock location flag
   - POST /v1/agents/me/jobs/{id}/arrive
   - Unlock survey workflow
4. Cannot start survey until geofence confirms
```

**Photo capture:**
```
1. Expo Camera opens (no gallery picker)
2. On capture:
   - Get GPS: {lat, lng, accuracy}
   - Compute SHA-256 hash
   - Save to local filesystem
   - Add to upload queue (SQLite)
3. Upload: GET presigned S3 URL → PUT to S3 → POST metadata to API
```

**Offline mode:**
```
SQLite tables:
  - upload_queue: {id, job_id, file_path, s3_key, status}
  - survey_draft: {job_id, step_id, response_json}
  - location_buffer: {lat, lng, timestamp}

On reconnect:
  1. Flush location_buffer (batch POST)
  2. Upload pending files (presigned URL → S3)
  3. Submit survey if all media uploaded
```

**Sprint 4 Deliverable:** Agent app on iOS + Android. OTP login. Real-time job feed (WebSocket). Accept/decline. Google Maps navigation. In-app camera with geotagging. GPS trace. Geofence gate. Offline support. S3 presigned upload.

---

### Sprint 5 (Weeks 9-10): Landowner Dashboard (Next.js)

**Goal:** Web app: register parcels, track agents, view surveys, manage subscription.

#### Key Pages

| Page | Features |
|------|----------|
| **Dashboard** (`/`) | All parcels on Mapbox map, color-coded by risk (green/yellow/red). Click to expand. Summary stats. |
| **New Parcel** (`/parcels/new`) | Step 1: Enter location details. Step 2: Draw boundary on Mapbox (Mapbox GL Draw). Step 3: Select plan → Razorpay checkout. |
| **Parcel Detail** (`/parcels/[id]`) | Boundary on map. Risk gauge (0-100). Latest survey summary. Next survey date. Action: request on-demand visit. |
| **Live Tracking** (`/parcels/[id]/tracking`) | Agent dot on map (WebSocket updates every 15s). Status: assigned → en route → on site → surveying. |
| **Survey History** (`/parcels/[id]/surveys`) | Timeline of all surveys. Click to see: photos, GPS trace, checklist answers. |
| **Reports** (`/parcels/[id]/reports`) | Download PDF reports. View online summary. |
| **Alerts** (`/alerts`) | Notification feed: survey completed, risk changes, payment confirmations. Mark read. |
| **Settings** (`/settings`) | Profile, notification preferences, subscription management. |

#### WebSocket Integration (Live Tracking)

```
Client connects: ws://api.landintel.in/ws?token=JWT

On connect:
  - Server validates JWT
  - Client sends: { "type": "subscribe", "room": "job:{job_id}" }
  - Server adds client to room

Server pushes (from Redis Pub/Sub):
  - Agent location: { "type": "agent_location", "lat": ..., "lng": ..., "ts": ... }
  - Job status: { "type": "job_status", "status": "agent_on_site" }
  - Survey progress: { "type": "survey_progress", "step": 3, "total": 8 }
```

**Sprint 5 Deliverable:** Landowner dashboard: Mapbox parcel map, boundary drawing, live agent tracking, survey timeline, alert feed. Razorpay subscription checkout. Responsive (works on mobile browsers — no native app needed yet).

---

### Sprint 6 (Weeks 11-12): QA + Reports + Notifications + E2E Testing

**Goal:** Every survey runs through QA. PDF reports generated. Notifications sent. Full loop tested.

#### Automated QA Pipeline

Triggered when survey is submitted (via task_queue):

```
QA CHECKS (sequential, scored):

1. GEOLOCATION (25% weight)
   - For each photo: ST_Contains(parcel_boundary, photo_location)
   - Score = photos_inside / total_photos
   - If < 50% inside → REJECT

2. COMPLETENESS (25% weight)
   - All required checklist steps answered?
   - Minimum photos per step met?
   - Score = completed_steps / required_steps

3. BOUNDARY WALK (20% weight)
   - ST_HausdorffDistance(gps_trail, parcel_boundary)
   - < 50m → score 1.0
   - < 100m → score 0.5
   - > 100m → score 0.0, FLAG

4. TIMESTAMPS (15% weight)
   - All media within 2-hour window?
   - Total on-site time > 15 min?
   - Timestamps sequential?

5. DUPLICATE DETECTION (15% weight)
   - Perceptual hash (pHash) of each photo
   - Compare against previous survey photos for same parcel
   - > 80% similarity → FLAG

RESULT:
- Score >= 0.7 → auto_passed → generate report
- Score 0.5-0.7 → flagged → human QA queue
- Score < 0.5 → rejected → re-schedule survey
- Random 20% → flagged for human review regardless
```

#### PDF Report Generation

```
Triggered by: task_queue item "generate_report"

1. Load: job + parcel + survey_response + all media + previous survey + risk score
2. Render HTML template (Go html/template):
   - Mapbox Static Images API: boundary + GPS trace + photo points
   - Photo grid: presigned S3 URLs (thumbnails)
   - Checklist answers formatted as table
   - Risk gauge (SVG)
   - Side-by-side with previous survey (if exists)
3. HTML → PDF via chromedp (headless Chrome) or wkhtmltopdf
4. Upload PDF to S3: reports/{parcel_id}/{date}.pdf
5. Enqueue notification tasks: email (SendGrid), SMS (MSG91), push (FCM)
```

#### Notification Module

```
Channels (Phase 1):
  1. Email (SendGrid) — report delivery, account alerts
  2. SMS (MSG91) — OTP, survey status, critical alerts
  3. Push (FCM) — job offers (agent), survey updates (landowner)
  4. In-app (WebSocket + alerts table) — all notifications

Routing:
  "report.generated" → email (with PDF) + push + in-app
  "job.offered"      → push to agent + in-app
  "job.assigned"     → push to landowner + in-app
  "risk.high"        → email + SMS + push + in-app
```

#### E2E Testing

```
Full loop test (automated):
  1. Register user → get JWT
  2. Create parcel with boundary
  3. Create subscription (mock Razorpay)
  4. Trigger scheduler → job created
  5. Register agent near parcel
  6. Agent goes online → matching finds agent
  7. Agent accepts offer
  8. Agent marks arrival (location within geofence)
  9. Agent submits survey (mock photos/responses)
  10. QA pipeline runs → passes
  11. Report generated → PDF in S3
  12. Notification sent → alert in DB
  13. Landowner fetches report → 200 OK

Target: < 5% failure rate on this flow.
```

**Sprint 6 Deliverable:** Full E2E loop working. QA validates every survey. PDF reports generated and downloadable. Email + SMS + push notifications operational. Ready for alpha testing with real users.

---

## PHASE 2: Scale + Monetize (Weeks 13-24)

**Phase 2 adds what we deliberately skipped: agent training, KYC, billing, legal intelligence, risk ML, and admin tools.**

| Sprint | Weeks | Focus | Key Deliverables |
|--------|-------|-------|-----------------|
| **7** | 13-14 | **Agent Training + KYC** | In-app video training + quiz. DigiLocker Aadhaar eKYC. Bank verification (penny drop). Agent tier auto-promotion. |
| **8** | 15-16 | **Billing + Payouts** | Razorpay subscription integration. Agent wallet. Weekly batch payouts via Razorpay Payouts. On-demand visit billing. |
| **9** | 17-18 | **Legal Intelligence (Python enters)** | Scrapy spiders for Karnataka (eCourts, Kaveri, Gazette). Fuzzy entity matching. Results stored in PostgreSQL. Events trigger risk re-scoring. |
| **10** | 19-20 | **Risk Scoring ML** | XGBoost model trained on survey data. SHAP explainability. Replaces rule-based scoring. Monthly retraining pipeline. |
| **11** | 21-22 | **Admin Dashboard + Landowner Mobile** | QA review interface. Agent management. System metrics. Landowner React Native app (PWA wasn't enough). |
| **12** | 23-24 | **Beta Hardening** | Survey comparison tool. Load testing (Locust). Security audit. Analytics dashboard. WhatsApp notifications (Gupshup). |

**Phase 2 Exit Criteria:** 100 beta users, 50 agents, 2 cities (Bangalore + Hyderabad). Legal intelligence live for Karnataka. Agent payouts operational. ML risk scoring deployed.

---

## PHASE 3: Multi-City + Enterprise (Weeks 25-36)

| Sprint | Weeks | Focus | Key Deliverables |
|--------|-------|-------|-----------------|
| **13-14** | 25-28 | **6 New Cities** | Chennai, Mumbai, Pune, Delhi NCR, Kolkata, Ahmedabad. State-specific legal scrapers. i18n (Hindi, Tamil, Telugu, Marathi). Agent recruitment. |
| **15-16** | 29-32 | **Enterprise API** | API key auth. Batch verification endpoint. White-label reports. Custom checklists. SLA monitoring. |
| **17-18** | 33-36 | **Growth Features** | Pre-purchase reports (one-time). Route optimization (batch nearby parcels). Drone pilot (2 cities). Stripe (international). |

**Phase 3 Exit Criteria:** 8 cities, 500+ users, 200+ agents, 5L+ MRR, first enterprise client.

---

## Summary: What We Build, When

```
PHASE 1 (Weeks 1-12): Ship the Core Loop
──────────────────────────────────────────
Sprint 1: Infra (Terraform + ECS) + Auth (Go JWT + OTP)
Sprint 2: Land module (parcels + PostGIS) + Agent module (registration + location)
Sprint 3: Job Engine (matching + dispatch + cascade)
Sprint 4: Agent Mobile App (React Native + camera + GPS + offline)
Sprint 5: Landowner Dashboard (Next.js + Mapbox + WebSocket)
Sprint 6: QA + Reports (PDF) + Notifications (email/SMS/push) + E2E tests

Tech: Go monolith + Keycloak + Kong + PostgreSQL + Redis + S3 + ECS Fargate
Cost: ~$136/mo dev, ~$600-800/mo prod
Team: 5 (2 backend Go, 1 mobile RN, 1 frontend Next.js, 1 fullstack/devops)

PHASE 2 (Weeks 13-24): Scale to 100 Users
──────────────────────────────────────────
Sprints 7-12: Training, KYC, billing, legal (Python enters), ML risk, admin

Added: Python workers (Scrapy + XGBoost), Razorpay Payouts, WhatsApp
Cost: ~$250/mo dev, ~$1000-1500/mo prod
Team: 8 (+1 backend, +1 data eng, +1 agent ops)

PHASE 3 (Weeks 25-36): Scale to 500+ Users
──────────────────────────────────────────
Sprints 13-18: Multi-city, enterprise API, i18n, Stripe, drone pilot

Added: Stripe, route optimization, enterprise endpoints
Cost: ~$400/mo dev, ~$2000-3000/mo prod
Team: 12-14 (+more engineering, sales, ops)
```

---

## Implementation Progress

| Sprint | Status | Key Deliverables |
|--------|--------|-----------------|
| Sprint 1 | **COMPLETE** | Go skeleton, docker-compose (PG+Redis+Keycloak+Kong), 8 migrations, sqlc codegen, auth module (register/login/OTP/refresh), JWT middleware, CI pipeline |
| Sprint 2 | **COMPLETE** | Land module (parcel CRUD, GeoJSON validation, India bbox, owner-scoped), Agent module (registration, profile, location hot path Redis→PostGIS flusher, FCM token, availability toggle) |
| Sprint 3 | IN PROGRESS | Job allocation engine |
| Sprint 4 | Not started | Agent mobile app (React Native) |
| Sprint 5 | Not started | Landowner dashboard (Next.js) |
| Sprint 6 | Not started | QA + Reports + Notifications + E2E |

### Codebase Structure (as-built)
```
cmd/server/main.go          — Server entry, wires all modules
internal/
  auth/                      — Sprint 1: Keycloak + OTP + JWT middleware
    handler.go, service.go, repository.go, middleware.go, keycloak.go, otp.go
  land/                      — Sprint 2: Parcel CRUD + boundary validation
    handler.go, service.go, repository.go, validation.go
  agent/                     — Sprint 2: Agent registration + profile + location
    handler.go, service.go, repository.go, location.go
  job/                       — Sprint 3: Job allocation engine (planned)
    doc.go (placeholder)
  platform/                  — Shared infra: errors, httputil, eventbus, taskqueue, config, etc.
db/
  migrations/                — 8 migration files (001-008)
  queries/                   — sqlc SQL (users, parcels, agents, jobs, surveys, alerts, billing, tasks)
  sqlc/                      — Generated Go code
```

---

**END OF DOCUMENT**

*LandIntel v4.0 | Startup-Optimized Architecture | Modular Monolith | Ship Fast, Scale Later*
