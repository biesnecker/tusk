# Tusk

A lightweight CLI client for Mastodon, written in Go.

## Features

- **Simple Authentication**: OAuth flow with automatic browser launch and localhost callback
- **Post Status**: Post text directly, use your `$EDITOR`, or pipe from stdin
- **Reply to Posts**: Reply to specific posts or your last posted status
- **Delete Posts**: Delete posts with confirmation (or `--force` to skip)
- **Content Warnings**: Add spoiler text to your posts
- **Visibility Control**: Choose post visibility (public, unlisted, private, direct)
- **Dry Run**: Preview what would be posted without actually posting
- **Secure Storage**: SQLite-based storage in platform-appropriate directories

## Installation

### Build from source

```bash
make build
```

### Install

```bash
make install
```

This will build a release binary and install it to `~/.local/bin/tusk`.

## Usage

### Authentication

Authenticate with your Mastodon instance:

```bash
tusk auth
```

You'll be prompted for your instance domain (e.g., `mastodon.social`), and your browser will open for authorization.

### Posting

Post a simple status:

```bash
tusk post "Hello, Mastodon!"
```

Compose in your editor:

```bash
tusk post -e
```

Pipe from stdin:

```bash
echo "Hello from the command line" | tusk post
cat status.txt | tusk post
```

### Replies

Reply to a specific status:

```bash
tusk post -r STATUS_ID "This is a reply"
```

Reply to your last posted status:

```bash
tusk post -R "Adding to my previous thought..."
```

### Image Uploads

Attach an image to your post:

```bash
tusk post -i /path/to/image.jpg --alt "Description of image" "Check out this photo!"
```

**Features:**
- **HEIC Support**: HEIC/HEIF images are automatically converted to JPG
- **EXIF Stripping**: All EXIF metadata is automatically removed for privacy
- **Alt Text**: You'll be prompted with a warning if you forget alt text (recommended for accessibility)
- **Supported formats**: JPG, PNG, HEIC/HEIF

Post without alt text (not recommended):

```bash
tusk post -i photo.jpg "My photo"
# You'll get a warning and can choose to proceed or cancel
```

### Visibility and Content Warnings

Post with custom visibility:

```bash
tusk post -v unlisted "This won't show in public timeline"
tusk post -v private "Only followers can see this"
```

Add a content warning:

```bash
tusk post -w "politics" "Here's my take on..."
```

Combine options:

```bash
tusk post -v unlisted -w "food" "I made the best sandwich today!"
```

### Deleting

Delete a specific status by ID (with confirmation):

```bash
tusk post -d STATUS_ID
```

Delete your last posted status:

```bash
tusk post -D
```

Force delete without confirmation:

```bash
tusk post -D -f
```

### Post History

Tusk maintains a stack of your posted statuses. When you delete a post, it's removed from the stack, and `-R` and `-D` will then operate on the next most recent post.

Clear the post history stack:

```bash
tusk clear
```

Skip confirmation:

```bash
tusk clear -f
```

### Dry Run

Preview what would be posted:

```bash
tusk post --dry-run "Test post"
tusk post -R --dry-run -v unlisted "Reply test"
```

### Logout

Revoke your access token and clear local data:

```bash
tusk logout
```

## Data Storage

Tusk stores configuration and tokens in platform-appropriate locations:

- **macOS**: `~/.local/share/tusk/tusk.db`
- **Linux**: `$XDG_DATA_HOME/tusk/tusk.db` or `~/.local/share/tusk/tusk.db`
- **Windows**: `%APPDATA%\tusk\tusk.db`

## Development

### Running Tests

```bash
go test ./...
```

### Building

Debug build:
```bash
make build
```

Release build (optimized):
```bash
make release
```

### Project Structure

```
tusk/
├── main.go                 # Entry point
├── cmd/                    # Command implementations
│   ├── root.go
│   ├── auth.go
│   ├── post.go
│   └── logout.go
├── internal/
│   ├── config/            # SQLite storage
│   ├── mastodon/          # Mastodon API client
│   ├── oauth/             # OAuth flow handler
│   └── output/            # Pretty terminal output
└── Makefile
```

## TODO

- [ ] Add media attachment support
- [ ] Add OS keychain integration for secure token storage
- [ ] Add multiple account support
- [ ] Add timeline viewing commands
- [ ] Add bookmarks/favorites
- [ ] Add search functionality

## License

MIT
