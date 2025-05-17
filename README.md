# GoferShell

GoferShell is a collection of shell-agnostic Go utilities designed to enhance your terminal experience. Originally crafted for Arch Linux and its derivatives, these tools—covering package updates, speedy downloads, and a YouTube toolkit—are compiled to work seamlessly in any shell environment.

## Overview

GoferShell includes the following utilities:

- **check_updates**: Check for package updates on Arch Linux, including official repositories and the AUR.
- **dlfast**: Download files efficiently using `aria2`.
- **dlfast_batch**: Batch download multiple files using `dlfast`.
- **ytmax**: Download YouTube videos with customizable resolution and codec options.
- **yt_batch**: Batch download multiple YouTube videos interactively.
- **ytstream**: Stream YouTube videos with customizable resolution and codec options.

## Prerequisites

Ensure the following dependencies are installed on your system before proceeding:

```
go pacman-contrib paru/yay aria2 yt-dlp mpv
```

The `yt_batch` utility also requires the `github.com/chzyer/readline` Go package, which is fetched during installation.

## Installation and Usage

### Installation

Follow these steps to install the GoferShell utilities:

1. **Clone the repository**:

   ```sh
   git clone https://github.com/Evren-os/GoferShell.git
   ```

2. **Navigate to the repository directory**:

   ```sh
   cd GoferShell
   ```

3. **Initialize a Go module**:

   ```sh
   go mod init github.com/Evren-os/GoferShell
   ```

4. **Fetch the required Go dependency**:

   ```sh
   go get github.com/chzyer/readline
   ```

5. **Build the utilities**:

   ```sh
   go build *.go
   ```

   This command compiles each `.go` file into its respective executable.

6. **Copy the executables to a directory in your PATH** (e.g., `~/.local/bin/`):

   ```sh
   mkdir -p ~/.local/bin/
   cp check_updates dlfast dlfast_batch ytmax yt_batch ytstream ~/.local/bin/
   ```

7.  **Make the utilities executable**:

    ```sh
    chmod +x ~/.local/bin/check_updates ~/.local/bin/dlfast ~/.local/bin/dlfast_batch ~/.local/bin/ytmax ~/.local/bin/yt_batch ~/.local/bin/ytstream
    ```
   
8. **Verify your PATH**: Ensure `~/.local/bin/` is in your shell’s PATH. If not, add it by appending this line to your shell configuration file (e.g., `~/.bashrc`, `~/.zshrc`):

   ```sh
   export PATH="$HOME/.local/bin:$PATH"
   ```

   Then, reload the configuration:

   ```sh
   source ~/.bashrc  # or ~/.zshrc, depending on your shell
   ```

### Usage

#### check_updates

Checks for available package updates on Arch Linux, including official and AUR packages.

- **Usage**:

  ```sh
  check_updates [-no-version]
  ```

- **Options**:
  - `-no-version`: Omit version details from the output

- **Example**:

  ```sh
  check_updates
  ```

- **Note**: This utility is tailored for Arch Linux and requires `checkupdates` and an AUR helper.

#### dlfast

Downloads files using `aria2c` for enhanced speed and reliability.

- **Usage**:

  ```sh
  dlfast <URL> [target_directory_or_filepath]
  ```

- **Examples**:
  - Download to the current directory:

    ```sh
    dlfast http://example.com/file.zip
    ```

  - Download to a specific directory:

    ```sh
    dlfast http://example.com/file.zip /path/to/directory/
    ```

  - Download to a specific file path:

    ```sh
    dlfast http://example.com/file.zip ~/Downloads/file.zip
    ```

#### dlfast_batch

Batch downloads multiple files using `dlfast`.

- **Usage**:

  ```sh
  dlfast_batch [-d target_directory] url1 [url2 ...]
  ```

- **Options**:
  - `-d target_directory`: specify a directory for all downloads

- **Example**:

  ```sh
  dlfast_batch -d ~/Downloads "http://example.com/file1.zip" "http://example.com/file2.zip"
  ```

#### ytmax

Downloads YouTube videos using `yt-dlp`, with options for resolution and codec.

- **Usage**:

  ```sh
  ytmax [options] URL
  ```

- **Options**:
  - `--max-res RES`: maximum resolution in pixels (default: 2160)
  - `--codec CODEC`: preferred codec (`av1` or `vp9`, default: `av1`)

- **Example**:

  ```sh
  ytmax --max-res 1080 --codec vp9 https://www.youtube.com/watch?v=video_id
  ```

#### yt_batch

Batch downloads multiple YouTube videos interactively.

- **Usage**:

  Run the command and enter URLs separated by commas when prompted:

  ```sh
  yt_batch
  ```

- **Example**:

  ```sh
  yt_batch
  Enter video URLs (separated by commas). Press [ENTER] when done: https://www.youtube.com/watch?v=video1, https://www.youtube.com/watch?v=video2
  ```

#### ytstream

Streams YouTube videos using `yt-dlp` and `mpv`, with customizable resolution and codec options.

- **Usage**:

  ```sh
  ytstream [options] URL
  ```

- **Options**:
  - `--max-res RES`: maximum resolution in pixels (default: 2160)
  - `--codec CODEC`: preferred codec (`av1` or `vp9`, default: `av1`)

- **Example**:

  ```sh
  ytstream --max-res 720 --codec av1 https://www.youtube.com/watch?v=video_id
  ```
