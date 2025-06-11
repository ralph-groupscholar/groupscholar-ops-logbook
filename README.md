# Group Scholar Ops Logbook

Ops Logbook is a lightweight operations signal tracker for Group Scholar. It helps program teams capture incidents, escalations, and wins in a shared logbook so interventions stay visible and actionable.

## Features
- Log operational signals with severity, owner, and status.
- Filter recent signals by status or category.
- Review live summary metrics for status, severity, and ownership trends.
- Store all activity in a Postgres-backed logbook schema.

## Tech Stack
- Go (Vercel serverless functions)
- Postgres (production data store)
- Vanilla HTML/CSS/JS frontend

## Local Development
1. Set `DATABASE_URL` to a Postgres instance.
2. Run a local server that proxies `/api` to Vercel functions or deploy to Vercel for full functionality.

## Production Notes
- The production database schema lives in `db/schema.sql`.
- Seed data should be inserted into the production database after deployment.
