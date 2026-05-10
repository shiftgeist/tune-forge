# Playlist Organizer - Specification

## 1. Overview

A service that organizes a user's playlist library into **groups** with customizable rules. It enforces membership constraints and generates intersection playlists — all while using **playlist descriptions** as the source of truth.

The service is designed to be service-agnostic and can support multiple services in parallel (e.g., Spotify, YouTube, Deezer, Tidal, SoundCloud). V1 targets **Spotify** only.

**Key Principles**:

- **Service-agnostic**: Designed to work with any music service that supports playlists and track metadata. Services may have limited feature support — missing features result in disabled functionality, not errors.
- **Self-documenting**: Rules are defined in playlist descriptions in a human-readable format.
- **Opt-in**: No playlist is processed unless the user has explicitly opted it in during setup. The expected structure may be present, but nothing runs without the user's confirmation.
- **Self-healing**: The system handles transient failures (e.g., rate limits) internally via retries and exponential backoff. Users are not notified of infrastructure-level issues.

## V1

## 2. Core Features

### 2.1 Groups

A **group** represents a category (e.g., `Energy`, `Genre`, `Occasion`).

Groups are not a native concept in any supported music service. Instead, they are derived: a playlist declares its group membership in its description. On each sync, the service scans all opted-in playlists and assembles groups from these declarations.

### 2.2 Playlists

#### 2.2.1 Group Membership Declaration

Any opted-in playlist declares its group and rules in its description. The syntax is human-readable — exact format TBD. Example:

```
group: Energy, max: 1 per song
```

- `group` — the group this playlist belongs to.
- `max per song` — maximum number of playlists within the group a single song may appear in. Omitted means unlimited (no enforcement).

#### 2.2.2 User Playlists

Any name (e.g., `High Energy`, `House`). Rules declared in the description.

#### 2.2.3 Auto-Generated Playlists

| Playlist Name                  | Purpose                                                                 | Trigger                                    |
| ------------------------------ | ----------------------------------------------------------------------- | ------------------------------------------ |
| `🤖❌ Error: Duplicate`        | Song appears in more than 1 playlist in a group with `max: 1 per song`. | Conflict detected during validation.       |
| `🤖❌ Error: Max {n} exceeded` | Song exceeds a `max: n per song` limit greater than 1.                  | Conflict detected during validation.       |
| `🤖❌ Error: {message}`        | General error.                                                          | Any unclassified error.                    |
| `🤖ℹ️ {info}`                   | General info message.                                                   | Any unclassified info state.               |
| `🤖 {A} + {B}`                 | Intersection of songs from playlists A and B.                           | User creates a playlist named `{A} + {B}`. |

### 2.3 Group Membership Note

A track appearing in multiple playlists across groups is natural and expected — no enforcement applies unless `max per song` is explicitly set on a group. Unlimited membership is the default and requires no validation.

There is no Spotify API endpoint to query which playlists contain a given track. The service builds this mapping itself by scanning all opted-in playlists and inverting the track index.

## 3. Workflows

### 3.1 Opt-In Setup

Before any sync or validation runs, the user explicitly opts playlists in via the setup step in the UI. Only opted-in playlists are scanned or modified.

### 3.2 Sync & Validation

**Trigger**: Manual or cron schedule.

For each group assembled from opted-in playlists:

1. Fetch all member playlists and their tracks.
2. Build a map of track → playlists it appears in within the group.
3. If a track's count exceeds `max per song` (and the limit is set), move it to the appropriate error playlist.

### 3.3 Generated (Intersection) Playlists

**Trigger**: User creates a playlist named `{A} + {B}`.

1. Detect the `+` pattern in the playlist name.
2. Split into parts and find the matching opted-in playlists.
3. Populate with songs that appear in **all** source playlists.

Updates are triggered on manual sync or cron — not automatically on source playlist changes.

## 4. Music Service Integration

### 4.1 Common Interface

Each supported music service implements a common interface covering:

- Fetching opted-in playlists and their tracks
- Reading and writing playlist descriptions
- Creating, updating, and deleting playlists
- Adding and removing tracks (batch where supported, sequential fallback)
- Fetching track metadata

Services with limited capabilities implement what they can. Missing features disable the relevant UI functionality.

### 4.2 Track Metadata Caching

Track metadata is cached by track ID with a long TTL — track metadata rarely changes and should be retained as long as possible. Invalidation is on-demand only.

### 4.3 Rate Limiting

