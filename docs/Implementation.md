# LANDINTEL — Land Intelligence + Field Verification Platform

## Revised Architecture: Rapido-Style Field Agent Model

**Version 2.0 | February 2026 | CONFIDENTIAL**

Human-Powered Verification Network | Gig-Economy Workforce | Real-Time Job Allocation

---

# Table of Contents

1. [Revised Platform Concept](#1-revised-platform-concept)
2. [Revised System Architecture](#2-revised-system-architecture)
3. [Field Agent Ecosystem (Supply Side)](#3-field-agent-ecosystem-the-supply-side)
4. [Database Schema (Revised)](#4-database-schema-revised)
5. [Feature-by-Feature Implementation](#5-feature-by-feature-implementation)
6. [Infrastructure as Code (Revised)](#6-infrastructure-as-code-revised)
7. [Development Phases (Revised Roadmap)](#7-development-phases-revised-roadmap)
8. [Complete Third-Party Integration List](#8-complete-third-party-integration-list)
9. [Security & Anti-Fraud (Agent-Specific)](#9-security--anti-fraud-agent-specific)
10. [Unit Economics: Agent Model vs Satellite Model](#10-unit-economics-agent-model-vs-satellite-model)
11. [API Endpoint Reference](#11-api-endpoint-reference)
12. [Environment Variables Reference](#12-environment-variables-reference)

---

# 1. Revised Platform Concept

## 1.1 What Changed and Why

The original LandIntel design relied on satellite imagery + AI/ML models for encroachment detection. This revision **replaces that entire detection layer** with a human-powered field verification network, modeled after the Rapido/Uber driver allocation system.

| Aspect | Previous (Satellite + AI) | Revised (Field Agent Network) |
|--------|--------------------------|-------------------------------|
| **Detection Method** | Sentinel-2/Planet satellite imagery, CNN-based change detection, NDVI analysis | On-ground field agents physically visit the site, take geotagged photos/videos, fill structured survey checklists |
| **Accuracy** | 70-85% (high false positive rate, cloud cover issues, 10m resolution limits) | 95%+ (human eyes on ground, photos as evidence, verifiable visit via GPS) |
| **Latency** | 5-30 day satellite revisit cycle, processing delay | 24-72 hours from job dispatch to completed survey report |
| **Cost Structure** | High fixed cost (GPU inference, satellite API fees, ML training) | Variable cost (pay-per-visit), scales linearly with demand |
| **Coverage** | Limited by satellite resolution; cannot detect small sheds/fencing at 10m | Can detect anything visible: small huts, fencing changes, soil dumping, marker stone removal |
| **Evidence Quality** | Satellite image overlay (hard to use in court) | Geotagged timestamped photos + video + signed survey form (strong legal evidence) |
| **Infrastructure** | GPU nodes, ML pipeline, Airflow DAGs, model training | Mobile app for agents, job allocation engine, photo/video storage |
| **Scalability** | Compute-limited (GPU costs grow with parcels) | Labor-limited (recruit more agents), but unit economics improve at scale |

## 1.2 The Rapido Analogy (Core Mental Model)

> **Think of it this way:** When a user subscribes and registers a land parcel, it is like a customer booking a ride. The system must find the nearest available qualified field agent, offer them the job, and get them to the site within the SLA window. Just like Rapido allocates a driver to a rider.

| Rapido Ride | LandIntel Survey Job | System Parallel |
|-------------|---------------------|-----------------|
| Customer opens app, enters destination | Landowner subscribes, registers parcel with GPS coordinates | Job creation trigger |
| System finds nearest available driver | System finds nearest available field agent with right skills | Matching algorithm |
| Driver gets notification with ride details | Agent gets push notification with parcel location, survey type, deadline | Job offer dispatch |
| Driver accepts/declines within timeout | Agent accepts/declines within 30 min; if declined, cascades to next | Accept/timeout/cascade logic |
| Customer sees driver on map | Landowner sees agent status: assigned, en-route, on-site, completed | Real-time tracking |
| Driver completes ride, gets paid | Agent completes survey, uploads report, payment credited to wallet | Completion + settlement |
| Customer rates driver | Landowner rates survey quality; platform QA reviews report | Quality scoring |

## 1.3 Revised Platform Summary

LandIntel is now a **two-sided marketplace platform**:

- **Side A:** Landowners who subscribe and register parcels for monitoring.
- **Side B:** Field agents (gig workers) who are recruited, trained, and allocated survey jobs.

The platform handles everything in between:

- Job scheduling based on subscription frequency
- Intelligent agent matching and dispatch
- Structured survey workflows via a mobile app
- Geotagged evidence collection with tamper-proofing
- Quality assurance and report generation
- Payment settlement and agent performance management

---

# 2. Revised System Architecture

## 2.1 High-Level Architecture

The system has three primary interfaces (Landowner Web/Mobile, Agent Mobile App, Admin Dashboard) feeding into a core platform with **job allocation as the central nervous system**.

```
LANDOWNER APP/WEB                    AGENT MOBILE APP                 ADMIN DASHBOARD
       │                                    │                               │
       ▼                                    ▼                               ▼
  ══════════════════════════════════════════════════════════════════════════════════
  │                              API GATEWAY (Kong)                               │
  ══════════════════════════════════════════════════════════════════════════════════
       │                │                │                │                │
       ▼                ▼                ▼                ▼                ▼
  ┌──────────┐   ┌────────────┐   ┌─────────────┐   ┌─────────────┐   ┌──────────┐
  │  Auth     │   │   Land     │   │    Job      │   │   Agent     │   │ Report   │
  │  Service  │   │  Service   │   │ Allocation  │   │  Service    │   │ Service  │
  │(Keycloak) │   │(Parcel     │   │  Engine     │   │(Profile,    │   │(PDF gen, │
  │           │   │ CRUD,      │   │(Matching,   │   │ location,   │   │ QA,      │
  │           │   │ boundary)  │   │ dispatch,   │   │ skills,     │   │ photos)  │
  │           │   │            │   │ cascade)    │   │ ratings)    │   │          │
  └──────────┘   └────────────┘   └─────────────┘   └─────────────┘   └──────────┘
       │                │                │                │                │
       ▼                ▼                ▼                ▼                ▼
  ══════════════════════════════════════════════════════════════════════════════════
  │  PostgreSQL+PostGIS  │  Redis  │  S3 (photos/video)  │  RabbitMQ  │  ES       │
  ══════════════════════════════════════════════════════════════════════════════════
```

### Service Descriptions

| Service | Responsibility | Technology |
|---------|---------------|------------|
| **Auth Service** | User/agent authentication, JWT tokens, RBAC | Keycloak, OAuth2.0 |
| **Land Service** | Parcel registration, boundary CRUD, ownership records, PostGIS queries | FastAPI, GeoAlchemy2 |
| **Job Allocation Engine** | The core — matching, offer dispatch, cascade, scheduling, state machine | FastAPI, Celery, RabbitMQ |
| **Agent Service** | Agent profiles, KYC, availability, location tracking, performance, tiers | FastAPI, Redis (location), FCM |
| **Report Service** | Survey report PDF generation, photo processing, QA checks | FastAPI, WeasyPrint, imagehash |
| **Billing Service** | Subscriptions, landowner payments, agent payouts | FastAPI, Razorpay, Stripe |
| **Alert Service** | Event-driven notifications: email, SMS, WhatsApp, push | FastAPI, SendGrid, MSG91, Gupshup |
| **Legal Intelligence Service** | Web scraping of court/registry portals, entity matching | Scrapy, Playwright, spaCy |

## 2.2 The Job Allocation Engine — How It Works (Step by Step)

This is the **heart of the entire system**, equivalent to Rapido's ride-matching algorithm. It replaces the satellite pipeline + ML inference layer from v1.

### Step 1 — Job Generation (Automatic Scheduling)

A background scheduler (Celery Beat) runs every hour. It queries all active parcels and checks:

- When was the last completed survey for this parcel?
- Based on the subscription tier, is a new survey due?
- If due → create a `SurveyJob` record with status = `pending_assignment`
- Publish a `JobCreatedEvent` to RabbitMQ

```python
# Simplified scheduler logic (Celery Beat task)

@celery_app.task
def schedule_pending_surveys():
    now = datetime.utcnow()
    
    parcels = db.query(Parcel).join(Subscription).filter(
        Subscription.status == 'active'
    ).all()
    
    for parcel in parcels:
        last_survey = db.query(SurveyJob).filter(
            SurveyJob.parcel_id == parcel.id,
            SurveyJob.status == 'completed'
        ).order_by(SurveyJob.created_at.desc()).first()
        
        interval = get_frequency_interval(parcel.subscription.plan)
        buffer = timedelta(days=3)
        
        if last_survey:
            next_due = last_survey.survey_submitted_at + interval
        else:
            next_due = parcel.registered_at  # First survey ASAP
        
        if next_due <= now + buffer:
            existing = db.query(SurveyJob).filter(
                SurveyJob.parcel_id == parcel.id,
                SurveyJob.status.in_([
                    'pending_assignment', 'offered', 'assigned',
                    'agent_en_route', 'agent_on_site', 'in_progress'
                ])
            ).first()
            
            if not existing:
                job = create_survey_job(parcel, next_due)
                publish_event('job.created', job.id)
```

### Step 2 — Agent Matching (The Core Algorithm)

The Job Allocation Engine consumes the `JobCreatedEvent` and runs the matching algorithm:

```
MATCHING ALGORITHM

Input:  SurveyJob (parcel location, survey type, deadline, required skills)
Output: Ranked list of eligible agents

1. GEOGRAPHIC FILTER
   - Query agents within configurable radius (default: 25km) of parcel centroid
   - PostGIS: ST_DWithin(agent.last_location, parcel.centroid, 25000)
   - If < 3 agents found → expand to 50km, then 100km

2. AVAILABILITY FILTER
   - Agent must be 'active' status (not on leave, not suspended)
   - Agent must not have > N active jobs (configurable, default: 3 concurrent)
   - Agent must not have surveyed this same parcel in the last cycle (rotation)

3. SKILL FILTER
   - Agent must hold required certifications for the survey type
     - 'basic_survey'         = any trained agent
     - 'legal_verification'   = agent with legal training badge
     - 'premium_inspection'   = senior agent with 50+ completed jobs

4. SCORING (for remaining candidates)
   ┌────────────────────┬────────┬──────────────────────────────────┐
   │ Factor             │ Weight │ Formula                          │
   ├────────────────────┼────────┼──────────────────────────────────┤
   │ Distance           │ 40%    │ 1 - (distance_km / max_radius)   │
   │ Rating             │ 30%    │ agent.avg_rating / 5.0           │
   │ Completion rate    │ 20%    │ agent.completion_rate (0-1)       │
   │ Freshness (idle)   │ 10%    │ min(hours_since_last_job / 48, 1) │
   └────────────────────┴────────┴──────────────────────────────────┘

5. RANK by composite score → return top 5 candidates
```

### Step 3 — Job Offer Dispatch (Cascade Model)

The system offers the job to the **highest-ranked agent first** via:

- Push notification (Firebase Cloud Messaging)
- In-app notification

**What the agent sees:** parcel location on map, distance from current location, survey type + estimated duration, payout amount, deadline.

**Cascade logic:**

- Agent has **30 minutes** to accept
- If accepted → job status = `assigned`, landowner notified
- If declined or timeout → offer cascades to next agent in ranked list
- If all 5 candidates exhaust → re-run matching with expanded radius
- If still no match after **3 cascade rounds** → flag for manual assignment by ops team

```python
MAX_CASCADE_ROUNDS = 3
AGENTS_PER_ROUND = 5
OFFER_TIMEOUT_MINUTES = 30

async def handle_job_created(job_id: str):
    job = await get_job(job_id)
    
    for round_num in range(MAX_CASCADE_ROUNDS):
        candidates = await find_matching_agents(
            parcel=job.parcel,
            survey_type=job.survey_type,
            excluded_agent_ids=get_already_offered_agents(job_id),
            expand_radius=(round_num > 0),
            required_tier=get_required_tier(job.parcel.subscription.plan)
        )
        
        if not candidates:
            if round_num < MAX_CASCADE_ROUNDS - 1:
                continue  # try with expanded radius
            else:
                await mark_job_unassigned(job_id)
                await notify_ops_team(job_id, "No agent found after all cascades")
                return
        
        for rank, agent in enumerate(candidates[:AGENTS_PER_ROUND]):
            offer = await create_offer(job_id, agent.id, rank + 1)
            await send_push_notification(agent.fcm_token, {
                "type": "new_job_offer",
                "job_id": job_id,
                "parcel_location": job.parcel.centroid_coords,
                "survey_type": job.survey_type,
                "payout": calculate_payout(job, agent),
                "deadline": job.deadline.isoformat(),
                "expires_at": offer.expires_at.isoformat()
            })
            
            response = await wait_for_response(
                offer.id, timeout=OFFER_TIMEOUT_MINUTES * 60
            )
            
            if response == "accepted":
                await assign_job_to_agent(job_id, agent.id)
                await notify_landowner(job.parcel.user_id, "Agent assigned!")
                return
            elif response == "declined":
                continue
            else:  # expired
                continue
    
    await mark_job_unassigned(job_id)
```

### Step 4 — Agent En Route + On-Site

- Agent's app shows **navigation to parcel** (Google Maps deep link)
- When agent arrives within the parcel's **geofence** (configurable, default: 100m radius), the app auto-marks `arrived` and **unlocks the survey workflow**
- Agent **cannot start the survey checklist until GPS confirms** they are physically at the site — this prevents fake surveys

### Step 5 — Survey Execution (Structured Workflow)

The agent app presents a **step-by-step survey checklist** (details in Section 5). Each step requires:

- **Geotagged photos** (camera must be used in-app, no gallery uploads)
- **Checklist answers** (multiple choice + free text)
- **Boundary walk** (GPS trace as agent walks the perimeter)
- **Video walkthrough** (30-60 second mandatory video)

All media uploaded in real-time to S3 with metadata: GPS coordinates, timestamp, device ID, agent ID. App prevents submission until all mandatory steps are completed.

### Step 6 — Submission + QA

Automated QA checks run on every submission:

- Were all photos geotagged within the parcel boundary?
- Was the GPS trace consistent with the parcel perimeter?
- Were all checklist items answered?
- Was the video of sufficient duration?

If automated QA passes → report generated and delivered to landowner. A random **20% of surveys** are additionally flagged for human QA review.

### Step 7 — Payment Settlement

- On successful completion + QA pass → agent wallet credited
- Payouts batched weekly via bank transfer (UPI/NEFT via **Razorpay Payouts**)
- Agents see earnings, pending payouts, and payout history in the app

---

# 3. Field Agent Ecosystem (The Supply Side)

## 3.1 Who Are the Agents?

Field agents are gig workers recruited from the local area. Ideal profile:

- Familiarity with local geography
- Basic smartphone proficiency
- Ability to travel to rural/semi-urban sites
- Willingness to work flexible hours

**Target demographics:** college students, part-time workers, retired revenue department staff, local surveyors, existing gig workers (Swiggy/Rapido/Dunzo).

## 3.2 Agent Lifecycle

| Stage | What Happens | System Components |
|-------|-------------|-------------------|
| **1. Recruitment** | Agent downloads app, fills profile (name, Aadhaar/PAN, bank details, location, vehicle type, preferred radius). Background verification via Aadhaar eKYC. | Agent App registration, DigiLocker/Aadhaar eKYC API, Admin approval queue |
| **2. Training** | In-app training module: video tutorials on survey process, practice quiz, mock survey. Must score > 80% to activate. | In-app training module, video + quiz engine, auto-grading |
| **3. Activation** | Profile verified + training passed → agent marked `active`. Appears in job matching pool. Gets `basic_survey` certification. | Agent Service status update, certification grant, welcome notification |
| **4. Job Execution** | Receives job offers, accepts, travels to site, completes survey, submits. Earns per-job payout. | Job Allocation Engine, Agent App survey workflow, Payment Service |
| **5. Performance Management** | Rating tracked per survey (landowner rating + QA score). Weekly performance digest. Low performers get retraining alerts. Top performers get premium job access. | Rating aggregation, performance dashboard, automated alerts |
| **6. Leveling Up** | 25 jobs + avg rating > 4.0 → `Experienced`. 100 jobs → eligible for `Senior` badge. 250 jobs → `Expert` tier with highest payouts. | Agent tier system, automatic promotion rules, badge engine |
| **7. Deactivation** | Rating below 3.0 for 10 consecutive surveys, or 3 QA failures → suspended. Appeal process via admin. | Automated suspension, appeal workflow, admin review |

## 3.3 Agent Payout Structure

| Survey Type | Base Payout | Distance Bonus | Urgency Bonus | Total Range |
|------------|-------------|----------------|---------------|-------------|
| Basic Site Check | ₹200 | +₹5/km beyond 10km | +₹100 if < 24hr deadline | ₹200 - ₹500 |
| Detailed Survey | ₹400 | +₹5/km beyond 10km | +₹150 if < 24hr deadline | ₹400 - ₹800 |
| Premium Inspection | ₹800 | +₹8/km beyond 10km | +₹200 if < 24hr deadline | ₹800 - ₹1,500 |
| Legal Verification | ₹600 | +₹5/km beyond 10km | N/A (always 48hr) | ₹600 - ₹1,000 |
| Emergency Visit | ₹1,000 | +₹10/km | N/A (inherently urgent) | ₹1,000 - ₹2,000 |

**Platform commission:** 20% of payout is the platform's margin. For a ₹400 survey, the agent receives ₹400 and the platform's cost is ₹500 (₹400 payout + ₹100 margin earned from subscription).

## 3.4 Agent Mobile App Features

| Screen / Feature | Functionality | Technical Notes |
|-----------------|---------------|-----------------|
| **Home / Job Feed** | Available job offers nearby. Cards show: parcel location, distance, payout, deadline, survey type. Accept/Decline buttons. | Real-time via WebSocket (Socket.IO). Location updated every 60s via background GPS. |
| **Active Jobs** | Accepted jobs with status: Accepted → En Route → On Site → In Progress → Submitted. Tap to navigate or resume survey. | Job state machine managed server-side. Geofence check on arrival. |
| **Survey Workflow** | Step-by-step guided checklist. Mandatory photo/video capture per step. GPS trace for boundary walk. Cannot skip steps. | Camera forced in-app (no gallery). EXIF metadata embedded. Offline mode with sync-on-reconnect. |
| **Navigation** | One-tap navigation to parcel via Google Maps / Apple Maps deep link. | Intent/URI scheme for native maps app. |
| **Earnings Dashboard** | Total earnings (weekly/monthly), pending payouts, payout history, next payout date. | Razorpay Payouts API for disbursement. |
| **Profile & Certifications** | Personal details, bank info, current tier/badge, training modules, rating history. | Aadhaar eKYC for verification. Bank details validated via penny drop. |
| **Training Center** | Video tutorials, practice quizzes, certification tests for advanced survey types. | Video hosted on S3/CloudFront. Quiz engine with question bank. |
| **Support / Help** | In-app chat support, FAQ, report issue with specific job. | Intercom / Freshdesk integration. |

## 3.5 Real-Time Agent Location Tracking

- Agent app sends GPS every **60 seconds** when online (background location service)
- Sent as lightweight `POST /v1/agents/me/location`
- Stored in **Redis** (hot) as `agent:{id}:location` with 5-min TTL
- Periodically flushed to PostGIS (`agents.last_known_location`) for matching queries

**Battery optimization:** Android WorkManager / iOS Background Tasks. Reduce to every 5 min when no active job. Full tracking (every 15s) only when `agent_en_route` or `agent_on_site`.

**Privacy:** Location tracked only when agent is `online`. Goes offline = tracking stops. Location history purged after 30 days except for survey GPS traces.

---

# 4. Database Schema (Revised)

## 4.1 New Core Tables

```sql
-- ═══════════════════════════════════════
-- AGENTS
-- ═══════════════════════════════════════

CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Personal Info
    full_name VARCHAR(200) NOT NULL,
    phone VARCHAR(15) NOT NULL UNIQUE,
    email VARCHAR(200),
    aadhaar_hash VARCHAR(64),                -- SHA-256 hash, never store raw
    date_of_birth DATE,
    
    -- Location
    home_location GEOMETRY(POINT, 4326),
    last_known_location GEOMETRY(POINT, 4326),
    last_location_at TIMESTAMPTZ,
    preferred_radius_km INTEGER DEFAULT 25,
    state_code VARCHAR(10),
    district_code VARCHAR(10),
    
    -- Status & Tier
    status VARCHAR(20) DEFAULT 'pending_verification',
        -- pending_verification | training | active | suspended | deactivated
    tier VARCHAR(20) DEFAULT 'basic',
        -- basic | experienced (25+ jobs) | senior (100+) | expert (250+)
    vehicle_type VARCHAR(20),                -- bike | car | bicycle | none
    
    -- Performance Metrics
    total_jobs_completed INTEGER DEFAULT 0,
    avg_rating NUMERIC(3,2) DEFAULT 0.0,
    completion_rate NUMERIC(3,2) DEFAULT 1.0,
    qa_pass_rate NUMERIC(3,2) DEFAULT 1.0,
    
    -- Financial (encrypted)
    bank_account_number_enc TEXT,
    bank_ifsc VARCHAR(11),
    upi_id VARCHAR(100),
    wallet_balance NUMERIC(10,2) DEFAULT 0.0,
    
    -- Certifications
    certifications JSONB DEFAULT '["basic_survey"]',
    
    -- Push notifications
    fcm_token TEXT,
    device_id VARCHAR(100),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agents_location ON agents USING GIST(last_known_location);
CREATE INDEX idx_agents_status ON agents(status, state_code);


-- ═══════════════════════════════════════
-- AGENT AVAILABILITY
-- ═══════════════════════════════════════

CREATE TABLE agent_availability (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agents(id),
    is_online BOOLEAN DEFAULT FALSE,
    went_online_at TIMESTAMPTZ,
    went_offline_at TIMESTAMPTZ,
    available_days JSONB DEFAULT '[]',       -- ["mon","tue","wed","thu","fri"]
    available_hours_start TIME,
    available_hours_end TIME
);


-- ═══════════════════════════════════════
-- SURVEY JOBS (central entity)
-- ═══════════════════════════════════════

CREATE TABLE survey_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id UUID NOT NULL REFERENCES parcels(id),
    subscription_id UUID REFERENCES subscriptions(id),
    
    -- Job Details
    survey_type VARCHAR(30) NOT NULL,
        -- basic_check | detailed_survey | premium_inspection
        -- legal_verification | emergency_visit
    checklist_template_id UUID REFERENCES checklist_templates(id),
    priority VARCHAR(10) DEFAULT 'normal',   -- low | normal | high | urgent
    deadline TIMESTAMPTZ NOT NULL,
    
    -- Allocation State Machine
    status VARCHAR(30) DEFAULT 'pending_assignment',
        -- pending_assignment   → system finding agent
        -- offered              → awaiting agent response
        -- assigned             → agent accepted
        -- agent_en_route       → agent traveling
        -- agent_on_site        → geofence confirmed
        -- in_progress          → survey started
        -- submitted            → pending QA
        -- qa_review            → human QA in progress
        -- completed            → done, report delivered
        -- failed_qa            → rejected, needs re-survey
        -- cancelled            → cancelled by system/admin
        -- unassigned           → no agent found
    
    -- Agent Assignment
    assigned_agent_id UUID REFERENCES agents(id),
    assigned_at TIMESTAMPTZ,
    offer_cascade_count INTEGER DEFAULT 0,
    current_offer_agent_id UUID,
    offer_sent_at TIMESTAMPTZ,
    offer_expires_at TIMESTAMPTZ,
    
    -- Execution Tracking
    agent_arrived_at TIMESTAMPTZ,
    survey_started_at TIMESTAMPTZ,
    survey_submitted_at TIMESTAMPTZ,
    
    -- Geofence Verification
    arrival_location GEOMETRY(POINT, 4326),
    arrival_distance_meters REAL,
    
    -- Payout
    base_payout NUMERIC(8,2),
    distance_bonus NUMERIC(8,2) DEFAULT 0,
    urgency_bonus NUMERIC(8,2) DEFAULT 0,
    total_payout NUMERIC(8,2),
    payout_status VARCHAR(20) DEFAULT 'pending',
        -- pending | credited | paid_out | disputed
    
    -- Quality
    landowner_rating NUMERIC(2,1),
    qa_score NUMERIC(3,2),
    qa_reviewed_by UUID,
    qa_notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_jobs_status ON survey_jobs(status);
CREATE INDEX idx_jobs_parcel ON survey_jobs(parcel_id, created_at DESC);
CREATE INDEX idx_jobs_agent ON survey_jobs(assigned_agent_id, status);
CREATE INDEX idx_jobs_deadline ON survey_jobs(deadline)
    WHERE status IN ('pending_assignment', 'offered', 'assigned');


-- ═══════════════════════════════════════
-- JOB OFFER LOG (cascade history)
-- ═══════════════════════════════════════

CREATE TABLE job_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES survey_jobs(id),
    agent_id UUID NOT NULL REFERENCES agents(id),
    offer_order INTEGER NOT NULL,            -- 1st, 2nd, 3rd in cascade
    distance_km REAL,
    match_score NUMERIC(4,3),
    
    status VARCHAR(20) DEFAULT 'sent',
        -- sent | accepted | declined | expired | cancelled
    sent_at TIMESTAMPTZ DEFAULT NOW(),
    responded_at TIMESTAMPTZ,
    response_channel VARCHAR(20),            -- push_notification | in_app | auto_expired
    decline_reason VARCHAR(100)              -- too_far | too_busy | personal | other
);

CREATE INDEX idx_offers_job ON job_offers(job_id, offer_order);


-- ═══════════════════════════════════════
-- CHECKLIST TEMPLATES
-- ═══════════════════════════════════════

CREATE TABLE checklist_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    survey_type VARCHAR(30) NOT NULL,
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    steps JSONB NOT NULL
    -- Example steps structure:
    -- [
    --   { "step_id": "boundary_photos", "title": "Boundary Photos",
    --     "type": "photo_capture", "min_photos": 4, "required": true,
    --     "instructions": "Take photos of all 4 boundaries..." },
    --   { "step_id": "encroachment_check", "title": "Encroachment Check",
    --     "type": "checklist", "required": true,
    --     "questions": [
    --       { "q_id": "enc_1", "text": "Any new structures visible?",
    --         "type": "yes_no_na", "photo_required_if": "yes" },
    --       { "q_id": "enc_2", "text": "Boundary markers intact?",
    --         "type": "yes_no_na" }
    --     ] },
    --   { "step_id": "boundary_walk", "title": "Walk the Boundary",
    --     "type": "gps_trace", "required": true },
    --   { "step_id": "video_walkthrough", "title": "Video Walkthrough",
    --     "type": "video_capture", "min_duration_sec": 30, "required": true }
    -- ]
);


-- ═══════════════════════════════════════
-- SURVEY RESPONSES
-- ═══════════════════════════════════════

CREATE TABLE survey_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES survey_jobs(id),
    agent_id UUID NOT NULL REFERENCES agents(id),
    template_id UUID REFERENCES checklist_templates(id),
    
    responses JSONB NOT NULL,
    -- Example:
    -- { "boundary_photos": { "photos": ["s3://..."], "completed_at": "..." },
    --   "encroachment_check": { "answers": { "enc_1": "yes", "enc_2": "no" } },
    --   "boundary_walk": { "gps_trace": { "type": "LineString", "coordinates": [] } },
    --   "video_walkthrough": { "video_url": "s3://...", "duration_sec": 45 } }
    
    gps_trail GEOMETRY(LINESTRING, 4326),    -- breadcrumb during entire survey
    
    device_info JSONB,
    -- { "device_id": "...", "os": "android", "os_version": "14",
    --   "app_version": "1.2.0", "battery_level": 72, "network": "4g" }
    
    submitted_at TIMESTAMPTZ DEFAULT NOW()
);


-- ═══════════════════════════════════════
-- SURVEY MEDIA (photos, videos)
-- ═══════════════════════════════════════

CREATE TABLE survey_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES survey_jobs(id),
    agent_id UUID NOT NULL REFERENCES agents(id),
    step_id VARCHAR(100) NOT NULL,
    
    media_type VARCHAR(10) NOT NULL,         -- photo | video
    s3_key TEXT NOT NULL,
    s3_bucket VARCHAR(100) NOT NULL,
    
    -- Geotag
    location GEOMETRY(POINT, 4326) NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    
    -- Metadata
    file_size_bytes BIGINT,
    dimensions VARCHAR(20),                  -- "1920x1080"
    duration_sec INTEGER,                    -- for video
    device_id VARCHAR(100),
    
    -- Anti-tampering
    file_hash_sha256 VARCHAR(64) NOT NULL,
    exif_data JSONB,
    
    -- Auto-computed QA
    within_parcel_boundary BOOLEAN,          -- ST_Contains(parcel.boundary, location)
    
    uploaded_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_media_job ON survey_media(job_id);


-- ═══════════════════════════════════════
-- AGENT PAYOUTS
-- ═══════════════════════════════════════

CREATE TABLE agent_payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id),
    
    payout_period_start DATE,
    payout_period_end DATE,
    
    total_jobs INTEGER,
    gross_amount NUMERIC(10,2),
    platform_commission NUMERIC(10,2),
    tds_deducted NUMERIC(10,2),              -- TDS for gig workers
    net_amount NUMERIC(10,2),
    
    status VARCHAR(20) DEFAULT 'pending',
        -- pending | processing | completed | failed
    razorpay_payout_id VARCHAR(100),
    razorpay_status VARCHAR(30),
    
    initiated_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);
```

## 4.2 Retained Tables from v1 (Unchanged)

- **`parcels`** — Land parcel registration, boundary polygons, metadata
- **`users`** — Landowner accounts, subscriptions, preferences
- **`alerts`** — Notifications, now triggered by survey findings instead of satellite detection
- **`legal_records`** — Legal intelligence from web scraping (retained as optional premium feature)
- **`risk_scores`** — Now computed from survey findings + legal data instead of satellite + ML

## 4.3 Removed Tables from v1

- **`satellite_observations`** — No longer needed (no satellite imagery pipeline)
- **`change_detections`** — Replaced by `survey_responses` + `survey_media`

---

# 5. Feature-by-Feature Implementation

## 5.1 Landowner Parcel Registration (Updated)

### User Flow

1. Landowner creates account (email/phone + OTP via MSG91)
2. Selects subscription plan (determines visit frequency)
3. Registers parcel: enters state, district, survey number; draws/uploads boundary on Mapbox map
4. Uploads title deed (optional, for legal verification add-on)
5. System validates boundary polygon, geocodes address, stores in PostGIS
6. **FIRST VISIT** is scheduled immediately (within 48-72 hours) — this creates the baseline survey
7. Subsequent visits auto-scheduled based on plan frequency

### Subscription Plans (Revised for Agent Model)

| Feature | Basic (₹1,499/mo) | Pro (₹3,999/mo) | Premium (₹11,999/mo) |
|---------|-------------------|-----------------|---------------------|
| Visit Frequency | Once per quarter | Once per month | Twice per month |
| Survey Type | Basic Site Check | Detailed Survey | Premium Inspection |
| Photos per Visit | 8-12 geotagged | 15-20 + video | 25+ photos + video + drone* |
| Boundary Walk | GPS trace included | GPS trace included | GPS trace + area measurement |
| Report Type | Basic PDF summary | Detailed with annotations | Comprehensive + legal-ready |
| Alerts | Email on completion | Email + SMS | Email + SMS + WhatsApp + real-time |
| Legal Monitoring | Not included | Court cases + mutations | Full legal intelligence |
| On-Demand Visit | ₹500/extra visit | 1 free/month, then ₹400 | 3 free/month, then ₹300 |
| Agent Tier | Any available | Experienced+ only | Senior+ only |
| Support | Email (48hr) | Priority email (24hr) | Dedicated account manager |

*Drone where available (Phase 3)

### Implementation Details

- **API Endpoint:** `POST /v1/parcels`
- **Validation:** Shapely `is_valid` check, minimum area > 50 sqm, `ST_IsValid(boundary)` in PostGIS
- **Boundary Resolution:** If user provides only survey number + state → query SVAMITVA/DILRMP database. If unavailable → user draws manually on Mapbox
- **Title Deed OCR:** AWS Textract processes uploaded documents, extracts owner name + survey number for cross-validation
- **First Survey Trigger:** On parcel registration → immediately create `SurveyJob` with `priority = 'high'`, deadline = 72 hours

## 5.2 Automatic Job Scheduling

### Frequency Mapping

| Plan | Interval | Grace Period | Deadline |
|------|----------|-------------|----------|
| Basic | 90 days | 7 days | next_due + 7 days |
| Pro | 30 days | 3 days | next_due + 3 days |
| Premium | 15 days | 2 days | next_due + 2 days |

### On-Demand Visits

Landowner can request extra visits anytime from dashboard → creates immediate `SurveyJob` with `priority = 'high'`, deadline = 48 hours.

## 5.3 Job Allocation Engine

### State Machine

```
pending_assignment ──[match found]──→ offered
       │                                 │
       │                          ┌──────┴──────┐
       │                    [accepted]      [declined/expired]
       │                          │               │
       │                      assigned        [cascade next]──→ offered (next agent)
       │                          │               │
       │                   agent_en_route    [all exhausted]
       │                          │               │
       │                    agent_on_site      unassigned
       │                          │
       │                     in_progress
       │                          │
       │                      submitted
       │                       │      │
       │                 [QA pass]  [QA fail]
       │                     │          │
       │                 completed   failed_qa → (re-schedule)
       │
   cancelled ←── [admin cancel]
```

### Offer Timeout Flow

```
Offer sent to Agent #1 (rank 1, score 0.92)
    ├── Agent accepts within 30 min → ASSIGNED ✓
    ├── Agent declines → Offer sent to Agent #2 (rank 2, score 0.87)
    │       ├── Agent accepts → ASSIGNED ✓
    │       ├── Agent declines → Offer sent to Agent #3 ...
    │       └── 30 min timeout → Offer sent to Agent #3 ...
    └── 30 min timeout → Offer sent to Agent #2 ...

After 5 agents in round 1 → Round 2 (expanded radius: 50km)
After 5 agents in round 2 → Round 3 (expanded radius: 100km)
After round 3 exhausted → Mark UNASSIGNED, notify ops team
```

## 5.4 Survey Execution Workflow (Agent App)

### Basic Site Check Template

| Step | Type | Agent Action | Required | Anti-Fraud Measure |
|------|------|-------------|----------|-------------------|
| 1. Arrive at Site | Geofence check | App auto-detects arrival when GPS within 100m of parcel centroid | Yes | Cannot proceed until GPS confirms. Spoofing detection via accelerometer + cell tower. |
| 2. Panoramic Photos | Photo capture | Take 4 photos from each boundary edge (N, S, E, W facing inward) | Yes (4 min) | In-app camera only. EXIF GPS embedded. Each photo must be within parcel boundary. |
| 3. Boundary Markers | Photo + checklist | Photograph each marker/stone. Answer: All intact? Any shifted? Any missing? | Yes | Photo GPS must be near parcel boundary edges. |
| 4. Encroachment Check | Checklist | Any new structures? Unauthorized fencing? Construction material? Unauthorized farming? | Yes | If "Yes" to any → mandatory photo of the issue. |
| 5. Boundary Walk | GPS trace | Walk entire parcel boundary with app recording GPS trail | Yes | GPS breadcrumbs every 2s. Trail must match boundary polygon (Hausdorff distance check). |
| 6. General Condition | Checklist | Land condition, access road, notice boards, neighboring activity | Yes | Free-text notes for additional observations. |
| 7. Video Walkthrough | Video capture | 30-second video walking across the parcel | Yes (30s min) | In-app recording. Video GPS trail during recording. |
| 8. Submit | Submission | Review all responses, add notes, submit | Yes | Cannot submit if incomplete. SHA-256 hash of all media for tamper detection. |

### Detailed Survey (Pro Tier — Additional Steps)

All Basic steps, plus:

- **Area Measurement:** GPS-based area calculation vs registered area. Flag if > 5% discrepancy.
- **Neighbor Interview:** Notes on conversations with adjacent landowners/occupants.
- **Infrastructure Check:** Road development, drainage, electricity poles, water supply changes.
- **Encroachment Detail:** Measure approximate encroachment area. Photos from multiple angles.
- 15-20 photos minimum, 60-second video minimum.

### Premium Inspection (Premium Tier — Additional Steps)

All Detailed steps, plus:

- **Soil/Land Use Assessment:** Current condition, earth-moving evidence, water logging.
- **Legal Notice Check:** Physical inspection for notices, government markers, acquisition signs.
- **Photo Documentation:** 25+ photos covering every aspect.
- 90-second video with narration describing observations.
- Drone survey where available (Phase 3).

### Offline Mode

Many rural parcels have poor connectivity. The agent app works **fully offline** during survey:

- Photos/videos stored locally on device
- Checklist responses saved to local SQLite database
- GPS traces recorded locally
- On connectivity restore → background sync queue uploads all data
- Chunked multipart upload to S3 with resume support
- App shows sync status; nothing is lost

## 5.5 Quality Assurance System

### Automated QA Checks (Every Submission)

| Check | What It Verifies | Failure Action |
|-------|-----------------|----------------|
| **Geolocation Verification** | All photos/videos captured within parcel boundary? `ST_Contains(parcel.boundary, media.location)` | Flag photos outside boundary. If > 50% outside → reject, require re-survey. |
| **Completeness Check** | All mandatory steps completed? All required photos uploaded? | Cannot submit if incomplete (in-app). Server double-check on receipt. |
| **Boundary Walk Validation** | GPS trace matches parcel boundary? Hausdorff distance < 50m threshold. | Mismatch → flag for manual review. Could indicate wrong site. |
| **Timestamp Consistency** | All media within reasonable time window? Timestamps sequential? | Gaps > 2 hours → flag. Future/far-past timestamps → reject. |
| **Device Consistency** | All media from same device? Device ID matches agent's registration? | Mixed devices → flag for review (possible delegation). |
| **Duplicate Detection** | pHash comparison against agent's previous survey photos for same parcel. | > 80% similarity → reject. Agent may be reusing old photos. |
| **Photo Quality** | Not blurry (Laplacian variance), not too dark/bright (histogram). | > 50% photos fail quality → flag for review. |

### Human QA Review (20% Random Sample + All Flagged)

QA team member reviews in Admin Dashboard:

- All photos on map overlay
- GPS trace vs boundary comparison
- Checklist responses
- Video playback
- Assign QA score (1-5), approve / request re-survey / escalate

## 5.6 Report Generation & Delivery

### What the Landowner Receives

After each completed survey, landowner receives a **structured PDF report** within 24 hours (generated via WeasyPrint).

| Report Section | Contents | Data Source |
|---------------|----------|-------------|
| Cover Page | Parcel ID, survey date, agent (anonymized), survey type, status (Clear / Issues Found) | `survey_jobs` |
| Parcel Overview Map | Mapbox static image: boundary, GPS trace, photo capture points | PostGIS + GPS trace + media locations |
| Photo Evidence Grid | All geotagged photos by step, with captions and timestamps | `survey_media`, S3 pre-signed URLs |
| Boundary Walk Analysis | GPS trace overlaid on boundary. Area comparison. Discrepancy highlighted. | `survey_responses` GPS trace |
| Encroachment Findings | Each "Yes" on encroachment items: description, photos, map location, severity | Checklist responses |
| Comparative Analysis | Side-by-side with previous survey. What changed since last visit. | Previous `survey_responses` |
| Risk Summary | Updated Land Risk Score with factor breakdown | `risk_scores` |
| Agent Notes | Free-text observations | `survey_responses` notes |
| Verification Seal | Digital signature, unique report ID, QR code for online verification | Report service |

## 5.7 Risk Scoring Engine (Revised)

Input sources changed from satellite imagery to structured survey data:

| Risk Factor | Weight | Data Source | Scoring Logic |
|------------|--------|-------------|---------------|
| Encroachment Findings | 35% | Survey checklist responses | Each "yes" on encroachment items adds 10-25 points. Multiple issues compound. |
| Boundary Integrity | 20% | GPS trace vs boundary, marker status | Hausdorff distance > 20m = +15. Missing markers = +10 each. Area > 10% off = +20. |
| Legal Risk | 25% | Legal engine (courts, mutations, gazette) | Active litigation = +30. Unknown mutation = +25. Acquisition notice = +40. |
| Environmental | 10% | Survey observations + gov GIS data | Flood zone = +15. Soil disturbance = +10. |
| Neighborhood | 10% | Nearby parcels + agent observations | High encroachment density nearby = +10. Adjacent construction = +5. |

**Computation:** Hybrid rule-based (60%) + XGBoost model (40%). SHAP values for explainability. Retrained monthly.

## 5.8 Landowner Dashboard

| Section | What the User Sees | Key Interactions |
|---------|-------------------|------------------|
| **Map View** | All parcels on Mapbox, color-coded by risk. Click to expand. | Click parcel, zoom, toggle layers |
| **Parcel Detail** | Latest survey summary, risk score gauge, next visit date, alerts | Download report, request on-demand visit, share access |
| **Survey Timeline** | All completed surveys chronologically. Side-by-side comparison tool. | Select two surveys to compare |
| **Live Tracking** | Agent en-route/on-site: real-time location on map, status updates | View progress, see ETA (cannot contact agent directly) |
| **Alerts Feed** | Survey completed, issue found, legal record, risk score change | Mark read, click details, manage preferences |
| **Reports Library** | All PDFs, downloadable and shareable via unique link | Download, share, re-generate |
| **Subscription** | Plan, billing history, upgrade/downgrade, add-ons | Change plan, update payment, manage parcels |

---

# 6. Infrastructure as Code (Revised)

## 6.1 What Changed from v1 IaC

| Component | v1 (Satellite) | v2 (Agent Model) | Cost Impact |
|-----------|----------------|-------------------|-------------|
| GPU Nodes (EKS) | Required for ML (g4dn.xlarge) | **REMOVED** | Saves ~$375/mo |
| Satellite API Costs | SentinelHub + Planet Labs | **REMOVED** | Saves $200-2000/mo |
| Airflow (Heavy) | Complex DAGs for imagery | Simplified: scheduling + legal ETL | Smaller instance |
| S3 Storage | Massive (GeoTIFF, COG) | Moderate (JPEG, MP4) | 70% less storage |
| Firebase (NEW) | Not needed | FCM for push notifications | Free tier |
| WebSocket (NEW) | Not needed | Real-time agent tracking | Small compute add |
| Mobile Backend (NEW) | Web-only | Agent + landowner app APIs | Slightly more load |

## 6.2 Repository Structure

```
infra/
├── terragrunt.hcl                      # Root config
├── environments/
│   ├── dev/
│   │   ├── env.hcl                     # Dev variables
│   │   ├── foundation/                 # VPC, subnets, NAT, IGW
│   │   │   └── terragrunt.hcl
│   │   ├── eks/                        # EKS cluster (NO GPU nodes)
│   │   │   └── terragrunt.hcl
│   │   ├── data/                       # RDS PostGIS, ElastiCache, S3
│   │   │   └── terragrunt.hcl
│   │   ├── messaging/                  # AmazonMQ (RabbitMQ)
│   │   │   └── terragrunt.hcl
│   │   ├── monitoring/                 # Prometheus, Grafana
│   │   │   └── terragrunt.hcl
│   │   └── services/                   # Helm charts
│   │       └── terragrunt.hcl
│   ├── staging/                        # Same structure
│   └── prod/                           # Same structure
└── modules/
    ├── vpc/
    ├── eks/                            # REVISED: no GPU node group
    ├── rds-postgis/
    ├── s3-media/                       # REVISED: photos/videos (not imagery)
    ├── elasticache-redis/
    ├── amazonmq-rabbitmq/
    ├── keycloak/
    ├── kong-gateway/
    ├── firebase-config/                # NEW
    └── socketio-server/                # NEW
```

## 6.3 Key Terraform Modules

### 6.3.1 VPC Module

```hcl
# modules/vpc/main.tf
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${var.project}-${var.environment}"
  cidr = var.vpc_cidr  # "10.0.0.0/16"

  azs              = ["ap-south-1a", "ap-south-1b", "ap-south-1c"]
  private_subnets  = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets   = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]
  database_subnets = ["10.0.201.0/24", "10.0.202.0/24", "10.0.203.0/24"]

  enable_nat_gateway   = true
  single_nat_gateway   = var.environment != "prod"
  enable_dns_hostnames = true
  enable_dns_support   = true

  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }
  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
  }
}
```

### 6.3.2 EKS Cluster (No GPU)

```hcl
# modules/eks/main.tf — REVISED: no GPU node group
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = "${var.project}-${var.environment}"
  cluster_version = "1.29"
  vpc_id          = var.vpc_id
  subnet_ids      = var.private_subnet_ids

  cluster_endpoint_public_access = true
  cluster_addons = {
    coredns    = { most_recent = true }
    kube-proxy = { most_recent = true }
    vpc-cni    = { most_recent = true }
  }

  eks_managed_node_groups = {
    # General workloads (API services, job engine, websockets)
    general = {
      instance_types = ["m6i.large"]           # Downsized from xlarge
      min_size       = var.environment == "prod" ? 2 : 1
      max_size       = var.environment == "prod" ? 8 : 3
      desired_size   = var.environment == "prod" ? 3 : 1
      labels         = { workload = "general" }
    }

    # Background workers (Celery, scraping, report generation)
    workers = {
      instance_types = ["c6i.large"]           # Compute-optimized
      min_size       = var.environment == "prod" ? 1 : 0
      max_size       = var.environment == "prod" ? 5 : 2
      desired_size   = var.environment == "prod" ? 2 : 1
      labels         = { workload = "workers" }
    }

    # ✅ NO GPU NODE GROUP — key cost saving vs v1
  }
}
```

### 6.3.3 RDS PostGIS

```hcl
# modules/rds-postgis/main.tf
module "rds" {
  source  = "terraform-aws-modules/rds/aws"
  version = "~> 6.0"

  identifier           = "${var.project}-postgis-${var.environment}"
  engine               = "postgres"
  engine_version       = "16.3"
  family               = "postgres16"
  major_engine_version = "16"
  instance_class       = var.environment == "prod" ? "db.r6g.xlarge" : "db.t4g.medium"

  allocated_storage     = 100
  max_allocated_storage = 1000

  db_name  = "landintel"
  username = "landintel_admin"
  port     = 5432

  parameters = [
    { name = "shared_preload_libraries", value = "pg_stat_statements" },
    { name = "rds.allowed_extensions",
      value = "postgis,postgis_topology,pg_trgm,uuid-ossp" }
  ]

  multi_az               = var.environment == "prod"
  db_subnet_group_name   = var.db_subnet_group_name
  vpc_security_group_ids = [var.db_security_group_id]

  backup_retention_period = var.environment == "prod" ? 30 : 7
  deletion_protection     = var.environment == "prod"
  performance_insights_enabled = true
}
```

### 6.3.4 S3 Media Storage (Revised)

```hcl
# modules/s3-media/main.tf — photos/videos instead of satellite imagery
resource "aws_s3_bucket" "survey_media" {
  bucket = "${var.project}-survey-media-${var.environment}"
}

resource "aws_s3_bucket_lifecycle_configuration" "media_lifecycle" {
  bucket = aws_s3_bucket.survey_media.id

  rule {
    id     = "survey-photos"
    status = "Enabled"
    filter { prefix = "surveys/" }
    transition {
      days          = 90
      storage_class = "STANDARD_IA"
    }
    transition {
      days          = 365
      storage_class = "GLACIER_IR"
    }
  }

  rule {
    id     = "reports"
    status = "Enabled"
    filter { prefix = "reports/" }
    transition {
      days          = 180
      storage_class = "STANDARD_IA"
    }
  }

  rule {
    id     = "documents"
    status = "Enabled"
    filter { prefix = "documents/" }
    transition {
      days          = 365
      storage_class = "STANDARD_IA"
    }
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "enc" {
  bucket = aws_s3_bucket.survey_media.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = var.kms_key_arn
    }
  }
}

# S3 key structure:
# surveys/{job_id}/photos/{step_id}_{sequence}.jpg
# surveys/{job_id}/videos/{step_id}.mp4
# surveys/{job_id}/gps_trace.geojson
# reports/{parcel_id}/{date}_report.pdf
# documents/{parcel_id}/title_deed.pdf
```

### 6.3.5 Redis Cache

```hcl
# modules/elasticache-redis/main.tf
resource "aws_elasticache_replication_group" "redis" {
  replication_group_id       = "${var.project}-redis-${var.environment}"
  description                = "LandIntel Redis — agent locations, sessions, dedup"
  node_type                  = var.environment == "prod" ? "cache.r6g.large" : "cache.t4g.medium"
  num_cache_clusters         = var.environment == "prod" ? 3 : 1
  port                       = 6379
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  subnet_group_name          = var.redis_subnet_group_name
  security_group_ids         = [var.redis_security_group_id]
}
```

### 6.3.6 CI/CD Pipeline

```yaml
# .github/workflows/deploy.yml
name: Deploy LandIntel
on:
  push:
    branches: [main, staging]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgis/postgis:16-3.4
        env:
          POSTGRES_DB: test_landintel
          POSTGRES_PASSWORD: test
        ports: ["5432:5432"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: "3.12" }
      - run: pip install -r requirements-test.txt
      - run: pytest tests/ --cov=app --cov-report=xml
      - uses: codecov/codecov-action@v4

  build:
    needs: test
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service:
          - land-service
          - job-allocation-engine
          - agent-service
          - alert-service
          - report-service
          - billing-service
          - legal-intelligence
          - api-gateway
    steps:
      - uses: actions/checkout@v4
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: ap-south-1
      - uses: aws-actions/amazon-ecr-login@v2
      - run: |
          docker build -t ${{ matrix.service }} \
            -f services/${{ matrix.service }}/Dockerfile .
          docker tag ${{ matrix.service }} \
            $ECR_REGISTRY/${{ matrix.service }}:$GITHUB_SHA
          docker push $ECR_REGISTRY/${{ matrix.service }}:$GITHUB_SHA

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: |
          aws eks update-kubeconfig --name landintel-$ENVIRONMENT
          helm upgrade --install landintel ./helm/landintel \
            --set global.imageTag=$GITHUB_SHA \
            --values ./helm/values-$ENVIRONMENT.yaml \
            --wait --timeout 10m
```

## 6.4 Infrastructure Cost Estimation

| Resource | Dev/Staging | Production | vs v1 Change |
|----------|-------------|------------|-------------|
| EKS Cluster | $73/mo | $73/mo | Same |
| EC2 General Nodes | $100/mo (1x m6i.large) | $300/mo (3x m6i.large) | ⬇ from $450 |
| EC2 Worker Nodes | $0-60/mo | $120-250/mo (2x c6i.large) | Replaces GPU nodes |
| **GPU Nodes** | **ELIMINATED** | **ELIMINATED** | **SAVES $375/mo** |
| RDS PostGIS | $105/mo | $520/mo | Same |
| S3 Media Storage | $10/mo | $50-150/mo | ⬇ from $200-500 |
| ElastiCache Redis | $65/mo | $375/mo | Same |
| AmazonMQ RabbitMQ | $50/mo | $130/mo | Same |
| Firebase (FCM) | $0 | $0 | NEW but free |
| **Satellite API Fees** | **ELIMINATED** | **ELIMINATED** | **SAVES $200-2000/mo** |
| ALB + Kong | $25/mo | $50/mo | Same |
| CloudWatch | $20/mo | $75/mo | Same |
| NAT Gateway | $35/mo | $100/mo | Same |
| **TOTAL** | **~$485/mo** | **~$1,800-2,000/mo** | **SAVES 30-40% vs v1** |

> **KEY INSIGHT:** The agent model trades fixed infrastructure costs (GPUs, satellite APIs) for variable labor costs (agent payouts). Lower infra spend, but the real cost is agent payouts — which scale linearly with revenue.

---

# 7. Development Phases (Revised Roadmap)

## 7.1 Phase 1: Foundation + Agent MVP (Months 1-3)

> **GOAL:** Working platform where a landowner registers a parcel → survey job created → agent assigned via matching algorithm → agent completes survey via mobile app → landowner receives PDF report. All infrastructure deployed via Terraform.

### Month 1: Infrastructure + Core Backend

| Week | Tasks | Deliverables | Tools |
|------|-------|-------------|-------|
| 1-2 | Terraform: VPC, EKS (no GPU), RDS PostGIS, S3, Redis, RabbitMQ. CI/CD pipeline. Docker base images. Keycloak + Kong setup. | Dev environment fully deployed via IaC. Auth working. API Gateway routing. CI/CD builds + deploys. | Terraform, Terragrunt, AWS, Docker, GitHub Actions, Keycloak, Kong |
| 3-4 | Land Service (parcel CRUD, PostGIS). User Service (registration, subscription). Agent Service (registration, profile, KYC placeholder). All DB migrations. | Parcels created with boundary. Users registered. Agents onboarded. All via API. | FastAPI, SQLAlchemy, GeoAlchemy2, Alembic, PostGIS |

### Month 2: Job Engine + Agent Mobile App

| Week | Tasks | Deliverables | Tools |
|------|-------|-------------|-------|
| 5-6 | Job Allocation Engine: matching algorithm, offer dispatch, cascade, state machine. Scheduler (Celery Beat). Push notifications (FCM). Agent location tracking (Redis). | Jobs auto-created. Matching finds nearest agent. Offers sent via push. Cascade on timeout. Full state machine. | Celery, RabbitMQ, Firebase Admin SDK, Redis, PostGIS |
| 7-8 | Agent Mobile App (React Native): login, job feed, accept/decline, navigation, survey workflow (basic template), in-app camera + geotagging, GPS trace, offline storage, submission. | Agent can: login → see offers → accept → navigate → complete survey → take photos → record GPS → submit. Works offline. | React Native, Expo Camera, expo-location, SQLite, S3 upload |

### Month 3: Landowner Dashboard + QA + Reports

| Week | Tasks | Deliverables | Tools |
|------|-------|-------------|-------|
| 9-10 | Landowner Dashboard (React): map view, parcel detail, live agent tracking (WebSocket), survey timeline, alerts. Razorpay subscription integration. | Landowner can: register parcels → see agent status → view surveys → download reports → manage subscription → pay. | React, Mapbox GL JS, Socket.IO, TailwindCSS, Razorpay |
| 11-12 | Automated QA (geofence, completeness, timestamp, duplicate). Report generation (WeasyPrint). Admin dashboard for manual QA. SMS/email alerts. E2E testing. | Full loop working. Admin reviews flagged surveys. Ready for alpha testing. | WeasyPrint, imagehash, SendGrid, MSG91, pytest, Locust |

**Phase 1 Team:** 2 Backend, 1 React Native Mobile, 1 Frontend (web), 1 DevOps. **Total: 5.** Burn: ₹12-15L/month.

**Exit Criteria:** 10 parcels registered, 5 test agents completing surveys, end-to-end flow < 5% failure rate, report delivered within 48 hours.

## 7.2 Phase 2: Scale Agent Network + Legal Engine (Months 4-6)

> **GOAL:** 100 paid landowners, 50 active agents in 2 cities (Bangalore + Hyderabad). Legal intelligence operational. Agent payouts live. QA hardened.

| Month | Focus | Key Deliverables | Tools |
|-------|-------|-----------------|-------|
| 4 | Agent Recruitment at Scale | In-app training (video + quiz). Aadhaar eKYC (DigiLocker). Agent tiers. Referral program. Target: 30 agents in Bangalore. | DigiLocker API, video player, quiz engine |
| 5 | Legal Intelligence + WhatsApp | Karnataka crawlers (Kaveri, eCourts, Gazette). Entity matching. Legal dashboard. WhatsApp alerts. Agent payout system (Razorpay Payouts). | Scrapy, Playwright, Gupshup, Razorpay Payouts, spaCy |
| 6 | Beta Launch | Risk scoring v1 (survey-driven). Comparative analysis. Landowner mobile app. Performance dashboard. Beta: Bangalore + Hyderabad. Target: 100 users. | XGBoost, SHAP, React Native, Intercom, Mixpanel |

**Phase 2 Team (adds):** +1 Backend, +1 Data Engineer, +1 Agent Operations Manager. **Total: 8.** Burn: ₹20-25L/month.

**Exit Criteria:** ML risk model operational, legal crawlers 95%+ uptime, 100 beta users, churn < 10%, NPS > 40.

## 7.3 Phase 3: Multi-City + Enterprise (Months 7-12)

> **GOAL:** 8 cities, 500+ paid landowners, 200+ active agents, first enterprise client, ₹5L+ MRR.

| Month | Focus | Key Deliverables |
|-------|-------|-----------------|
| 7-8 | Multi-city expansion | 6 more cities (Chennai, Mumbai, Pune, Delhi NCR, Kolkata, Ahmedabad). 6-state legal crawlers. Regional languages (Hindi, Tamil, Telugu, Marathi). Agent recruitment partnerships. |
| 8-9 | Enterprise features | Enterprise API (batch verification, bulk surveys). White-label reports. SLA guarantees (48hr survey, 99.5% completion). Custom checklist templates. |
| 9-10 | Pre-Purchase Report Product | One-time purchase flow (no subscription). Comprehensive due diligence. ₹15K-50K pricing. Marketplace. Partnership with 99acres/MagicBricks. |
| 10-11 | Advanced features | Drone integration (Premium). Historical visual diff. Route optimization for agents. Automated fraud pattern detection. |
| 11-12 | Growth + Fundraising | Performance optimization. Cost optimization (reserved instances). Metrics package. Data room. Government partnership exploration. |

**Phase 3 Team (adds):** +2 Backend, +1 Mobile, +1 Sales/BD, +1 Agent Ops, +1 Customer Success. **Total: ~14.** Burn: ₹35-45L/month.

---

# 8. Complete Third-Party Integration List

| Category | Service | Purpose | New/Retained | Cost |
|----------|---------|---------|-------------|------|
| Maps | Mapbox GL JS | Dashboard map, boundary visualization | Retained | Free 50K loads/mo |
| Maps | Google Maps Platform | Geocoding, agent navigation deep links | Retained | $200 free/mo |
| Database | PostgreSQL + PostGIS | Geospatial DB, spatial queries | Retained | Open source |
| Push Notifications | Firebase Cloud Messaging | Agent push notifications | **NEW** | Free |
| Real-time | Socket.IO (self-hosted) | Live agent tracking, job updates | **NEW** | Open source |
| KYC | DigiLocker / Aadhaar eKYC | Agent identity verification | **NEW** | Gov API (free) |
| KYC | Penny Drop (Razorpay) | Agent bank account verification | **NEW** | ₹2/verification |
| Auth | Keycloak | Identity management, RBAC | Retained | Open source |
| Gateway | Kong | API routing, rate limiting | Retained | Open source |
| Email | SendGrid | Emails, report delivery | Retained | Free 100/day |
| SMS | MSG91 | OTP, survey alerts | Retained | Per SMS |
| WhatsApp | Gupshup | WhatsApp notifications | Retained | Per message |
| Payments (Collection) | Razorpay | Subscription billing | Retained | 2% per txn |
| Payments (Payout) | Razorpay Payouts / RazorpayX | Agent disbursement (UPI/NEFT) | **NEW** | ₹5-10/payout |
| Payments (Intl) | Stripe | NRI payments | Retained | 2.9% + 30c |
| PDF | WeasyPrint | Report generation | Retained | Open source |
| Search | Elasticsearch | Full-text search, logs | Retained | Open source |
| Queue | RabbitMQ | Event bus, job dispatch | Retained | Open source |
| Cache | Redis | Agent locations, sessions, dedup | Retained | Open source |
| Task Queue | Celery + Celery Beat | Background tasks, scheduling | Retained | Open source |
| Scraping | Scrapy + Playwright | Legal portal crawling | Retained | Open source |
| Proxy | BrightData | Residential proxies | Retained | Per GB |
| OCR | AWS Textract | Title deed processing | Retained | Per page |
| Image Hashing | imagehash (Python) | Duplicate photo detection | **NEW** | Open source |
| Monitoring | Prometheus + Grafana | Metrics, dashboards | Retained | Open source |
| Logging | EFK Stack | Centralized logging | Retained | Open source |
| CI/CD | GitHub Actions | Build, test, deploy | Retained | Free 2000 min/mo |
| IaC | Terraform + Terragrunt | Infrastructure as Code | Retained | Open source |
| Containers | Docker + AWS EKS | Orchestration | Retained | EKS $0.10/hr |
| Mobile | React Native + Expo | Agent + landowner apps | **NEW** | Open source |
| Crash Reporting | Firebase Crashlytics | Mobile crash reports | **NEW** | Free |
| Support | Intercom / Freshdesk | User + agent support | **NEW** | $0-74/mo |
| Analytics | Mixpanel / Google Analytics | Funnel analysis | **NEW** | Free tier |
| CDN | AWS CloudFront | Static assets, training videos | Retained | Per GB |

### Removed from v1

SentinelHub, Google Earth Engine, Planet Labs API, TorchServe, MLflow, Label Studio, PyTorch — all satellite/ML services eliminated.

---

# 9. Security & Anti-Fraud (Agent-Specific)

## 9.1 Agent Fraud Prevention

| Fraud Type | How Agent Might Cheat | Prevention Mechanism |
|-----------|----------------------|---------------------|
| **Fake Location (GPS Spoofing)** | Use GPS spoofing app to fake presence at parcel | 1) Check mock location providers (Android API). 2) Cross-validate with cell tower triangulation. 3) Accelerometer data (movement patterns). 4) WiFi scan fingerprinting. |
| **Photo Reuse** | Submit old photos or internet images | 1) pHash against previous survey photos. 2) EXIF timestamp validation. 3) In-app camera only (no gallery). 4) Watermark with job_id in metadata. |
| **Delegation** | Give phone to unauthorized person | 1) Periodic selfie verification (face match vs registration). 2) Device ID consistency. 3) Behavioral analysis (typing, speed). |
| **Rushed Survey** | Complete checklist without inspecting | 1) Minimum time-on-site (15 min for basic). 2) GPS trace must cover parcel area. 3) Photo timestamps span reasonable duration. 4) Minimum video duration. |
| **Collusion** | Agent + encroacher collude to hide issues | 1) Agent rotation (can't survey same parcel twice consecutively). 2) 20% random human QA. 3) Comparative analysis flags inconsistencies. |

## 9.2 Data Security (Retained from v1)

- Encryption at rest (AWS KMS), in transit (TLS 1.3)
- Row-Level Security in PostGIS
- Audit logging (all access tracked)
- AWS WAF + Shield
- Secrets Manager for all credentials

**Additional for agent model:**

- Agent PII (Aadhaar, bank) encrypted with per-agent keys
- Location history purged after 30 days
- Survey photos owned by platform + landowner, not agent
- Agent app uses certificate pinning

## 9.3 Compliance

- India IT Act 2000 (Section 43A)
- Geospatial Data Guidelines 2021 (liberalized, no prior approval)
- Personal Data Protection Bill (consent, data localization in ap-south-1)
- Gig Worker Regulations (proper TDS, no employer classification)

---

# 10. Unit Economics: Agent Model vs Satellite Model

| Metric | Satellite (v1) | Agent (v2) | Notes |
|--------|---------------|------------|-------|
| Cost per check (Basic) | ~₹50-100 (API + compute) | ~₹250-300 (payout + ops) | Agent costs more per check |
| Revenue per parcel/mo (Basic) | ₹999/mo | ₹1,499/mo | Higher pricing justified |
| Gross margin (Basic) | ~85-90% | ~60-70% | Lower but healthy |
| Evidence quality | Satellite overlay (low legal value) | Geotagged photos + GPS (court-admissible) | Dramatically better |
| False positive rate | 15-30% | < 5% | Human verification wins |
| Detection capability | Limited by 10m resolution | Anything visible to human eye | Sheds, fencing, marker removal |
| Customer willingness to pay | Moderate (abstract data) | High (tangible verification) | Users trust human reports |
| Scalability bottleneck | Compute + GPU | Agent recruitment + quality | Different challenges |
| Break-even per parcel | ~10 parcels covers infra | ~5 parcels per cycle | Both viable at small scale |

> **BOTTOM LINE:** Lower gross margins but dramatically higher value. Court-admissible evidence, 95%+ accuracy, detection of anything visible. At scale (500+ parcels), agent costs become predictable via route optimization, batching nearby parcels, and volume-based payout negotiations.

---

# 11. API Endpoint Reference

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/v1/auth/register` | User registration | No |
| POST | `/v1/auth/login` | Login, returns JWT | No |
| POST | `/v1/parcels` | Register new parcel | Yes |
| GET | `/v1/parcels` | List user's parcels | Yes |
| GET | `/v1/parcels/{id}` | Parcel details | Yes (owner) |
| PUT | `/v1/parcels/{id}/boundary` | Update boundary | Yes (owner) |
| POST | `/v1/parcels/{id}/request-visit` | Request on-demand visit | Yes (owner) |
| GET | `/v1/parcels/{id}/surveys` | List completed surveys | Yes (owner) |
| GET | `/v1/parcels/{id}/legal` | Legal records | Yes (Pro+) |
| GET | `/v1/parcels/{id}/risk-score` | Current risk score | Yes (Pro+) |
| POST | `/v1/parcels/{id}/reports` | Generate report | Yes (Pro+) |
| GET | `/v1/alerts` | List alerts | Yes |
| PUT | `/v1/alerts/{id}/read` | Mark as read | Yes |
| PUT | `/v1/alerts/preferences` | Update preferences | Yes |
| POST | `/v1/agents/register` | Agent registration | No |
| POST | `/v1/agents/me/location` | Update agent location | Yes (agent) |
| PUT | `/v1/agents/me/availability` | Toggle online/offline | Yes (agent) |
| GET | `/v1/agents/me/jobs` | Active + available jobs | Yes (agent) |
| POST | `/v1/agents/me/jobs/{id}/accept` | Accept job offer | Yes (agent) |
| POST | `/v1/agents/me/jobs/{id}/decline` | Decline job offer | Yes (agent) |
| POST | `/v1/agents/me/jobs/{id}/arrive` | Mark arrival (geofence) | Yes (agent) |
| POST | `/v1/agents/me/jobs/{id}/start` | Start survey | Yes (agent) |
| POST | `/v1/agents/me/jobs/{id}/submit` | Submit survey | Yes (agent) |
| POST | `/v1/agents/me/jobs/{id}/media` | Upload photo/video | Yes (agent) |
| GET | `/v1/agents/me/earnings` | Earnings dashboard | Yes (agent) |
| GET | `/v1/agents/me/training` | Training modules | Yes (agent) |
| POST | `/v1/enterprise/batch-verify` | Batch verification | API key |
| GET | `/v1/enterprise/portfolio` | Portfolio risk summary | API key |
| GET | `/v1/admin/jobs/unassigned` | Unassigned jobs | Admin |
| POST | `/v1/admin/jobs/{id}/assign` | Manual assignment | Admin |
| GET | `/v1/admin/qa/pending` | Surveys pending QA | Admin |
| POST | `/v1/admin/qa/{id}/review` | Submit QA review | Admin |

---

# 12. Environment Variables Reference

```bash
# ── Database ──
DATABASE_URL=postgresql+asyncpg://user:pass@host:5432/landintel
POSTGIS_ENABLED=true

# ── Redis ──
REDIS_URL=redis://host:6379/0

# ── RabbitMQ ──
RABBITMQ_URL=amqp://user:pass@host:5672/landintel

# ── AWS ──
AWS_REGION=ap-south-1
S3_MEDIA_BUCKET=landintel-survey-media-prod
S3_REPORTS_BUCKET=landintel-reports-prod
KMS_KEY_ARN=arn:aws:kms:ap-south-1:xxx:key/xxx

# ── Auth ──
KEYCLOAK_URL=https://auth.landintel.in
KEYCLOAK_REALM=landintel
JWT_SECRET_KEY=xxx

# ── Firebase (Push Notifications) ──
FIREBASE_PROJECT_ID=landintel-prod
FIREBASE_SERVER_KEY=xxx
FIREBASE_CREDENTIALS_JSON=base64_encoded

# ── Notifications ──
SENDGRID_API_KEY=xxx
MSG91_AUTH_KEY=xxx
GUPSHUP_API_KEY=xxx
GUPSHUP_APP_NAME=LandIntel

# ── Payments ──
RAZORPAY_KEY_ID=xxx
RAZORPAY_KEY_SECRET=xxx
RAZORPAY_PAYOUT_ACCOUNT_NUMBER=xxx
STRIPE_SECRET_KEY=xxx

# ── KYC ──
DIGILOCKER_CLIENT_ID=xxx
DIGILOCKER_CLIENT_SECRET=xxx

# ── Scraping ──
BRIGHTDATA_PROXY_URL=http://proxy:port
SCRAPER_CONCURRENCY=5
SCRAPER_RETRY_MAX=3

# ── Job Allocation Engine ──
JOB_MATCHING_DEFAULT_RADIUS_KM=25
JOB_MATCHING_MAX_RADIUS_KM=100
JOB_OFFER_TIMEOUT_MINUTES=30
JOB_MAX_CASCADE_ROUNDS=3
JOB_AGENTS_PER_ROUND=5

# ── Survey Config ──
GEOFENCE_RADIUS_METERS=100
MIN_SURVEY_DURATION_MINUTES=15
MIN_VIDEO_DURATION_SECONDS=30
QA_RANDOM_SAMPLE_PERCENT=20
DUPLICATE_PHOTO_THRESHOLD=0.80

# ── Agent Config ──
AGENT_LOCATION_UPDATE_INTERVAL_SEC=60
AGENT_MAX_CONCURRENT_JOBS=3
AGENT_LOCATION_HISTORY_RETENTION_DAYS=30
PAYOUT_BATCH_DAY=monday
```

---

**END OF DOCUMENT**

*LandIntel v2.0 | Rapido-Style Field Agent Model | Complete Development Plan*