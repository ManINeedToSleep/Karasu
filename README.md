<div align="center">

```
    ██╗  ██╗ █████╗ ██████╗ ██╗   ██╗███████╗██╗   ██╗
    ██║ ██╔╝██╔══██╗██╔══██╗██║   ██║██╔════╝██║   ██║
    █████╔╝ ███████║██████╔╝██║   ██║███████╗██║   ██║
    ██╔═██╗ ██╔══██║██╔══██╗██║   ██║╚════██║██║   ██║
    ██║  ██╗██║  ██║██║  ██║╚██████╔╝███████║╚██████╔╝
    ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝ ╚═════╝
```

**烏 — the crow that hunts your music**

*A self-hosted music manager. Lidarr, but actually works.*

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat-square&logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-ready-2496ED?style=flat-square&logo=docker)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)

</div>

---

Karasu (烏) is named after the Japanese word for crow. Crows are obsessive collectors — they hunt for shiny things, hoard them, and organize them. That's exactly what this app does for your music.

It watches artists you care about, hunts them down on Soulseek, picks the best quality files it can find, organizes everything into a clean library structure, and tags every file properly. No babysitting required.

Built as a replacement for Lidarr, which is currently unmaintained and broken. Built in Go for speed, low memory footprint, and a single deployable binary.

---

## What it does

```
You add an artist
       ↓
Karasu fetches their full discography from MusicBrainz
       ↓
You mark an album as wanted (or it auto-queues)
       ↓
Karasu searches Soulseek via slskd
       ↓
Scores every result: FLAC > MP3, bitrate, completeness, uploader speed
       ↓
Downloads the best match
       ↓
Organizes files: /music/Artist/Album (Year)/01 - Track.flac
       ↓
Writes ID3 tags: title, artist, album, genres, track numbers
       ↓
Updates the database, marks album as downloaded
```

Everything happens in the background. You add an artist, walk away, come back to music.

---

## Features

