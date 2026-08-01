# UltraCopySam

[![CI](https://github.com/cahosoft2/UltraCopySam/actions/workflows/ci.yml/badge.svg)](https://github.com/cahosoft2/UltraCopySam/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/cahosoft2/UltraCopySam)](https://github.com/cahosoft2/UltraCopySam/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/cahosoft2/UltraCopySam/total)](https://github.com/cahosoft2/UltraCopySam/releases)
[![License: 0BSD](https://img.shields.io/badge/license-0BSD-blue.svg)](LICENSE)
[![Platform: Windows](https://img.shields.io/badge/platform-Windows%20x64-0078D6)](https://github.com/cahosoft2/UltraCopySam/releases/latest)

*[Versión en español](README.es.md)*

A Windows command-line tool that copies an entire directory tree to another
location, **overwriting whatever is there without asking**, as fast as the
hardware allows.

It is written in Go, but it does not use the standard library to copy: it calls
the native Win32 API directly, so **the Windows kernel moves the bytes** and
they never pass through the program's buffers.

```powershell
UltraCopySam.exe "D:\projects" "E:\backup\projects"
```

```text
1284 files | 3491.84 MB | 812.44 MB/s
```

---

## Contents

- [Download](#download)
- [Windows SmartScreen and antivirus](#windows-smartscreen-and-antivirus)
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Options](#options)
- [Examples](#examples)
- [Incremental copies](#incremental-copies)
- [Benchmarks vs robocopy](#benchmarks-vs-robocopy)
- [Should you use this instead of robocopy?](#should-you-use-this-instead-of-robocopy)
- [Double quotes and paths with spaces](#double-quotes-and-paths-with-spaces)
- [External USB drives](#external-usb-drives)
- [What it shows while copying](#what-it-shows-while-copying)
- [Behaviour](#behaviour)
- [Validation](#validation)
- [How it gets its speed](#how-it-gets-its-speed)
- [Choosing the worker count](#choosing-the-worker-count)
- [Limitations](#limitations)
- [Building](#building)
- [Code layout](#code-layout)
- [License](#license)

---

## Download

Get the latest build from the
**[Releases page](https://github.com/cahosoft2/UltraCopySam/releases)**, or
directly:

**[⬇ UltraCopySamV004.exe](https://github.com/cahosoft2/UltraCopySam/releases/download/v0.0.4/UltraCopySamV004.exe)**
— Windows 64-bit · 1.91 MB · no installer, no dependencies

Verify the download against its SHA256 hash:

```powershell
Get-FileHash UltraCopySamV004.exe -Algorithm SHA256
```

```text
BC870DDA649ABE6ED8B22F628C3A651785962DCD5F7747E2B4B761CD3963A472
```

> [!NOTE]
> Windows will show a *"Windows protected your PC"* warning the first time you
> run the downloaded file. This is expected and takes one command to fix — see
> [Windows SmartScreen and antivirus](#windows-smartscreen-and-antivirus).

---

## Windows SmartScreen and antivirus

When you run the freshly downloaded `.exe`, Windows shows a blue screen reading
**"Windows protected your PC"**, offering only a *Don't run* button.

**This does not mean the program is dangerous.** It happens because the
executable is not signed with a code-signing certificate, which costs 300-600
USD per year. Windows flags every unsigned binary downloaded from the internet,
regardless of what it does.

There are three ways around it, from quickest to safest.

### Option 1 — Unblock the file (recommended)

When you download any file, Windows attaches an invisible marker called the
*Mark of the Web* that means "this came from the internet". Just remove it:

```powershell
Unblock-File .\UltraCopySamV004.exe
```

From then on the program runs normally, with no further warnings.

You can also do it with the mouse: right-click the file → **Properties** → tick
**Unblock** at the bottom of the *General* tab → *OK*.

### Option 2 — Run it anyway

On the warning screen itself, click **More info** and then the **Run anyway**
button that appears below. Windows remembers the decision for that file.

### Option 3 — Build it yourself (safest)

If you would rather not trust a downloaded executable — a reasonable stance for
a tool that overwrites files — build it from source. A binary compiled on your
own machine **never triggers SmartScreen**, because it did not come from the
internet:

```powershell
git clone https://github.com/cahosoft2/UltraCopySam.git
cd UltraCopySam
go build -trimpath -ldflags "-s -w" -o UltraCopySam.exe .
```

Requires [Go](https://go.dev/dl/). The whole source is about 1,300 lines across
6 files — auditable in a few minutes.

### About antivirus false positives

Unsigned Go binaries trigger false positives fairly often, because the compiler
links everything statically into a single file and several heuristic engines
treat that pattern as suspicious.

If your antivirus blocks the file, first check its SHA256 hash (see
[Download](#download)): if it matches the published one, the file is exactly
what was compiled and has not been tampered with. Then add an exclusion, or use
Option 3.

### Why isn't it signed?

Signing a Windows executable requires a certificate from a recognised
authority, with identity verification and annual renewal. For a free
open-source utility the cost is hard to justify.

The available routes, should the project grow:

| Route | Approximate cost | Removes SmartScreen? |
| --- | --- | --- |
| [Azure Trusted Signing](https://azure.microsoft.com/products/trusted-signing) | ~10 USD/month | Yes, immediately |
| EV certificate (DigiCert, Sectigo…) | 300-600 USD/year | Yes, immediately |
| [SignPath Foundation](https://signpath.org/) (free for OSS) | Free | Gradually, as reputation builds |
| Publishing on the Microsoft Store | ~19 USD, one-off | Yes (Store apps bypass SmartScreen) |

---

## Features

- **Low-level copying** via `CopyFileExW`: the kernel transfers data from one
  handle to another without going through user space.
- **True parallelism**: separate worker pools walk the tree and copy at the same
  time, without waiting for the listing to finish before copying starts.
- **Bounded memory**: peak usage stays flat no matter how large the tree —
  measured at 16 MB for both 20,000 and 300,000 files.
- **Always overwrites**, no confirmations — even if the destination file is
  marked *read-only*, *hidden* or *system*.
- **Incremental mode** (`-u`): skips files whose size and timestamp already
  match at the destination.
- **No `MAX_PATH` limit**: handles paths longer than 260 characters through the
  `\\?\` extended prefix.
- **Fault tolerant**: a locked or permission-denied file does not abort the
  copy; it is reported and the rest continues.
- **Up-front validation** of arguments and paths, with explanatory messages.
- **A single executable** with no dependencies: no Go runtime, no extra DLLs.

---

## Installation

### Automatic install (recommended)

Run this in PowerShell:

```powershell
irm https://raw.githubusercontent.com/cahosoft2/UltraCopySam/main/install.ps1 | iex
```

The installer takes care of everything:

1. Downloads the latest release from GitHub.
2. Prints its SHA256 hash so you can verify it.
3. Installs it to `%LOCALAPPDATA%\Programs\UltraCopySam`.
4. **Unblocks** it so SmartScreen stays out of the way (`Unblock-File`).
5. Adds it to the user `PATH`.
6. Checks that the executable responds.

**No administrator rights required** — everything lands in your user profile.
Open a new terminal afterwards so the `PATH` change takes effect.

> [!TIP]
> If you would rather read the script before running it — a good habit with any
> installer from the internet — open it first: [install.ps1](install.ps1). You
> can also download and run it separately:
>
> ```powershell
> irm https://raw.githubusercontent.com/cahosoft2/UltraCopySam/main/install.ps1 -OutFile install.ps1
> notepad install.ps1
> .\install.ps1
> ```

#### Installer options

| Parameter | Description |
| --- | --- |
| `-Version v0.0.4` | Install a specific version instead of the latest |
| `-InstallDir "D:\tools"` | Change the install folder |
| `-FromFile ".\UltraCopySam.exe"` | Install from a local file, no download |
| `-NoPath` | Do not modify `PATH` |
| `-Uninstall` | Uninstall: remove the folder and clean up `PATH` |

```powershell
# Install elsewhere, leaving PATH alone
.\install.ps1 -InstallDir "D:\tools\UltraCopySam" -NoPath

# Uninstall
.\install.ps1 -Uninstall
```

> [!NOTE]
> If PowerShell blocks the script because of its execution policy, run it like
> this (affects only that process, it does not change your system settings):
>
> ```powershell
> powershell -ExecutionPolicy Bypass -File .\install.ps1
> ```

### Manual install

Download `UltraCopySam.exe` from
[Releases](https://github.com/cahosoft2/UltraCopySam/releases) or build it (see
[Building](#building)), unblock it and put it wherever you like:

```powershell
Unblock-File .\UltraCopySamV003.exe
```

To call it as `UltraCopySam` from any folder, add its directory to `PATH`:

```powershell
# Current session only
$env:PATH += ";D:\tools\UltraCopySam"
```

To make it permanent, the safest route is the installer with `-FromFile`, which
writes `PATH` without expaUnblock-File .\UltraCopySamV004.exe
```

Or pass it directly to `install.ps1`:

```powershell
.\install.ps1 -FromFile ".\UltraCopySamV004.exe" -InstallDir "D:\tools\UltraCopySam"
```

Without it on the `PATH` you must call it by full path, or with `.\` when you
are sitting in its folder (in PowerShell the `.\` is mandatory):

```powershell
.\UltraCopySam.exe "D:\source" "E:\destination"
```

**Requirements:** 64-bit Windows. No administrator rights needed, unless the
folders involved require them.

---

## Usage

```text
UltraCopySam [options] "<source-directory>" "<destination-directory>"
```

The **contents** of the source are copied into the destination, not the source
folder itself:

```text
UltraCopySam "D:\dev\old" "E:\backup"

D:\dev\old\project\x.txt   ->   E:\backup\project\x.txt
```

To keep the folder name, include it in the destination:

```text
UltraCopySam "D:\dev\old" "E:\backup\old"

D:\dev\old\project\x.txt   ->   E:\backup\old\project\x.txt
```

**Exit codes:**

| Code | Meaning |
| --- | --- |
| `0` | Everything copied without errors |
| `1` | The copy finished, but some file failed |
| `2` | Bad usage: missing arguments or an invalid path |

---

## Options

| Option | Description |
| --- | --- |
| `-r` | Resume an interrupted copy using `.ucsam-state`, skipping completed subtrees and replacing partial files. |
| `-u` | Skip files whose size and timestamp already match at the destination. See [Incremental copies](#incremental-copies). |
| `-w N` | Number of parallel copies. Defaults to twice the CPU core count. |
| `-v` | List every copied file instead of showing the progress line. |
| `-q` | Quiet: no progress, no summary (errors are still shown). |
| `-L` | Follow symbolic links and junctions. Skipped by default. |
| `-cola N` | Files that may sit queued, 4,096 by default. This is what bounds memory use; you rarely need to touch it. |

> [!IMPORTANT]
> Options go **before** the paths. `UltraCopySam -u "D:\a" "E:\b"` works;
> `UltraCopySam "D:\a" "E:\b" -u` does not, and the program tells you so.

---

## Examples

### Basic copy

```powershell
UltraCopySam "D:\projects" "E:\backup\projects"
```

### Paths with spaces

Always in double quotes.

```powershell
UltraCopySam "D:\My Documents\Accounts" "E:\Backup\Accounts"
```

### Incremental backup

```powershell
UltraCopySam -u -w 4 "D:\dev" "E:\backup\dev"
```

### Resume interrupted copy

```powershell
UltraCopySam -r "D:\projects" "E:\backup\projects"
```

### To an external USB drive

Lower the parallelism — see [External USB drives](#external-usb-drives).

```powershell
UltraCopySam -w 4 "D:\dev\old" "E:\backup"
```

### From a network share

```powershell
UltraCopySam "\\server\share\data" "D:\local\data"
```

### List every file copied

```powershell
UltraCopySam -v "D:\source" "E:\destination"
```

### Quiet mode, logging only errors

```powershell
UltraCopySam -q "D:\source" "E:\destination" 2> errors.log
```

### Check for failures from a script

```powershell
UltraCopySam "D:\source" "E:\destination"
if ($LASTEXITCODE -ne 0) {
    Write-Host "The copy finished with errors" -ForegroundColor Red
}
```

### Daily backup into a dated folder

```powershell
$date = Get-Date -Format "yyyy-MM-dd"
UltraCopySam -u -w 4 "D:\dev" "E:\backups\$date\dev"
```

---

## Incremental copies

With `-u`, before copying each file it checks whether the destination already
holds one with **the same size and the same modification time**. If both match,
the file is skipped.

```powershell
UltraCopySam -u -w 4 "D:\dev" "E:\backup\dev"
```

```text
0 files, 41 directories, 0.00 MB in 20ms (0.00 MB/s)
20000 unchanged files, 7.63 MB that did not have to be rewritten
```

### When to use it

Any time you **repeat** a copy onto an already-populated destination: periodic
backups, syncing an external drive, or resuming an interrupted copy. The gain is
large because the real cost of copying is not moving the bytes — it is creating
each file at the destination.

Measured on a 20,000-file tree:

| Scenario | Without `-u` | With `-u` |
| --- | --- | --- |
| First copy (empty destination) | 15.7 s | 14.3 s |
| Second pass (nothing changed) | 10.3 s | **0.05 s** |

The first copy is **not penalised**: when a destination folder has just been
created it is empty by definition, so it is never listed. The second pass is
roughly **200× faster**.

### How it checks without wasting time

Source metadata already comes for free with the directory walk. Destination
metadata is fetched with **one directory listing per folder**, not one query per
file — a single system call per directory, folded into the same overlapped walk,
with no separate analysis pass.

### Comparison accuracy

On **NTFS the timestamp comparison is exact** (100 ns resolution). The 2-second
tolerance is applied *only* when the destination is FAT or exFAT, which round
timestamps to that granularity. The filesystem is detected automatically.

> [!WARNING]
> `-u` treats a file as identical when **size and timestamp** match. If some
> program modifies a file while keeping both exactly the same, the change goes
> undetected. This is practically impossible by accident, but if you need
> absolute certainty that the destination mirrors the source, run without `-u`:
> the normal mode always rewrites.

---

## Benchmarks vs robocopy

Honest numbers, including where UltraCopySam loses. Measured on a single NVMe
SSD, destination wiped before every run, against `robocopy /E /MT` (its
multi-threaded mode). Median of 3 runs, except the small-file scenario, which
uses 7 alternating runs because that is where the two tools are closest.

| Scenario | robocopy `/MT` | UltraCopySam | Winner |
| --- | --- | --- | --- |
| 20,000 small files (7.6 MB) | 10.70 s | **10.44 s** | Tie (within noise) |
| 8 large files (2,000 MB) | 1.44 s | **0.87 s** | UltraCopySam, **1.65× faster** |
| Mixed: 5,006 files (734 MB) | **2.90 s** | 3.17 s | robocopy, by 9% |
| Second pass, nothing changed | 0.03 s | 0.02 s | Tie |

**Reading the results:**

- With **large files** UltraCopySam is clearly ahead, thanks to
  `COPY_FILE_NO_BUFFERING` on files of 32 MiB or more.
- With **many small files** the two are level. Across 7 alternating runs the
  medians land 2.4% apart, which is well inside the run-to-run noise.
- For **repeated backups** both are equally fast, because neither rewrites what
  has not changed.

### How the small-file gap was closed

Version 0.0.2 was **21% slower** than robocopy on small files. Three plausible
explanations were tested with a standalone prototype, and all three turned out
to be wrong:

| Hypothesis | Result |
| --- | --- |
| `CopyFileExW` is too heavy for small files; manual `ReadFile`/`WriteFile` would be faster | **Wrong** — within 0.5% of each other |
| The `\\?\` extended paths cost extra | **Wrong** — marginally *faster* |
| The `sync.Cond` work queue is slower than a Go channel | **Wrong** — the queue beat the channel |

The real cause turned up while fixing something else entirely: a single worker
pool handled **both** walking and copying, so a worker busy listing a directory
was a worker not copying. Splitting them into two pools — 4 walkers and `-w`
copiers — closed the gap and **halved the time on large trees** (300,000 files:
127 s → 60 s).

Reproduce them yourself with the script in
[bench/bench.ps1](bench/bench.ps1).

---

## Should you use this instead of robocopy?

`robocopy` ships with Windows, is battle-tested and, as the numbers above show,
is faster for small files. It is a great tool and you should not switch away
from it for speed alone.

Where UltraCopySam is worth it:

- **Large files** — measurably faster.
- **Simplicity** — two paths and you are done, versus robocopy's ~80 switches.
  There is no `/E /MT:32 /NFL /NDL /NJH /NJS` to memorise.
- **Clear error messages** instead of numeric exit codes you have to look up.
  It tells you what went wrong and how to fix it, including the classic Windows
  trap of a trailing backslash inside quotes.
- **Readable source** — about 1,300 lines of Go you can audit, instead of a
  closed-source binary.
- **Predictable behaviour** — it never deletes anything at the destination.
  There is no `/MIR` that can wipe a folder by mistake.

If you need mirroring, filters, retries or NTFS permission copying, use
robocopy: this tool deliberately does not cover those.

---

## Double quotes and paths with spaces

Always wrap paths in **double quotes**. That is what lets a path with spaces
arrive as a single argument:

```powershell
UltraCopySam "D:\My Data\source" "E:\Backup\destination"
```

> [!WARNING]
> **Never leave a trailing backslash inside the quotes.**
>
> On Windows, `"D:\source\"` makes the `\` **escape the closing quote**, so both
> paths merge into a single argument and the command fails. This is a classic
> Windows command-line trap, not a flaw in this tool.
>
> ```text
> ✗  UltraCopySam "D:\source\" "E:\dest\"     <- wrong
> ✓  UltraCopySam "D:\source"  "E:\dest"      <- right
> ✓  UltraCopySam "D:\source\\" "E:\dest\\"   <- also valid
> ```
>
> UltraCopySam detects this specific case and explains what happened, instead of
> failing with a cryptic message.

**Relative paths** are also accepted, resolved against the current directory:

```powershell
cd D:\dev
UltraCopySam ".\old" "E:\backup\old"
```

---

## External USB drives

Copying to a USB drive is where configuration matters most. These
recommendations are ordered by actual impact.

### 1. Enable write caching (biggest win)

Windows configures removable drives for *"Quick removal"*, which **disables
write caching**: every write goes straight to the disk. Changing this usually
helps more than any tuning of the tool itself.

```text
Device Manager
  └─ Disk drives
       └─ (your USB drive) → Properties
            └─ "Policies" tab → select "Better performance"
```

> [!CAUTION]
> With this enabled you **must** always use *"Safely Remove Hardware"* before
> unplugging the drive. Yanking it out can lose data still sitting in cache.

### 2. Lower the worker count

The default (twice the core count, typically 16-32) targets internal NVMe drives
and is **excessive** for USB:

```powershell
UltraCopySam -w 4 "D:\source" "E:\dest"     # mechanical USB drive (HDD)
UltraCopySam -w 8 "D:\source" "E:\dest"     # SSD in a USB enclosure
```

A mechanical drive has a single physical head: too many concurrent writes force
it to seek back and forth across the platter and throughput **drops** instead of
rising. If you hear continuous clicking, go down to `-w 2`.

On an external SSD the USB bus is the limit, so going beyond 8 gains nothing.

If you do not know what is inside the enclosure, start with `-w 4`: on an SSD
you lose a little speed, whereas 32 workers on a mechanical drive really do
sink throughput.

### 3. Check the port and cable

A USB 2.0 port caps you at roughly **35 MB/s** and no setting fixes that. Look
for USB 3.0 or later — usually blue inside or marked `SS` (*SuperSpeed*).

| Standard | Approximate real speed |
| --- | --- |
| USB 2.0 | 35 MB/s |
| USB 3.0 / 3.1 Gen1 | 400-450 MB/s |
| USB 3.2 Gen2 | ~1 GB/s |

A poor or overly long cable can make the drive negotiate a slower link than it
supports.

### 4. Format the destination as NTFS

| Filesystem | Recommendation |
| --- | --- |
| **NTFS** | ✅ Recommended. No practical size limit, supports `\\?\` long paths, preserves file attributes |
| exFAT | ⚠️ Works, but degrades with large numbers of small files |
| FAT32 | ❌ **No files larger than 4 GB**: they will fail, though the rest of the copy continues |

### 5. Have realistic expectations with small files

When copying development folders (`node_modules`, `.git`, `vendor`), the time is
**not** spent moving bytes: it goes into creating each file — allocating its MFT
record, committing the journal, closing the handle. Those are milliseconds per
file that the USB bus cannot parallelise beyond what the workers already do.

| Scenario | Typical USB 3.0 speed |
| --- | --- |
| Large files (video, ISOs, backups) | 100-110 MB/s on HDD; 400+ MB/s on SSD |
| Thousands of small files | 5-15 MB/s |

That drop comes from the hardware and the filesystem, not the program. If the
folder holds a lot of regenerable content (`node_modules`, `bin`, `obj`,
`target`), copying it as a single compressed archive is often faster than
copying the loose files.

### 6. Stop the drive from sleeping

Windows may suspend a USB drive during a long copy, causing intermittent errors.
Under **Power Options → Change advanced settings → Hard disk**, set *"Turn off
hard disk after"* to `0` (never) while the copy runs.

---

## What it shows while copying

While copying, UltraCopySam refreshes **a single line** every 250 ms:

```text
1284 files | 3491.84 MB | 812.44 MB/s
```

When it finishes it prints the summary:

```text
5327 files, 214 directories, 13137.92 MB in 16.204s (811.55 MB/s)
```

plus, where relevant, how many entries were skipped and how many errors occurred.

Useful details:

- Volumes are **always in MB**, never scaled to GB. A fixed unit keeps the
  number from switching scale mid-copy, which is exactly what makes it hard to
  tell at a glance whether throughput is rising or falling.
- **Progress goes to `stderr`** and the **summary to `stdout`**, so you can
  redirect the result to a file without it filling up with flickering lines.
- `-v` turns off the progress line and prints every copied file instead.
- `-q` shows nothing but errors.
- On very short copies (under 250 ms) you only see the final summary.
- The counter **adds each file once it completes**, not while it is being
  copied. With a single very large file the figure stays at `0.00 MB` until it
  finishes.
- There is no percentage or ETA: computing them would mean walking the whole
  tree before copying anything, delaying the start and defeating the point.

---

## Behaviour

- The **destination is created** if missing, including intermediate directories.
- Existing files at the destination are **always overwritten**, without asking.
  If the destination file is *read-only*, *hidden* or *system*, the attribute is
  cleared and the copy is retried.
- **It is not a mirror**: anything already at the destination that is not in the
  source **is kept**. Nothing is ever deleted.
- **Symlinks and junctions are skipped** by default, to avoid infinite loops and
  duplicated copies. They are reported in the final summary. With `-L` they are
  followed and their contents copied.
- **An error on one file does not abort the copy**: it is reported on `stderr`
  and the walk continues. The final count is shown and the exit code is `1`.
- File **attributes** (read-only, hidden…) and timestamps are copied, because
  `CopyFileExW` does so natively.
- NTFS **permissions (ACLs)** and alternate data streams (ADS) are **not**
  copied.

---

## Validation

Nothing is copied until all of these checks pass:

| Situation | Result |
| --- | --- |
| Argument count other than 2 | Error stating which one is missing or how many are extra |
| Arguments merged by a trailing backslash | Error explaining the cause and the fix |
| Leftover quotes around a path | Stripped automatically |
| Empty or whitespace-only path | Error |
| Characters Windows forbids (`< > " \| ? *`, `:` outside the drive letter, control characters) | Error pointing at the specific character |
| Source does not exist | Error |
| Source is a file, not a directory | Error |
| Destination exists but is a file | Error |
| Source and destination are the same directory | Error |
| Destination sits inside the source | Error: the copy would recurse |

Character validation understands drive letters (`D:`), network shares
(`\\server\share`) and the extended prefix (`\\?\`).

---

## How it gets its speed

- **`CopyFileExW`**, the native Win32 API. The kernel moves bytes from one
  handle to another: the data **never passes through user space** or Go buffers.
  Windows internally applies double buffering, asynchronous I/O and destination
  preallocation to avoid fragmentation.
- **`COPY_FILE_NO_BUFFERING`** on files of 32 MiB or more: avoids polluting the
  Windows file cache with single-use data. On small files the cache does help,
  so it is not enabled there. If the volume rejects the flag — some network
  shares do — it retries automatically without it.
- **Two separate worker pools**: 4 workers walk the tree while `-w` workers
  copy, each on its own queue. Walking and copying overlap completely, and a
  worker listing a directory is never a worker that could have been copying.
- **Bounded memory**: the file queue holds at most `-cola` entries (4,096 by
  default), so the walk — which is ~300× faster than copying — cannot run ahead
  and pile up the whole tree in RAM. Queue entries store only the file name plus
  a pointer to a shared parent folder, so path text is stored once per directory
  instead of once per file. Peak memory stays flat regardless of tree size:
  **16 MB for both 20,000 and 300,000 files**.
- **Extended `\\?\` paths**: besides removing the 260-character limit, they skip
  the path-normalisation cost Win32 applies on every call.
- **Unsorted listing** (`ReadDir(-1)`) with sizes taken from `FindFirstFileW`
  itself, with no extra `stat` per file.
- The queue is drained **LIFO**, which improves locality while descending the
  directory tree.

---

## Choosing the worker count

| Scenario | Recommendation |
| --- | --- |
| Internal NVMe → internal NVMe | Default (`NumCPU * 2`) |
| Internal SSD → external USB SSD | `-w 8` |
| Anything → **external USB HDD** | `-w 4` (or `-w 2` if the drive clicks) |
| Mechanical HDD → mechanical HDD | `-w 2` |
| Network share (SMB) | `-w 8` to `-w 16`: parallelism hides network latency |
| Same physical disk on both ends | `-w 2`: the head competes with itself |

Rule of thumb: **more workers only help if the device can serve several
operations at once.** Mechanical drives cannot; SSDs can.

---

## Limitations

- **Windows only.** It depends on the Win32 API; it does not build on Linux or
  macOS.
- **Not a mirror.** It never deletes destination files that are gone from the
  source.
- **No partial-file resume.** If a copy is interrupted, the file in flight is
  copied again in full next time (with `-u`, the ones that did finish are
  skipped).
- **`-u` compares size and timestamp**, not content. See
  [Incremental copies](#incremental-copies).
- **No ACL or alternate data stream copying.** If you need NTFS permissions
  preserved, use `robocopy /COPYALL`.
- **No filters** by extension, pattern or date.

---

## Building

Requires Go 1.21 or newer.

```powershell
go build -trimpath -ldflags "-s -w" -o UltraCopySam.exe .
```

- `-trimpath` strips build paths from the binary.
- `-ldflags "-s -w"` drops the symbol table and debug info, shrinking the
  executable.

The result is a self-contained binary under 2 MB, with no external dependencies
beyond `kernel32.dll`, which is part of Windows.

Static checks:

```powershell
go vet ./...
```

---

## Code layout

| File | Contents |
| --- | --- |
| `main.go` | Command-line interface, progress line and final summary |
| `args.go` | Argument sanitising and path validation |
| `copier.go` | Copy engine: the two worker pools, tree walk, `-u` logic and stats |
| `queue.go` | Directory queue and the shared-parent job type that bounds memory |
| `winapi.go` | Direct bindings to `kernel32.dll` (`CopyFileExW`, `CreateDirectoryW`, attributes, volume info) |
| `path.go` | Conversion to extended `\\?\` paths |
| `install.ps1` | PowerShell installer and uninstaller |
| `bench/bench.ps1` | Benchmark script used for the numbers above |

Every Go file carries the `//go:build windows` build tag.

> [!TIP]
> **[docs/ANALISIS.md](docs/ANALISIS.md)** is a deep technical write-up for
> developers (in Spanish): architecture, the concurrency model and why it cannot
> deadlock, every design decision with its measurements, the hypotheses that were
> tested and turned out wrong, and the Windows pitfalls found along the way. Read
> it before changing the engine.

---

## License

**[BSD Zero Clause License (0BSD)](LICENSE)** — the most permissive license
there is.

You may use, copy, modify, redistribute and sell this software, commercially or
not, with no conditions whatsoever: **you are not even required to keep the
copyright notice or credit the author**. It is equivalent to placing it in the
public domain, but worded as a license so that it holds up in every
jurisdiction.

The only thing the license does is what every open-source license must: make
clear that the software is provided **as is**, without warranty, and that the
author is not liable for any damage its use may cause.
