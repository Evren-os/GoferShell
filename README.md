# GoferShell

Personal command-line utilities written in Go for my own system management and media handling needs. These are leisure-time tools designed with my personal preferences for aria2c and yt-dlp workflows.

## Utilities

| Utility | Description | Requirements |
|---------|-------------|--------------|
| `check_updates` | Check for package updates on Arch Linux (official + AUR) | `pacman-contrib`, `paru` or `yay` |
| `dlfast` | High-performance file downloader using aria2c | `aria2c` |
| `ytmax` | Download YouTube videos with quality preferences | `yt-dlp`, `aria2c` |
| `ytstream` | Stream YouTube videos directly in mpv | `yt-dlp`, `mpv` |

## Installation

### Prerequisites

Install required dependencies:

```bash
# Arch Linux
sudo pacman -S go pacman-contrib aria2 yt-dlp mpv paru

# Or with yay instead of paru
sudo pacman -S go pacman-contrib aria2 yt-dlp mpv yay
```

### Build and Install

```bash
# Clone repository
git clone https://github.com/Evren-os/GoferShell.git
cd GoferShell

# Initialize Go module
go mod init github.com/Evren-os/GoferShell

# Build all utilities
go build check_updates.go
go build dlfast.go
go build ytmax.go
go build ytstream.go

# Install to local bin directory
mkdir -p ~/.local/bin
cp check_updates dlfast ytmax ytstream ~/.local/bin/
chmod +x ~/.local/bin/*

# Add to PATH (add to your shell config)
export PATH="$HOME/.local/bin:$PATH"
```

## Usage

### check_updates

Check for available package updates on Arch Linux.

```bash
check_updates [-no-ver]
```

**Options:**
- `-no-ver`: Hide version information in output

**Example:**
```bash
check_updates
```

### dlfast

Download files with high performance using aria2c.

```bash
dlfast [options] <URL> [URL2 ...]
```

**Options:**
- `-d <path>`: Target directory for downloads
- `-max-speed <speed>`: Limit download speed (e.g., 1M, 500K)
- `-timeout <seconds>`: Download timeout (default: 60)
- `-parallel <num>`: Number of parallel downloads (default: 3)
- `-quiet`: Suppress progress output

**Examples:**
```bash
dlfast https://example.com/file.zip
dlfast -d ~/Downloads https://example.com/file.zip
dlfast -max-speed 1M -parallel 2 url1 url2 url3
```

### ytmax

Download YouTube videos with quality and codec preferences.

```bash
ytmax [options] <URL> [URL2 ...]
```

**Options:**
- `-codec <name>`: Preferred codec (`av1` or `vp9`, default: `av1`)
- `-d <path>`: Output directory or full file path
- `-socm`: Download in MP4 format optimized for social media
- `-cookies-from <browser>`: Use browser cookies (e.g., `firefox`, `chrome`)
- `-p <num>`: Parallel downloads for batch mode (default: 4)

**Examples:**
```bash
# Single download
ytmax -codec vp9 -d ~/Videos https://youtu.be/VIDEO_ID

# Batch download
ytmax -d ~/Videos -p 6 "URL1" "URL2" "URL3"

# With browser cookies
ytmax --cookies-from firefox "URL1" "URL2"
```

### ytstream

Stream YouTube videos directly in mpv player.

```bash
ytstream [options] <URL>
```

**Options:**
- `-max-res <resolution>`: Maximum resolution (default: 2160)
- `-codec <name>`: Preferred codec (`av1` or `vp9`, default: `av1`)
- `-cookies-from <browser>`: Use browser cookies

**Example:**
```bash
ytstream -max-res 1080 -cookies-from firefox https://youtu.be/VIDEO_ID
```

## Features

- **Personal workflow**: Designed around my specific aria2c and yt-dlp preferences
- **Shell-agnostic**: Standalone binaries that work in any shell
- **High performance**: Optimized for speed with aria2c integration
- **Quality control**: Configurable video quality and codec preferences
- **Batch processing**: Support for downloading multiple files/videos concurrently
- **Error handling**: Robust error recovery and signal handling

## License

MIT License - see [LICENSE](LICENSE) file for details.