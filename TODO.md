* Don't use Unix time as it uses seconds and the tests need more accuracy
* Upgrade Go
* Don't stream content of non-ok responses from GCS

# Unimportant
 * Allow concurrent reads if there are no writes
 * Scan for available ports during tests to allow concurrency

# Tests

## Basic Endpoints

## Mock Systems
 * Create signed URL
 * Get GCS egress

## Sequences
 * Is the egress reset after a 'day'? Can the user then hit the limit again?
 * Multiple simultanious users?