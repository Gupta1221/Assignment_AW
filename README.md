## Overview
This application provides a RESTful API to manage risks with features like structured logging, validation, and graceful shutdown.

## Requirements
- Go 1.18+ installed.
- Port configuration through `APP_PORT` (default: `8080`).

## Running the Application
1. Clone the repository:
   git clone <repo-url>
   
## Install dependency:
go mod tidy

## Run the server:
go run main.go

## API Endpoints
GET /v1/risks: Retrieve all risks.
POST /v1/risks: Create a risk. 

Example payload:
json
{
  "state": "open",
  "title": "Sample Risk",
  "description": "Description of the risk."
}

GET /v1/risks/{id}: Retrieve a risk by its ID.

   
