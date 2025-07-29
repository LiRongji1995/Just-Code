#!/bin/bash
curl -X POST http://localhost:8080/announce \
  -H "Content-Type: application/json" \
  -d '{
    "file_hash": "abc123",
    "peer_id": "peer002",
    "ip": "192.168.1.101",
    "port": 8082,
    "uploaded": 0,
    "downloaded": 0,
    "left": 0,
    "event": "started"
  }'
