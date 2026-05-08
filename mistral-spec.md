# Spotify Sorter - Technical Specification

## 1. Overview

**Spotify Sorter** is a service that organizes a user's Spotify library into **groups (folders)** with customizable rules. It enforces constraints (e.g., one-to-one or one-to-many memberships), auto-sorts songs based on metadata, and generates intersection playlists—all while using **Spotify’s native folder/playlist structure** as the source of truth.

**Key Principles**:

- No local database: All data is read from/written to Spotify.
- No UI dependency: A lightweight dashboard is optional for non-power users.
- Self-documenting: Rules are defined in playlist descriptions.
- Extensible: Supports custom metadata keys and validation logic.

## 2. Spotify Structure

### 2.1 Folders (Groups)

- Each **folder** represents a **group** (e.g., `Energy`, `Genre`, `Moment`).
- Folders contain:
  - A `**🤖 {Group Name}**` playlist (auto-created if missing) with a **description** defining the group’s rules.
  - **User-created playlists** (e.g., `High Energy`, `House`).
  - **Auto-generated playlists** for errors or unsorted songs (e.g., `🤖❌ Error: Duplicate`).

### 2.2 Playlists

#### 2.2.1 Group Playlist (`🤖 {Group Name}`)

- **Purpose**: Defines the group’s rules and acts as a "drop zone" for unsorted songs.
- **Description Format**:
  ```
  max_per_song={n},sort_by={metadata_key}
  ```
  - `max_per_song` (int):
    - Maximum number of playlists a song can belong to in this group.
    - `0` = unlimited (default).
  - `sort_by` (string, optional):
    - Spotify track metadata key (e.g., `energy`, `genre`, `danceability`).
    - If set, songs added to `🤖 {Group Name}` are **auto-sorted** into sub-playlists based on this metadata.
- **Example**:
  - `🤖 Energy` (desc: `"max_per_song=1,sort_by=energy"`)
  - `🤖 Genre` (desc: `"max_per_song=3,sort_by=genre"`)
  - `🤖 Moment` (desc: `"max_per_song=0"`)

#### 2.2.2 User Playlists

- **Naming**: Any name (e.g., `High Energy`, `House`).
- **Description (Optional)**:
  - For sub-playlists under a group with `sort_by`, the description can define **value ranges** (e.g., `"energy>0.7"` for `High Energy`).
  - If no description is provided, the backend uses **default ranges** (configurable).

#### 2.2.3 Auto-Generated Playlists

| Playlist Name                  | Purpose                                                     | Trigger                                               |
| ------------------------------ | ----------------------------------------------------------- | ----------------------------------------------------- |
| `🤖❌ Error: Duplicate`        | Song appears in >`max_per_song` playlists in the group.     | Conflict detected during validation.                  |
| `🤖❌ Error: Max {n} exceeded` | Song exceeds `max_per_song` limit.                          | Conflict detected during validation.                  |
| `🤖❌ Error: Key not found`    | Song lacks the `sort_by` metadata.                          | Auto-sorting fails.                                   |
| `🤖❌ Error: No match`         | Song’s metadata doesn’t match any sub-playlist.             | Auto-sorting fails.                                   |
| `🤖 To be sorted`              | Temporary holding area for songs that can’t be auto-sorted. | `sort_by` is set but no matching sub-playlist exists. |
| `🤖 {A} + {B}`                 | Intersection of songs from playlists `A` and `B`.           | User creates a playlist named `A + B`.                |

## 3. Workflows

### 3.1 Validation

**Trigger**: Manual (via API call) or scheduled (e.g., daily).
**Steps**:

1. For each **group folder**:
   a. Fetch all sub-playlists and their songs.
   b. For each song, count how many playlists it appears in within the group.
   c. If `count > max_per_song` (and `max_per_song != 0`):
   - Move the song to `🤖❌ Error: Max {n} exceeded` (or `🤖❌ Error: Duplicate` if `max_per_song=1`).
2. Log all validation errors for the dashboard.

### 3.2 Auto-Sorting

**Trigger**: When a song is added to `🤖 {Group Name}`.
**Steps**:

1. Check if the group’s `🤖 {Group Name}` playlist has `sort_by` defined.
2. If yes:
   a. Fetch the song’s metadata (e.g., `energy` value).
   b. Find the sub-playlist whose description matches the metadata (e.g., `energy>0.7` for `High Energy`).
   c. If a match is found, move the song to that sub-playlist.
   d. If no match is found:
   - If a `🤖 To be sorted` playlist exists, move the song there.
   - Else, create `🤖 To be sorted` and move the song.
     e. If the song lacks the `sort_by` metadata, move to `🤖❌ Error: Key not found`.
3. If no `sort_by` is defined, leave the song in `🤖 {Group Name}`.

### 3.3 Generated Playlists

**Trigger**: When a user creates a playlist with `+` in the name (e.g., `House + High Energy`).
**Steps**:

1. Detect the `+` in the playlist name.
2. Split the name into parts (e.g., `["House", "High Energy"]`).
3. Find the playlists matching each part.
4. Create a new playlist named `🤖 {Part1} + {Part2}`.
5. Populate it with songs that appear in **all** source playlists (intersection).
6. If no songs match, leave the playlist empty (or delete it).

### 3.4 Error Resolution

- Users can manually move songs from error playlists to correct sub-playlists.
- The backend **re-validates** the group after manual changes.

## 4. Spotify API Integration

### 4.1 Required Scopes

| Scope                         | Purpose                                                  |
| ----------------------------- | -------------------------------------------------------- |
| `playlist-read-private`       | Read user’s private playlists.                           |
| `playlist-read-collaborative` | Read collaborative playlists.                            |
| `playlist-modify-private`     | Modify user’s private playlists.                         |
| `playlist-modify-public`      | Modify user’s public playlists.                          |
| `user-library-read`           | Read saved tracks (for metadata like `energy`, `genre`). |

### 4.2 Key Endpoints

| Endpoint                        | Purpose                                                          |
| ------------------------------- | ---------------------------------------------------------------- |
| `GET /me/playlists`             | Fetch all playlists (including folders).                         |
| `GET /playlists/{id}`           | Get playlist metadata (name, description).                       |
| `GET /playlists/{id}/tracks`    | Get songs in a playlist.                                         |
| `POST /playlists/{id}/tracks`   | Add songs to a playlist.                                         |
| `DELETE /playlists/{id}/tracks` | Remove songs from a playlist.                                    |
| `POST /playlists`               | Create a new playlist.                                           |
| `GET /audio-features/{id}`      | Get audio features (e.g., `energy`, `danceability`) for a track. |
| `GET /tracks/{id}`              | Get track metadata (e.g., `genre`, `release_date`).              |

### 4.3 Rate Limiting

- Spotify’s free tier allows **~5,000 requests/day**.
- **Mitigation**:
  - Batch requests where possible (e.g., fetch multiple tracks’ metadata in one call).
  - Cache metadata locally (in-memory) during a session to avoid repeated calls.
  - Use exponential backoff for retries.

## 5. Backend Service

### 5.1 Architecture

- **Language**: Go (recommended for simplicity/performance) or Node.js/Deno/Bun.
- **Deployment**: Docker container or serverless (e.g., AWS Lambda, Fly.io).
- **Authentication**: OAuth 2.0 flow to authorize with Spotify.
- **Configuration**: Environment variables for Spotify credentials and default ranges.

### 5.2 Core Components

#### 5.2.1 Spotify Client

- Wrapper for Spotify API calls.
- Handles authentication, retries, and rate limiting.

#### 5.2.2 Group Manager

- **Responsibilities**:
  - Fetch and parse group folders/playlists.
  - Validate groups against `max_per_song` rules.
  - Auto-sort songs in `🤖 {Group Name}` playlists.
  - Generate intersection playlists.

#### 5.2.3 Metadata Cache

- Stores track metadata (e.g., `energy`, `genre`) to avoid repeated API calls.
- **TTL**: 1 hour (configurable).

#### 5.2.4 Error Handler

- Logs errors (e.g., validation failures, API issues).
- Provides error details for the dashboard.

#### 5.2.5 Dashboard API (Optional)

- **Endpoints**:
  - `GET /groups`: List all groups and their status (e.g., errors, unsorted songs).
  - `POST /groups/{id}/validate`: Trigger validation for a group.
  - `POST /groups/{id}/resolve`: Manually resolve errors (e.g., move a song from `🤖❌ Error: Duplicate` to a sub-playlist).

## 6. Data Model

### 6.1 Group

```typescript
interface Group {
  id: string;               // Spotify folder ID (or derived from playlist name)
  name: string;             // e.g., "Energy"
  playlistId: string;       // Spotify ID of `🤖 {Group Name}` playlist
  maxPerSong: number;       // Default: 0 (unlimited)
  sortBy?: string;          // e.g., "energy"
  subPlaylists: SubPlaylist[];
  errors: ErrorPlaylist[];
}
```

### 6.2 Sub-Playlist

```typescript
interface SubPlaylist {
  id: string;               // Spotify playlist ID
  name: string;             // e.g., "High Energy"
  description: string;      // e.g., "energy>0.7"
  groupId: string;          // Parent group ID
  tracks: Track[];          // Songs in this playlist
}
```

### 6.3 Error Playlist

```typescript
interface ErrorPlaylist {
  id: string;               // Spotify playlist ID
  name: string;             // e.g., "🤖❌ Error: Duplicate"
  groupId: string;          // Parent group ID
  type: "duplicate" | "max_exceeded" | "key_not_found" | "no_match";
  tracks: Track[];          // Songs flagged for this error
}
```

### 6.4 Track

