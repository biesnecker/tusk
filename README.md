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

Post a simple status (the `post` command is the default, so you can omit it):

```bash
tusk "Hello, Mastodon!"
# or
tusk post "Hello, Mastodon!"
```

Compose in your editor:

```bash
tusk -e
```

Pipe from stdin:

```bash
echo "Hello from the command line" | tusk
cat status.txt | tusk
```

### Replies

Reply to a specific status:

```bash
tusk -r STATUS_ID "This is a reply"
```

Reply to your last posted status:

```bash
tusk -R "Adding to my previous thought..."
```

Interactive TUI to select which post to reply to:

```bash
tusk --reply-tui "This is my reply"
```

In reply-tui mode:
- Use arrow keys or `j`/`k` to navigate
- Press `enter` or `space` to select the post to reply to
- Press `s` to sync latest posts from Mastodon
- Press `q` to quit without selecting

### Image Uploads

Attach an image to your post:

```bash
tusk -i /path/to/image.jpg --alt "Description of image" "Check out this photo!"
```

**Features:**
- **HEIC Support**: HEIC/HEIF images are automatically converted to JPG
- **EXIF Stripping**: All EXIF metadata is automatically removed for privacy
- **Alt Text**: You'll be prompted with a warning if you forget alt text (recommended for accessibility)
- **Supported formats**: JPG, PNG, HEIC/HEIF

Post without alt text (not recommended):

```bash
tusk -i photo.jpg "My photo"
# You'll get a warning and can choose to proceed or cancel
```

### Visibility, Content Warnings, and Language

Post with custom visibility:

```bash
tusk -v unlisted "This won't show in public timeline"
tusk -v private "Only followers can see this"
```

Add a content warning:

```bash
tusk -w "politics" "Here's my take on..."
```

Specify the language of your post (using ISO 639 language codes):

```bash
tusk -l es "¡Hola mundo!"
tusk -l ja "こんにちは"
tusk -l fr "Bonjour le monde!"
```

Combine options:

```bash
tusk -v unlisted -w "food" "I made the best sandwich today!"
tusk -l de -v unlisted "Ein Post auf Deutsch"
```

### Editing

Edit a specific status by ID:

```bash
tusk edit STATUS_ID "Updated text"
```

Edit your most recent post:

```bash
tusk edit --latest "Updated text"
```

Interactive TUI selection mode:

```bash
tusk edit --tui "Updated text"
```

Edit with $EDITOR:

```bash
tusk edit STATUS_ID -e
tusk edit --latest -e
```

Pipe content to edit:

```bash
echo "Updated content" | tusk edit STATUS_ID
```

Edit with all posting flags:

```bash
tusk edit STATUS_ID -v unlisted -w "updated cw" --lang es "Texto actualizado"
tusk edit --latest -i new_image.jpg --alt "New image description" "Updated with new image"
```

In edit TUI mode:
- Use arrow keys or `j`/`k` to navigate
- Press `enter` or `space` to select the post to edit
- Press `s` to sync latest posts from Mastodon
- Press `q` to quit

### Deleting

Delete a specific status by ID (with confirmation):

```bash
tusk delete STATUS_ID
```

Delete your most recent post:

```bash
tusk delete --latest
```

Force delete without confirmation:

```bash
tusk delete STATUS_ID -f
tusk delete --latest -f
```

Interactive TUI selection mode:

```bash
tusk delete --tui
```

In delete TUI mode:
- Use arrow keys or `j`/`k` to navigate
- Press `space` to toggle selection
- Press `s` to sync latest posts from Mastodon
- Press `d` to delete selected posts (with confirmation)
- Press `q` to quit

### Post History

Tusk maintains a stack of your posted statuses. When you delete a post, it's removed from the stack, and `-R` and `delete --latest` will then operate on the next most recent post.

View your latest post:

```bash
tusk latest
```

This shows the post that `-R` (reply to last) and `delete --latest` would operate on.

Sync your recent posts from Mastodon to local history:

```bash
tusk sync
```

Sync a specific number of posts (default is 50, max 100):

```bash
tusk sync -n 100
```

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
tusk --dry-run "Test post"
tusk -R --dry-run -v unlisted "Reply test"
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