Handled internally via exponential backoff and silent retries. No user-facing notifications.

### 4.4 Batch Operations

Batch API calls are preferred. Services without batch support fall back to sequential requests transparently.

## 5. Spotify

### 5.1 Notes

- Spotify does not natively support folders. Groups are derived entirely from playlist descriptions.
- No playlist name prefixes are used — this would disrupt the in-app experience.
- There is no API endpoint to look up which playlists contain a given track. The service must build this by scanning opted-in playlists.

### 5.2 Required Scopes

- `playlist-read-private`
- `playlist-read-collaborative`
- `playlist-modify-private`
- `playlist-modify-public`
- `user-library-read`

### 5.3 Relevant Endpoints

| Feature             | Endpoint                                      |
| ------------------- | --------------------------------------------- |
| Fetch playlists     | `GET /me/playlists`                           |
| Get playlist tracks | `GET /playlists/{playlist_id}/tracks`         |
| Get track metadata  | `GET /audio-features/{track_id}`              |
| Create playlist     | `POST /users/{user_id}/playlists`             |
| Add/remove tracks   | `POST/DELETE /playlists/{playlist_id}/tracks` |

## 6. Data Storage

- **In-Memory Cache**: Assembled group state per sync cycle.
- **Persistent Track Cache**: Track metadata stored persistently (SQLite or equivalent). Long TTL — invalidated on demand only.
- **Opt-In List**: Persisted list of playlists the user has enabled. Storage mechanism TBD (SQLite or config file).

## 7. Lightweight UI

V1 UI covers:

- Spotify OAuth2 authentication.
- Opt-in setup: selecting which playlists to include in scanning.
- Manual trigger for sync and validation.

## 8. Open Questions

1. **Opt-In Storage**: Where is the opt-in list persisted — SQLite, config file, or service-side (e.g., a dedicated playlist)?
2. **Description Syntax**: Exact human-readable format for group declarations and rules. Needs a concrete decision before implementation.

## V2

## 9. V2 Features

### 9.1 Auto-Sorting

When a song is added to a drop-zone playlist with `sort by` defined in its description, the service automatically routes it to the correct sub-playlist based on track metadata.

Description example:

```
group: Energy, sort by: energy
```

Auto-generated playlists added in V2:

| Playlist Name               | Purpose                                                    | Trigger                                            |
| --------------------------- | ---------------------------------------------------------- | -------------------------------------------------- |
| `🤖❌ Error: Key not found` | Track lacks the `sort by` metadata key.                    | Auto-sorting fails — metadata key absent.          |
| `🤖❌ Error: No match`      | Track's metadata doesn't match any sub-playlist.           | Auto-sorting fails — no matching playlist found.   |
| `🤖ℹ️ To be sorted`          | Holding area for tracks that couldn't be auto-sorted.      | `sort by` set but no matching sub-playlist exists. |
| `🤖ℹ️ Queue`                 | Holding area for tracks pending deferred async processing. | Track added while async processing is in progress. |

### 9.2 Async Queue

Deferred and long-running operations (e.g., large auto-sort batches) are processed via a persistent queue that survives restarts. Tracks pending processing are surfaced in `🤖ℹ️ Queue`.

### 9.3 Additional Music Services

V2 extends support beyond Spotify. All services implement the common interface from section 4.1. The UI remains service-agnostic — unsupported features are disabled per service.

#### YouTube Music

- Folders: Not natively supported. Group discovery via description scanning.
- Metadata: Limited acoustic metadata. May require external sources or manual tagging.
- API: YouTube Data API.

#### Deezer

- Folders: Natively supported via the API.
- Metadata: BPM, gain, rank available. No acoustic features comparable to Spotify's audio features.
- API: Deezer REST API.

#### Tidal

- Folders: Not natively supported. Group discovery via description scanning.
- Metadata: Strong audio quality attributes (lossless, hi-res flags) but no energy/danceability equivalents.
- API: Tidal API.

#### SoundCloud

- Folders: Not natively supported. Group discovery via description scanning.
- Metadata: User-defined tags and genre only. No acoustic feature metadata.
- API: SoundCloud API.

### 9.4 Dashboard

An interactive dashboard for:

- Viewing group state and membership.
- Inspecting and resolving conflicts (duplicate songs, max exceeded).
- Managing opted-in playlists.

### 9.5 Non-Power-User Flows

Guided setup and simplified management for users unfamiliar with description syntax.
