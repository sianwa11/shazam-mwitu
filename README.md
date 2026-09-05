# Shazam Mwitu

Shazam Mwitu is a project exploring how audio fingerprinting works, the kind of technology behind apps that can identify a song just by "listening" to a short clip of it.

Built as a learning exercise to understand the full pipeline behind recognition, from raw audio all the way to a searchable fingerprint database and back to a confident (or not-so-confident) match.

## Tech stack

- **Go** - core language, entire pipeline hand-written (WAV parsing, windowing, hashing, matching)
- **[gonum](https://gonum.org/)** - FFT computation
- **FFmpeg** - audio format conversion and downsampling (invoked via `os/exec`)
- **SQLite** - fingerprint storage
- **[sqlc](https://sqlc.dev/)** - type-safe generated SQL queries
- **[goose](https://github.com/pressly/goose)** - database schema migrations

## How it works

1. Convert input audio to WAV (via FFmpeg) and downsample
2. Parse the WAV file, convert to mono
3. Slice the signal into overlapping frames, apply a Hamming window to each
4. Run an FFT on each frame → build a spectrogram (time × frequency)
5. Pick spectral peaks from the spectrogram
6. Sort peaks and group them into overlapping "target zones"
7. Generate hashes (address + anchor time) from each target zone
8. Store hashes in SQLite, keyed by address, for fast lookup

To identify a recording, the same pipeline runs on a short clip, and its
hashes are looked up against the stored database. Matches are scored by
checking how consistently they agree on a single time offset between the
recording and a candidate song — a real match produces many hashes agreeing
on the same offset; coincidental matches scatter randomly. The result is
reported as one of three confidence levels: a confident match, a possible
match, or no match found.

## Usage

Build the fingerprint database from a folder of songs:

​```bash
go run ./cmd/build
​```

Identify a recording against the database:

​```bash
go run ./cmd/match
​```

## Progress

- [x] WAV parsing
- [x] Mono conversion + downsampling
- [x] Windowing + FFT + spectrogram
- [x] Peak-picking
- [x] Target zones / hashing
- [x] Fingerprint storage (SQLite, via sqlc + goose migrations)
- [x] Matching / search with tiered confidence scoring

## Next up

- [ ] HTTP API wrapping the existing pipeline
- [ ] Minimal frontend for uploading/recording a clip
- [ ] Deployment
- [ ] Write-up explaining the theory and build process in detail
