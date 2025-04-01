# Banking Ledger System

A complete banking transaction system with account creation and transaction processing capabilities.

## Overview

The Banking Ledger system is a microservices-based architecture that provides:
- Account creation
- Transaction processing (deposits and withdrawals)
- Transaction history tracking
- API endpoints for a few banking operations

## System Architecture

The system consists of several components:
- **API Service**: RESTful API for client interactions
- **Account Creator Worker**: Processes account creation requests
- **Transaction Processor Worker**: Handles transaction requests
- **PostgreSQL**: Stores account data
- **Elasticsearch**: Stores transaction history for efficient querying
- **RabbitMQ**: Message broker for asynchronous processing

## API Endpoints

The system exposes the following RESTful endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/createAccount` | POST | Create a new bank account |
| `/account/status/{referenceId}` | GET | Check the status of an account creation request |
| `/transact` | POST | Process a deposit or withdrawal |
| `/account/{accountNumber}/transactionHistory` | GET | Retrieve transaction history for an account |

## Getting Started

### Prerequisites
- Docker and Docker Compose
- Git

### Installation and Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/siddarth99/banking-ledger.git
   cd banking-ledger
   ```

2. Start the services using Docker Compose:
   ```bash
   docker-compose up -d
   ```

3. Check if all services are running:
   ```bash
   docker-compose ps
   ```

## Using the System

### Swagger Documentation

The API is documented using OpenAPI. You can access the Swagger UI at:
```
http://localhost:9090
```

### API Usage Examples

Below are curl examples for interacting with the API:

#### Create a new account
```bash
curl -X POST "http://localhost:8080/createAccount" \
  -H "Content-Type: application/json" \
  -d '{
    "accountHolderName": "John Doe",
    "initialDeposit": 1000.00,
    "branchCode": "BR1"
  }'
```

#### Check account creation status and get your account Id
```bash
curl -X GET "http://localhost:8080/account/status/{referenceId}"
```
Replace `{referenceId}` with the UUID returned from the account creation request.

#### Process a transaction
```bash
curl -X POST "http://localhost:8080/transact" \
  -H "Content-Type: application/json" \
  -d '{
    "accountNumber": "BR11234567",
    "amount": 500.00,
    "type": "DEPOSIT"
  }'
```

#### Get transaction history
```bash
curl -X GET "http://localhost:8080/account/BR11234567/transactionHistory"
```

## Services and Ports

| Service | Port | Description |
|---------|------|-------------|
| API | 8080 | RESTful API |
| Swagger UI | 9090 | API Documentation |
| PostgreSQL | 6000 | Database (exposed) |
| Elasticsearch | 9200 | Search engine |
| RabbitMQ | 5672 | Message broker |
| RabbitMQ Management | 15671 | Admin interface |

## Technical Implementation

### Database Schema

The system uses PostgreSQL for persistent storage with tables for accounts and transactions.

### Message Processing

The system uses RabbitMQ for asynchronous processing:
- Account creation requests are processed by the account creator worker
- Transaction requests are processed by the transaction processor worker

### Search Capabilities

Transaction history is indexed in Elasticsearch to provide efficient querying and filtering.

## Monitoring and Management

### Elasticsearch

Access Elasticsearch directly at:
```
http://localhost:9200
```

## Troubleshooting

If services fail to start properly:

1. Check container logs:
   ```bash
   docker-compose logs api
   docker-compose logs worker
   docker-compose logs transaction_processor
   ```

2. Ensure all dependent services are healthy:
   ```bash
   docker-compose ps
   ```

3. Restart services if needed:
   ```bash
   docker-compose restart api worker transaction_processor
   ```

## Shutting Down

To stop all services:
```bash
docker-compose down
```

To stop services and remove all data volumes:
```bash
docker-compose down -v
```

## Architecture Discussion

The system uses a combination of synchronous and asynchronous processing to ensure high throughput and scalability. Account creation and transaction processing are handled asynchronously through message queues, while inquiries are processed synchronously for immediate feedback.
