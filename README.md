# SIUU Bank

SIUU Bank is a backend banking service built in Go, focused on correctness, security, and transactional integrity.  
The project exposes a RESTful API for account management and demonstrates real-world backend concepts such as authentication, database transactions, and containerized development.

---

## Overview

SIUU Bank is a minimal banking backend that allows users to:
- create accounts
- authenticate securely
- retrieve account information
- perform atomic money transfers

The project intentionally prioritizes backend fundamentals—data integrity, access control, and reproducibility—over UI or deployment complexity.

---

## Key Features

### Account Management
- Create and delete bank accounts
- Securely store user credentials using bcrypt hashing

### Authentication & Authorization
- JWT-based authentication
- Route-level authorization enforcing per-account access

### Transactional Money Transfers
- Atomic transfers implemented using PostgreSQL transactions
- Prevents partial updates, race conditions, and inconsistent balances

### Persistent Storage
- PostgreSQL used as the primary data store
- Explicit schema creation and initialization handled by the service

### Dockerized Local Development
- Fully containerized backend and database
- Reproducible setup using Docker and Docker Compose

---

## Tech Stack

- **Language:** Go
- **Database:** PostgreSQL
- **Authentication:** JWT
- **Containerization:** Docker, Docker Compose
- **Libraries:**
  - Gorilla Mux (routing)
  - bcrypt (password hashing)
  - lib/pq (PostgreSQL driver)

---

## Architecture Overview

The project follows a clear separation of concerns:

- **Domain Layer**
  - Core account model and validation logic
- **Storage Layer**
  - PostgreSQL-backed implementation using SQL transactions
- **HTTP Layer**
  - RESTful API handlers and authentication middleware
- **Infrastructure**
  - Dockerized environment for consistent local development

This structure keeps business logic independent from transport and persistence details.

---

## API Overview

The service exposes RESTful JSON endpoints for:
- account creation and retrieval
- user authentication
- protected account access
- money transfer operations

Authorization is enforced via JWT tokens attached to protected requests.

---

## Running Locally

### Prerequisites
- Docker
- Docker Compose

### Start the service

```bash
docker compose up --build
