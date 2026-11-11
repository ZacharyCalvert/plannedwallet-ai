# AGENTS.md - Repository Rules & Instructions

Back-end development is in golang.
Front-end is ReactJS using NextJS.
Database is postgres, with an in-memory variation for local testing.
No functions more than 40 lines.
No cyclomatic complexity in functions greater than 6.
User operations / API interactions log with WC3 trace context.
The integration tests in "itests" execute against a back-end with a completely fresh (wiped) database, every time.

# 1. Project Stack
- Tech Stack: Next.js for Front-End.  Golang for back-end.  Postgres for database.  Containers built with intent that they are deployed within a docker compose for local testing and integration testing.
- Back-end utilizes Gin for routing and pq for database interactions.

# 2. Folder structure

- backend-api for the back-end services
- frontend-ui for the front-end presentation layer in browsers
- docker-compose.yml for local development
- each work artifact should have a Makefile for executing builds, service start, and testing.

