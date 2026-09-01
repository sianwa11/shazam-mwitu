# Shazam Mwitu

Shazam Mwitu is a project exploring how audio fingerprinting works the kind of technology behind apps that can identify a song just by "listening" to a short clip of it.

This is a work in progress, built as a learning exercise to understand the pipeline behind recognition, from raw audio all the way to a recognizable "fingerprint" of a song.

More details will be added as the project matures.

## How it works

1. Convert input audio to WAV (via FFmpeg) and downsample
2. Parse WAV, convert to mono
3. Slice into overlapping frames, apply a Hamming window
4. Run FFT on each frame → build a spectrogram
5. Pick spectral peaks from the spectrogram
6. (next) Generate hashes from peaks and store them for fast lookup
7. (next) Match a short recording against the stored fingerprints

## Progress

- [x] WAV parsing
- [x] Mono conversion + downsampling
- [x] Windowing + FFT + spectrogram
- [x] Peak-picking
- [ ] Target zones / hashing
- [ ] Fingerprint storage
- [ ] Matching / search
