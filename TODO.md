* Allow concurrent reads if there are no writes
* Upgrade Go
* Don't stream content of non-ok responses from GCS

# Tests
* Replace mock sleep function with env properties for timings
* Time comparisons should use env properties instead of constants

## Basic Endpoints
 * Health
 * IP

## Mock Systems
 * Create signed URL
 * Get GCS egress
 * Sleep

## Sequences
 * 