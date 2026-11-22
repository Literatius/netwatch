# Netwatch

**Netwatch** is a lightweight network monitoring tool built with Go, Python, FastAPI, and SQLite.

The agent written in Go performs network checks at a configurable interval. Results are sent to the FastAPI backend written in Python and stored in local SQLite database.

A failed network check does not stop the monitoring agent.

## Features

- **TCP Monitoring**
- **HTTP Monitoring**
- **DNS Monitoring**
- **TLS Monitoring**
- **Concurrent Checks built with goroutine and channels** 
- **SQLite History**

## Installation

### Requirements

- Go
- Python 3.10+
- pip

### 1. Clone the repository

```bash
git clone https://github.com/Literatius/netwatch.git
cd netwatch
```

### 2. Set up the Python stack

Install the required packages:

```bash
python -m pip install fastapi uvicorn
```

Start the backend:

```bash
python -m uvicorn main:app --reload
```

The API will be available at:

```text
http://127.0.0.1:8000
```

Interactive API documentation:

```text
http://127.0.0.1:8000/docs
```

### 3. Configure the Go agent

Create a `config.json` file in the main directory or copy the existing example `config.example.json`.

Configuration options:

`interval_seconds` - Interval between monitoring cycles 
`timeout_seconds` - Network request timeout in seconds 
`api_url` - FastAPI endpoint for receiving results
`targets` -List of network targets to monitor 

### 4. Run the Go agent

```bash
go run .
```
