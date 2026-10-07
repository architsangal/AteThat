# Phase 4 Backlog - Platform Capabilities

## Purpose

This document records capabilities intentionally excluded from Phase 1. It is a backlog and design guide, not a commitment or implementation plan. Phase 1 must not depend on any of these capabilities to work locally or to be deployed.

## Phase 1 Boundary

Phase 1 is a browser-based, multi-user nutrition and health tracking application. It includes the core user journeys defined in the PRD: authentication, profile and preferences, Tracking Day, nutrition targets, food search/review/confirmation, custom foods, meals, health items, schedules, dashboard, and history.

The Phase 1 backend consists only of the API Service and Core Application Service. The Core Service owns domain logic and MongoDB access. No database-wrapper microservice will be created.

Phase 1 may emit structured application logs for debugging and security monitoring, but it does not require a durable asynchronous worker, audit-event store, analytics pipeline, or message broker.

## Deferred Phase 4 Scope

### 1. Worker Service and Durable Background Jobs

Introduce a separately deployable Go Worker Service only when there is real asynchronous work that benefits from it. Candidate work includes:

- Durable audit-event processing.
- Analytics-event processing and aggregations.
- Notification generation and delivery.
- Email retries and dead-letter handling.
- Scheduled maintenance tasks.
- Long-running exports, AI processing, or provider synchronization.

The first implementation should use a MongoDB-backed outbox/jobs collection. Core writes the business record and its outbox event together; the worker claims, processes, retries, and records failures. This preserves a recoverable event trail without immediately introducing another infrastructure dependency.

### 2. Audit Trail

Add an application-level audit log for meaningful security and data events, rather than logging all technical noise.

Candidate events:

- Login, logout, failed authentication, password reset, and account-linking events.
- Creation, update, and deletion of confirmed food logs, targets, meals, health items, schedules, and profile settings.
- Background-job attempts, failures, and retries.
- Future external-agent/MCP requests and exports.

Audit records should contain actor type, user ID, action, resource type/ID, timestamp, request/correlation ID, outcome, and minimal change metadata. Sensitive values, passwords, tokens, health notes, and full provider payloads must not be copied into audit logs by default.

Retention duration, user-visible audit history, and whether every ordinary edit is audited remain future product decisions.

### 3. Analytics

Analytics is deferred until there are useful questions to answer and a privacy policy for event data. The eventual analytics path is:

```text
Core Service -> MongoDB outbox -> Worker Service -> analytics storage/aggregates
```

No analytics SDK, third-party tracker, or user-behaviour recording is required in Phase 1. When added, analytics must be user-scoped, minimize personal data, and never expose nutrition or health data to a third party without an explicit product and privacy decision.

### 4. Message Broker

RabbitMQ (or an equivalent broker) is not part of Phase 1 or the first Worker Service iteration. Reassess it only when MongoDB outbox polling causes measurable throughput, latency, ordering, retry, or operational problems.

```text
Core Service -> MongoDB outbox -> Worker Service       (first approach)
Core Service -> message broker -> Worker Service       (future option)
```

The Core domain model must not depend directly on a specific broker.

### 5. Notifications and Reminders

Deferred capabilities include browser push notifications, medication/supplement reminders, notification preferences, retry policies, and delivery tracking. Health-item schedules in Phase 1 still record events; they simply do not send notifications.

### 6. Offline and PWA Support

Deferred capabilities include an installable PWA, offline application shell, offline food logging, local queues, background synchronization, and conflict resolution. Phase 1 should handle unavailable backend/API responses gracefully but does not promise offline operation.

### 7. Additional Identity Providers

Google Sign-In is included in Phase 1. Other OAuth/OpenID Connect providers remain deferred. A future provider must use the same backend-owned OAuth/OIDC flow, verified stable subject identifiers, and explicit account-linking safeguards; email matching alone must not link an identity to an existing account.

### 8. Scaling and Deployment Evolution

Phase 4 may introduce multiple backend instances, load balancing, a managed queue, advanced monitoring, and migration from a single VM/VPS to a cloud or container platform. The application will remain portable by using Docker images, environment-based configuration, mounted secret files, and provider-neutral deployment files.

These are deployment choices, not prerequisites for Phase 1. Initial local development and testing come first; a domain, VPS, and cloud account are intentionally not required now.

## Entry Criteria for Phase 4 Work

Start a Phase 4 item only when its user value, data/privacy impact, operational cost, and acceptance criteria are defined. A capability should not be added solely to increase the number of deployed services.

## Non-Negotiable Product Boundaries

- The product tracks and organizes health information; it does not provide diagnosis, prescriptions, treatment decisions, or automatic dosage changes.
- All user data remains isolated by authenticated user identity.
- Any future asynchronous or external-agent action must use the same application authorization rules as the web application.
- Confirmed historical records remain stable and traceable.
