# Playlist Organizer - Specification

## 1. Overview

A service that organizes a user's playlist library into **groups** with customizable rules. It enforces constraints (e.g., one-to-one or one-to-many memberships), auto-sorts songs based on metadata, and generates intersection playlists—all while using the **native playlist structure of the music service** as the source of truth.

The service is designed to be service agnostic and can support multiple services in parallel (e.g., Spotify, YouTube, Deezer, Tidal, SoundCloud). Initially, it will support **Spotify**.

**Key Principles**:

- **Service-agnostic**: Designed to work with any music service that supports playlists and track metadata.
- **Lightweight UI**: A minimal UI will be required for authentication, setup and normal operation for non-power users.
- **Self-documenting**: Rules are defined in playlist descriptions.

## 2. Core Features

### 2.1 Groups

- Each **group** represents a category (e.g., `Energy`, `Genre`, `Occasion`).
- Groups are defined by a special playlist (e.g., `🤖 {Group Name}`) with a **description** that defines the group’s rules.

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
    - Metadata key (e.g., `energy`, `genre`, `danceability`).
    - If set, songs added to `🤖 {Group Name}` are **auto-sorted** into sub-playlists based on this metadata.

#### 2.2.2 User Playlists

- **Naming**: Any name (e.g., `High Energy`, `House`).

#### 2.2.3 Auto-Generated Playlists

| Playlist Name                  | Purpose                                                     | Trigger                                               |
| ------------------------------ | ----------------------------------------------------------- | ----------------------------------------------------- |
| `🤖❌ Error: Duplicate`        | Song appears in >`max_per_song` playlists in the group.     | Conflict detected during validation.                  |
| `🤖❌ Error: Max {n} exceeded` | Song exceeds `max_per_song` limit.                          | Conflict detected during validation.                  |
| `🤖❌ Error: Key not found`    | Song lacks the `sort_by` metadata.                          | Auto-sorting fails.                                   |
| `🤖❌ Error: No match`         | Song’s metadata doesn’t match any sub-playlist.             | Auto-sorting fails.                                   |
| `🤖❌ Error: {message}`        | Any error message.                                          | On error without message format.                      |
| `🤖ℹ️ To be sorted`             | Temporary holding area for songs that can’t be auto-sorted. | `sort_by` is set but no matching sub-playlist exists. |
| `🤖ℹ️ Que`                      | Temporary holding area for songs that can’t be auto-sorted. | `sort_by` is set but no matching sub-playlist exists. |
| `🤖ℹ️ {info}`                   | Any info message.                                           | On info without message format.                       |
| `🤖 {A} + {B}`                 | Intersection of songs from playlists `A` and `B`.           | User creates a playlist named `A + B`.                |

## 3. Workflows

### 3.1 Validation

**Trigger**: Manual or configured scheduled (e.g., daily).
**Steps**:

- For each **group**:
  1. Fetch all sub-playlists and their songs.
  2. For each song, count how many playlists it appears in within the group.
  3. If `count > max_per_song` (and `max_per_song != 0`):
     - Move the song to `🤖❌ Error: Max {n} exceeded` (or `🤖❌ Error: Duplicate` if `max_per_song=1`).

### 3.2 Generated Playlists

**Trigger**: When a user creates a `🤖` playlist with `+` in the name (e.g., `🤖 House + High Energy`).
**Steps**:

1. Detect the `🤖` and `+` in the playlist name.
2. Split the name into parts (e.g., `["House", "High Energy"]`).
3. Find the playlists matching each part.
4. Create a new playlist named `🤖 {Part1} + {Part2}`.
5. Populate it with songs that appear in **all** source playlists (intersection).

## 4. Music Service Integration

### 4.1 Generic Requirements

The music service must support:

- **API**: The service must support some sort of read/write access via an api.
- **Track Metadata**: Fetch metadata (e.g., `energy`, `genre`) for tracks.
- **Playlists**: Create, read, update, and delete playlists.
- **Playlist Metadata**: Read and write playlist descriptions.
- **Batch Operations**: Add/remove multiple tracks from playlists in a single request (if possible).

### 4.2 Folder alternatives

If a service does not natively supported folders use prefix-named playlists (e.g., `Genre: House`).

## 5. Spotify-Specific Section

### 5.1 Spotify Requirements

- **Spotify Web API**: The only API needed for this service.
- **Scopes**: Required scopes for Spotify:
  - `playlist-read-private`
  - `playlist-read-collaborative`
  - `playlist-modify-private`
  - `playlist-modify-public`
  - `user-library-read`

### 5.2 Spotify API Endpoints

| Feature             | Endpoint                                                              |
| ------------------- | --------------------------------------------------------------------- |
| Fetch playlists     | `GET /me/playlists`                                                   |
| Get playlist tracks | `GET /playlists/{playlist_id}/tracks`                                 |
| Get track metadata  | `GET /audio-features/{track_id}` (for `energy`, `danceability`, etc.) |
| Create playlist     | `POST /users/{user_id}/playlists`                                     |
| Add/remove tracks   | `POST/DELETE /playlists/{playlist_id}/tracks`                         |

### 5.3 Folders in Spotify

- Spotify supports **folders** as a way to organize playlists.
- Folders can be treated as **groups** in this specification.
- Spotify API does not support folders.

## 6. Other Music services

### YouTube Music

- Folders: Not natively supported. Use prefix-named playlists.
- Metadata: Limited native metadata. May require external sources or manual tagging.
- API: Use the YouTube Data API.

### Deezer, Tidal, SoundCloud

- Folders: Check service-specific documentation.
- Metadata: Varies by service. Use available track attributes.
- API: Use the respective service’s API.

## 7. Data Storage

- **In-Memory Cache**: For temporary storage of track metadata and group rules.
- **SQLite (Optional)**: For persistent storage of configuration, cached metadata, or user preferences.

## 8. Lightweight UI

- **Purpose**: Required for:
  - Music service authentication (e.g., Spotify OAuth2 flow).
  - Setup and configuration for non-power users.
- **Implementation**: Minimal web interface.

## 9. Future Ideas

- **Extensible**: Supports custom metadata keys and validation logic.
- **User playlist sorting**: For sub-playlists the description can define optional **song definitions** (e.g., `"energy>0.7"` for `High Energy`).

### Song Auto-Sorting

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

## 10. Answered Questions

1. **Dashboard Scope**: Should the dashboard be **read-only** or **interactive**?

- Dashboard should be interactive.

2. **Conflict Resolution**: If a song is in `🤖❌ Error: Duplicate`, should the backend **auto-resolve** it or leave it for manual resolution?

- Conflict resolution is handled manually, by the user.

3. **Folder Creation**: Should the backend **auto-create folders** for groups if they don’t exist?

- If confirmed or toggled on in the dashboard, the backend should auto-create folders.

4. **Error Playlist Naming**: Should error playlists (e.g., `🤖❌ Error: Duplicate`) be configurable or use a fixed naming scheme?

- Preconfigured for now.

## 10. Open Questions

1. **Generated Playlist Updates**: Should `🤖 {A} + {B}` playlists **auto-update** when `A` or `B` changes, or only on manual trigger?
2. **Multi-Service Support**: How should the service handle differences in API capabilities between music services (e.g., some may not support folders)?
3. **Data Persistence**: Should metadata or configuration be stored in a local database (e.g., SQLite) for offline access or performance?
4. **Rate Limiting**: How should the service handle rate limits imposed by music service APIs? Should it implement retries, caching, or user notifications?
