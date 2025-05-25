# GoferShell

**GoferShell** is a collection of shell-agnostic command-line utilities written in Go, designed to provide a consistent, efficient, and portable workflow across any terminal or shell environment. Originally crafted for Arch Linux and its derivatives, these tools are practical, minimal, and focused on real-world daily tasks—like checking for system updates, fast file downloads, and streamlined YouTube video management.

> **Why GoferShell?**  
> If you frequently switch between shells (fish, nushell, zsh, etc.) and want your custom command-line tools to "just work" everywhere, GoferShell is for you. Each utility is a standalone binary, requiring no shell-specific scripting or configuration.

---

## Table of Contents

- [Features & Utilities](#features--utilities)
- [Installation](#installation)
- [Usage](#usage)
  - [check_updates](#check_updates)
  - [dlfast](#dlfast)
  - [dlfast_batch](#dlfast_batch)
  - [ytmax](#ytmax)
  - [yt_batch](#yt_batch)
  - [ytstream](#ytstream)
- [Dependencies](#dependencies)
- [Building from Source](#building-from-source)
- [Contributing](#contributing)
- [License](#license)

---

## Features & Utilities

GoferShell includes the following command-line tools:

| Utility         | Description                                                                                  | Platform         |
|-----------------|---------------------------------------------------------------------------------------------|------------------|
| **check_updates**   | Check for official and AUR package updates on Arch Linux and derivatives.                    | Arch-based only  |
| **dlfast**          | Fast, resumable single-file downloader using `aria2`.                                       | Any Linux Distro              |
| **dlfast_batch**    | Batch download multiple files via `dlfast`.                                                 | Any Linux Distro              |
| **ytmax**           | Download a single YouTube video with preferred resolution and codec using `yt-dlp`.         | Any Linux Distro              |
| **yt_batch**        | Batch download multiple YouTube videos interactively.                                       | Any Linux Distro              |
| **ytstream**        | Stream YouTube (and similar) videos directly in `mpv` with preferred quality.               | Any Linux Distro              |

All utilities are designed to be portable and shell-agnostic—just drop the binaries in your `$PATH` and use them anywhere.

---

## Installation

### 1. Prerequisites

Ensure the following are installed on your system:

- **Go** (for building from source)
- **aria2** (for downloads)
- **yt-dlp** (for YouTube tools)
- **mpv** (for streaming)
- **pacman-contrib** and an AUR helper (`paru` or `yay`) for `check_updates` (Arch-based only)
- **paru** or **yay** (for AUR updates)
- **Go package:** `github.com/chzyer/readline` (fetched automatically for `yt_batch`)

### 2. Clone the Repository

```sh
git clone https://github.com/Evren-os/GoferShell.git
cd GoferShell
```

### 3. Build All Utilities

```sh
go mod init github.com/Evren-os/GoferShell
go get github.com/chzyer/readline
go build check_updates.go
go build dlfast.go
go build dlfast_batch.go
go build ytmax.go
go build yt_batch.go
go build ytstream.go
```

### 4. Install the Binaries

Copy the executables to a directory in your `$PATH` (e.g., `~/.local/bin/`):

```sh
mkdir -p ~/.local/bin/
cp check_updates dlfast dlfast_batch ytmax yt_batch ytstream ~/.local/bin/
chmod +x ~/.local/bin/check_updates ~/.local/bin/dlfast ~/.local/bin/dlfast_batch ~/.local/bin/ytmax ~/.local/bin/yt_batch ~/.local/bin/ytstream
```

Ensure `~/.local/bin` is in your `$PATH` (add to your shell config if needed):

```sh
export PATH="$HOME/.local/bin:$PATH"
```

---

## Usage

### check_updates

Check for available package updates (official + AUR) on Arch Linux.

```sh
check_updates [-no-ver]
```

- `-no-ver` : Omit version details from output.

**Example:**

```sh
check_updates
```

> **Note:** Requires `checkupdates` (from `pacman-contrib`) and an AUR helper (`paru` or `yay`).

---

### dlfast

Fast, resumable file downloader using `aria2`.

```sh
dlfast [-d target_directory_or_filepath] <URL>
```

- `-d` : Target directory or full file path. If omitted, downloads to current directory.

**Examples:**

```sh
dlfast http://example.com/file.zip
dlfast -d ~/Downloads/ http://example.com/file.zip
dlfast -d ~/Downloads/file.zip http://example.com/file.zip
```

---

### dlfast_batch

Batch download multiple files using `dlfast`.

```sh
dlfast_batch [-d target_directory] <URL1> [URL2 ...]
```

- `-d` : Download all files to this directory.

**Example:**

```sh
dlfast_batch -d ~/Downloads "http://example.com/file1.zip" "http://example.com/file2.zip"
```

---

### ytmax

Download a single YouTube video with preferred resolution and codec.

```sh
ytmax [--max-res RES] [--codec CODEC] [-d DEST] <URL>
```

- `--codec`   : Preferred codec (`av1` or `vp9`, default: `av1`)
- `-d`        : Output directory or full file path

**Examples:**

```sh
ytmax -d ~/Videos/ https://www.youtube.com/watch?v=video_id
ytmax -d ~/Videos/myfile.mkv https://www.youtube.com/watch?v=video_id
```

---

### yt_batch

Interactively batch download multiple YouTube videos.

```sh
yt_batch [-d directory]
```

- Prompts for a comma-separated list of URLs.
- `-d` : Download directory (optional).

**Example:**

```sh
yt_batch
# Enter video URLs (separated by commas): https://youtu.be/abc, https://youtu.be/xyz
```

---

### ytstream

Stream a YouTube (or similar) video directly in `mpv`.

```sh
ytstream [--max-res RES] [--codec CODEC] <URL>
```

- `--max-res` : Maximum resolution (default: 2160)
- `--codec`   : Preferred codec (`av1` or `vp9`, default: `av1`)

**Example:**

```sh
ytstream --max-res 720 --codec av1 https://www.youtube.com/watch?v=video_id
```

---

## Dependencies

| Utility         | Required Programs / Libraries                |
|-----------------|---------------------------------------------|
| check_updates   | `pacman-contrib`, `paru` or `yay`           |
| dlfast          | `aria2`                                     |
| dlfast_batch    | `aria2`, `dlfast` in `$PATH`                |
| ytmax           | `yt-dlp`, `aria2`                           |
| yt_batch        | `ytmax`, Go package `github.com/chzyer/readline` |
| ytstream        | `yt-dlp`, `mpv`                             |

Install dependencies using your package manager. For Arch-based systems:

```sh
sudo pacman -S go pacman-contrib aria2 yt-dlp mpv
```

---

## Building from Source

1. **Clone the repo and enter the directory:**

    ```sh
    git clone https://github.com/Evren-os/GoferShell.git
    cd GoferShell
    ```

2. **Initialize Go module and fetch dependencies:**

    ```sh
    go mod init github.com/Evren-os/GoferShell
    go get github.com/chzyer/readline
    ```

3. **Build all utilities:**

    ```sh
    go build check_updates.go
    go build dlfast.go
    go build dlfast_batch.go
    go build ytmax.go
    go build yt_batch.go
    go build ytstream.go
    ```

4. **Copy binaries to your `$PATH` as described above.**

---

## Contributing

GoferShell is a personal workflow project, but contributions, suggestions, and improvements are welcome!  
Feel free to open issues or submit pull requests.

---

## License

This project is licensed under the MIT License.  
See [LICENSE](LICENSE) for details.

---

**GoferShell** - Minimal, portable, and shell-agnostic CLI tools for your daily workflow.