- **MusicBrainz integration** — Search and import artists with full discography metadata, cover art, genres, release dates
- **Soulseek downloads via slskd** — Automatically searches the Soulseek network and downloads your music
- **Intelligent result scoring** — Prefers FLAC over MP3, higher bitrates, 24-bit depth, complete albums, fast uploaders
- **Wildcard retry** — If a search returns nothing, retries with a wildcard query to bypass Soulseek filters
- **File organization** — Moves files into a clean, consistent folder structure automatically
- **ID3 tag writing** — Writes proper metadata tags so every music player sees the right info
- **Library state tracking** — Tracks wanted / downloading / downloaded / missing status per album
- **REST API** — Full HTTP API for a frontend (like [Melodix](https://github.com/neopolitician/Melodix)) to plug into
- **Auto-migrations** — Database schema is managed automatically on startup
- **Docker-ready** — Single statically-linked binary, minimal Alpine image

---

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.26 |
| HTTP framework | Gin |
| Database | PostgreSQL + sqlx |
| Migrations | golang-migrate |
| Soulseek client | slskd (REST API) |
| Metadata | MusicBrainz API |
| Tag writing | bogem/id3v2 |
| Tag reading | dhowden/tag |
| Config | godotenv |

---

## Project structure

```
karasu/
├── cmd/
│   └── karusu/
│       └── main.go              # Entrypoint — wires everything together
├── internal/
│   ├── api/
│   │   ├── handlers.go          # HTTP handlers for all routes
│   │   └── helpers.go           # Date parsing, type normalization, discography sync
│   ├── db/
│   │   ├── db.go                # Connection + migration runner
│   │   ├── repository.go        # All database queries (artists, albums, tracks, genres)
│   │   └── migrations/
│   │       ├── 001_initial.up.sql
│   │       └── 001_initial.down.sql
│   ├── library/
│   │   ├── downloader.go        # Full download pipeline: search → score → fetch → organize
│   │   └── organizer.go         # File moving, folder structure, tag writing
│   ├── metadata/
│   │   └── musicbrainz.go       # MusicBrainz API client (rate-limited to 1 req/s)
│   ├── models/
│   │   └── models.go            # Artist, Album, Track, Genre structs and status enums
│   └── slskd/
│       ├── client.go            # slskd REST API client (search, download, status)
│       └── client_test.go
├── Dockerfile
├── .env.example
└── go.mod
```

---

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Liveness check |
| `GET` | `/api/search?q=` | Search MusicBrainz for artists |
| `GET` | `/api/artists` | List all monitored artists |
| `POST` | `/api/artists` | Add an artist (auto-syncs discography) |
| `GET` | `/api/artists/:id` | Artist detail with albums and tracks |
| `DELETE` | `/api/artists/:id` | Remove artist and all data |
| `PUT` | `/api/artists/:id/monitored` | Toggle monitoring |
| `GET` | `/api/albums/:id` | Album detail with track listing |
| `PUT` | `/api/albums/:id/download` | Trigger download for an album |
| `GET` | `/api/library/wanted` | All wanted-but-not-downloaded albums |

---

## Getting started

### Prerequisites

- [slskd](https://github.com/slskd/slskd) running and accessible (Soulseek client)
- PostgreSQL 14+
- Docker (recommended) or Go 1.22+

### With Docker Compose

Copy the example env file and fill in your values:

```bash
cp .env.example .env
```

```env
DB_HOST=postgres
DB_PORT=5432
DB_USER=karasu
DB_PASSWORD=yourpassword
DB_NAME=karasu

SLSKD_URL=http://slskd:5030
SLSKD_USERNAME=your_soulseek_username
SLSKD_PASSWORD=your_soulseek_password

MUSIC_DIR=/mnt/music
PORT=8080
```

Then build and run:

```bash
docker build -t karasu .
docker run --env-file .env -p 8080:8080 -v /mnt/music:/mnt/music karasu
```

### Building from source

```bash
git clone https://github.com/neopolitician/Karasu
cd Karasu
go build -o karasu ./cmd/karusu
./karasu
```

Database migrations run automatically on startup. No manual setup needed.

---

## How result scoring works

When Karasu finds multiple results for an album on Soulseek, it scores each one and picks the best. Higher score wins.

| Signal | Points |
|--------|--------|
| Upload speed | `speed / 100,000` |
| Free upload slots | `slots × 10` |
| No free upload slots | `-20` |
| FLAC files | `+50 per file` |
| 320kbps MP3 | `+20 per file` |
| 256kbps MP3 | `+10 per file` |
| 192kbps MP3 | `+5 per file` |
| 24-bit depth | `+15 per file` |
| Perfect track count match | `+100` |
| Near-perfect match (±1 track) | `+50` |

FLAC will almost always win. If you see an MP3 get picked it means either no one was sharing FLAC, or the FLAC uploader had no free slots and sluggish speed.

---

## File organization

Downloaded files are moved and renamed automatically:

```
/mnt/music/
└── Kendrick Lamar/
    └── GNX (2024)/
        ├── 01 - wacced out murals.flac
        ├── 02 - squabble up.flac
        ├── 03 - Luther.flac
        └── ...
```

Pattern: `{Music Dir}/{Artist}/{Album} ({Year})/{Track Number} - {Title}.{ext}`

ID3 tags are written for every file: title, artist, album, year, genres, track number, disc number.

---

## Part of a larger stack

Karasu is the backend half. It pairs with **Melodix** — a private music streaming app for you and your family. Karasu fills the library, Melodix plays it.

```
[Karasu]  ── downloads & organizes ──►  /mnt/music  ◄── streams from ──  [Melodix]
    │                                                                          │
 slskd                                                                    Your family
(Soulseek)
```

---

## Roadmap

- [ ] FLAC tag writing (currently MP3 only)
- [ ] Automatic wanted album monitoring (cron-based)
- [ ] Cover art embedding into files
- [ ] Multi-disc album support
- [ ] Lyrics fetching (Genius API key already in .env)
- [ ] Frontend UI (or delegate fully to Melodix)
- [ ] Notification webhooks on download complete

---

<div align="center">

*Named for the crow. Built for the hoard.*

</div>