```typescript
interface Track {
  id: string;               // Spotify track ID
  name: string;
  metadata: Record<string, any>; // e.g., { energy: 0.85, genre: "house" }
}
```

## 7. Configuration

### 7.1 Environment Variables

| Variable                | Description                                                              |
| ----------------------- | ------------------------------------------------------------------------ |
| `SPOTIFY_CLIENT_ID`     | Spotify Developer App Client ID.                                         |
| `SPOTIFY_CLIENT_SECRET` | Spotify Developer App Client Secret.                                     |
| `SPOTIFY_REDIRECT_URI`  | OAuth redirect URI (e.g., `http://localhost:8080/callback`).             |
| `DEFAULT_RANGES`        | JSON string defining default ranges for `sort_by` keys. Example:         |
| &nbsp;                  | `{"energy": [{"min": 0.7, "max": 1.0, "playlist": "High Energy"}, ...]}` |

### 7.2 Default Ranges

If a sub-playlist lacks a description (e.g., `High Energy` has no `energy>0.7`), the backend uses **default ranges** from `DEFAULT_RANGES`.
Example for `energy`:

| Playlist    | Range     |
| ----------- | --------- |
| High Energy | 0.7 - 1.0 |
| Mid Energy  | 0.4 - 0.7 |
| Low Energy  | 0.0 - 0.4 |

## 8. Error Handling

| Scenario                          | Action                                               |
| --------------------------------- | ---------------------------------------------------- |
| Song in >`max_per_song` playlists | Move to `🤖❌ Error: Max {n} exceeded`.              |
| Song lacks `sort_by` metadata     | Move to `🤖❌ Error: Key not found`.                 |
| No sub-playlist matches metadata  | Move to `🤖 To be sorted`.                           |
| Spotify API rate limit hit        | Retry with exponential backoff.                      |
| Playlist not found                | Log error and skip.                                  |
| Invalid group description         | Log error and use defaults (e.g., `max_per_song=0`). |

## 9. Dashboard (Optional)

### 9.1 Features

- **Group Overview**: List all groups with their `max_per_song` and `sort_by` rules.
- **Error View**: Show all error playlists and their songs.
- **Validation Trigger**: Button to re-validate all groups.
- **Manual Resolution**: Drag-and-drop to move songs from error playlists to sub-playlists.

### 9.2 Tech Stack

- **Frontend**: Svelte (lightweight, reactive).
- **Backend**: Same as the main service (Go/Node) with additional API endpoints.
- **Deployment**: Static site (e.g., Vercel, Netlify) + backend API.

## 10. Example Scenarios

### 10.1 Scenario 1: Auto-Sorting a Song

1. User adds a song to `🤖 Energy` (desc: `"max_per_song=1,sort_by=energy"`).
2. Backend fetches the song’s `energy` value (e.g., `0.85`).
3. Matches `0.85` to `High Energy` (range: `0.7-1.0`).
4. Moves the song to `High Energy`.

### 10.2 Scenario 2: Detecting a Duplicate

1. User adds a song to both `High Energy` and `Low Energy`.
2. Backend validates the `Energy` group (`max_per_song=1`).
3. Detects the song in 2 playlists → moves it to `🤖❌ Error: Duplicate`.

### 10.3 Scenario 3: Generating an Intersection Playlist

1. User creates a playlist named `House + High Energy`.
2. Backend detects `+` and creates `🤖 House + High Energy`.
3. Populates it with songs that are in **both** `House` and `High Energy`.

## 11. Open Questions

1. **Metadata Ranges**:

- Should ranges be **hardcoded** in the backend or **defined in sub-playlist descriptions**?
- Example: `High Energy` desc: `"energy>0.7"` vs. backend using `DEFAULT_RANGES`.

2. **Conflict Resolution**:

- If a song is in `🤖❌ Error: Duplicate`, should the backend **auto-resolve** it (e.g., pick one playlist) or leave it for manual resolution?

3. **Generated Playlist Updates**:

- Should `🤖 {A} + {B}` playlists **auto-update** when `A` or `B` changes, or only on manual trigger?

4. **Folder Creation**:

- Should the backend **auto-create folders** for groups if they don’t exist, or assume they’re manually created?

5. **Dashboard Scope**:

- Should the dashboard be **read-only** (view errors) or **interactive** (resolve errors, trigger actions)?

## 12. Roadmap

| Phase | Priority | Description                                                              |
| ----- | -------- | ------------------------------------------------------------------------ |
| 1     | High     | Spotify API integration (auth, fetch playlists/tracks).                  |
| 2     | High     | Group validation (`max_per_song` enforcement).                           |
| 3     | High     | Auto-sorting (`sort_by` + metadata).                                     |
| 4     | Medium   | Generated playlists (intersection logic).                                |
| 5     | Medium   | Error playlists (auto-creation and population).                          |
| 6     | Low      | Dashboard (optional, for non-power users).                               |
| 7     | Low      | Advanced features (e.g., OR logic in generated playlists, bulk actions). |
