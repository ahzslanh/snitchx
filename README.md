# snitchX

**Network. Processes. Services. Compliance.**

snitchX is a cross-platform security inspection tool. It shows which processes
own which network endpoints, and extends that into service, startup, and
compliance inspection.

snitchX is a fork of [karol-broda/snitch](https://github.com/karol-broda/snitch),
released under the MIT license.

![snitchX demo](demo/demo.gif)

## platform support

| platform | endpoint inspection | process mapping | services / startup | compliance |
|----------|--------------------|-----------------|--------------------|------------|
| macOS    | libproc            | yes             | roadmap            | roadmap    |
| Linux    | `/proc/net/*`      | yes             | roadmap            | roadmap    |
| Windows  | iphlpapi           | yes             | in progress        | roadmap    |

## install

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/ahzslanh/snitchx/master/install.ps1 | iex
```

The installer detects OS/architecture, an existing installation and its
version, and whether the current shell is elevated. Without elevation it
installs per-user to `%LOCALAPPDATA%\Programs\snitchX`; from an elevated
terminal it installs machine-wide to `%ProgramFiles%\snitchX`.

Or download the release archive and run `snitchx.exe`:

- `snitchx_<version>_windows_amd64.tar.gz`
- `snitchx_<version>_windows_arm64.tar.gz`

### macOS / Linux (shell script)

```bash
curl -sSL https://raw.githubusercontent.com/ahzslanh/snitchx/master/install.sh | sh
```

Installs to `~/.local/bin` when it is on `PATH`, otherwise `/usr/local/bin`.
Override with:

```bash
curl -sSL https://raw.githubusercontent.com/ahzslanh/snitchx/master/install.sh | INSTALL_DIR=~/bin sh
```

> **macOS:** the install script removes the `com.apple.quarantine` attribute so
> the binary can run without Gatekeeper warnings. Set `KEEP_QUARANTINE=1` to
> disable this.

### go

```bash
go install github.com/ahzslanh/snitchx@latest
```

### nix

```bash
# try it
nix run github:ahzslanh/snitchx

# install to profile
nix profile install github:ahzslanh/snitchx
```

```nix
{
  inputs.snitchx.url = "github:ahzslanh/snitchx";
}
# then use: inputs.snitchx.packages.${system}.default
```

### home-manager

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    home-manager.url = "github:nix-community/home-manager";
    snitchx.url = "github:ahzslanh/snitchx";
  };

  outputs = { nixpkgs, home-manager, snitchx, ... }: {
    homeConfigurations."user" = home-manager.lib.homeManagerConfiguration {
      pkgs = nixpkgs.legacyPackages.x86_64-linux;
      modules = [
        snitchx.homeManagerModules.default
        {
          programs.snitchx = {
            enable = true;
            settings = {
              defaults = {
                theme = "catppuccin-mocha";
                interval = "2s";
                resolve = true;
              };
            };
          };
        }
      ];
    };
  };
}
```

### binary releases

Download from [releases](https://github.com/ahzslanh/snitchx/releases):

- **Windows:** `snitchx_<version>_windows_<arch>.tar.gz` (contains `snitchx.exe`)
- **Linux:** `snitchx_<version>_linux_<arch>.tar.gz` or `.deb` / `.rpm` / `.apk`
- **macOS:** `snitchx_<version>_darwin_<arch>.tar.gz`

```bash
tar xzf snitchx_*.tar.gz
sudo mv snitchx /usr/local/bin/
```

> **macOS:** if macOS blocks the binary, run:
>
> ```bash
> xattr -d com.apple.quarantine /usr/local/bin/snitchx
> ```

## quick start

```bash
snitchx              # launch interactive tui
snitchx -l           # tui showing only listening sockets
snitchx ls           # print styled table and exit
snitchx ls -l        # listening sockets only
snitchx ls -t -e     # tcp established connections
snitchx ls -p        # plain output (parsable)
snitchx json         # json output for scripting
```

## commands

### `snitchx` / `snitchx top`

Interactive TUI with a live-updating connection list.

```bash
snitchx                  # all connections
snitchx -l               # listening only
snitchx -t               # tcp only
snitchx -e               # established only
snitchx -i 2s            # 2 second refresh interval
```

**keybindings:**

```
j/k, arrows   navigate
g/G           top/bottom
t/u           toggle tcp/udp
l/e/o         toggle listen/established/other
s/S           cycle sort / reverse
w             watch/monitor process (highlight)
W             clear all watched
K             terminate process (with confirmation)
/             search
enter         connection details
?             help
q             quit
```

### `snitchx ls`

One-shot table output. Uses a pager automatically if output exceeds terminal height.

```bash
snitchx ls               # styled table (default)
snitchx ls -l            # listening only
snitchx ls -t -l         # tcp listeners
snitchx ls -e            # established only
snitchx ls -p            # plain/parsable output
snitchx ls -o json       # json output
snitchx ls -o csv        # csv output
snitchx ls -n            # numeric (no dns resolution)
snitchx ls --no-headers  # omit headers
```

### `snitchx json`

JSON output for scripting.

```bash
snitchx json
snitchx json -l
```

### `snitchx stats`

Aggregated connection counters.

```bash
snitchx stats
snitchx stats proto=tcp
```

### `snitchx watch`

Stream JSON frames at an interval.

```bash
snitchx watch -i 1s | jq '.count'
snitchx watch -l -i 500ms
```

### `snitchx trace`

Print new and closed connections as they happen.

```bash
snitchx trace
```

### `snitchx upgrade`

Check for updates and upgrade in-place.

```bash
snitchx upgrade              # check for updates
snitchx upgrade --yes        # upgrade automatically
snitchx upgrade -v 0.1.7     # install specific version
```

## filters

Shortcut flags work on all commands:

```
-t, --tcp           tcp only
-u, --udp           udp only
-l, --listen        listening sockets
-e, --established   established connections
-4, --ipv4          ipv4 only
-6, --ipv6          ipv6 only
```

For more specific filtering, use `key=value` syntax:

```bash
snitchx ls proto=tcp state=listen
snitchx ls pid=1234
snitchx ls proc=nginx
snitchx ls lport=443
snitchx ls contains=google
```

## resolution

```
--resolve-addrs     resolve ip addresses to hostnames (default: true)
--resolve-ports     resolve port numbers to service names
--no-cache          disable dns caching (force fresh lookups)
```

DNS lookups are performed in parallel and cached. Use `--no-cache` to bypass
the cache for debugging or when addresses change frequently.

## output

Styled table (default):

```
  ╭─────────────────┬───────┬───────┬─────────────┬─────────────────┬────────╮
  │ PROCESS         │ PID   │ PROTO │ STATE       │ LADDR           │ LPORT  │
  ├─────────────────┼───────┼───────┼─────────────┼─────────────────┼────────┤
  │ nginx           │ 1234  │ tcp   │ LISTEN      │ *               │ 80     │
  │ postgres        │ 5678  │ tcp   │ LISTEN      │ 127.0.0.1       │ 5432   │
  ╰─────────────────┴───────┴───────┴─────────────┴─────────────────┴────────╯
  2 connections
```

Plain output (`-p`):

```
PROCESS    PID    PROTO   STATE    LADDR       LPORT
nginx      1234   tcp     LISTEN   *           80
postgres   5678   tcp     LISTEN   127.0.0.1   5432
```

## configuration

Optional config file at `~/.config/snitchx/snitchx.toml`:

```toml
[defaults]
numeric = false      # disable name resolution
dns_cache = true     # cache dns lookups (set to false to disable)
theme = "auto"       # color theme: auto, dark, light, mono

[tui]
remember_state = false   # remember view options between sessions
```

Available themes: `ansi`, `catppuccin-mocha`, `catppuccin-macchiato`,
`catppuccin-frappe`, `catppuccin-latte`, `gruvbox-dark`, `gruvbox-light`,
`dracula`, `nord`, `tokyo-night`, `tokyo-night-storm`, `tokyo-night-light`,
`solarized-dark`, `solarized-light`, `one-dark`, `mono`.

### remembering view options

When `remember_state = true`, the TUI saves and restores:

- filter toggles (tcp/udp, listen/established/other)
- sort field and direction
- address and port resolution settings

State is saved to `$XDG_STATE_HOME/snitchx/tui.json` (defaults to
`~/.local/state/snitchx/tui.json`). CLI flags always take priority over saved
state.

### environment variables

```bash
SNITCHX_THEME=dark          # set default theme
SNITCHX_RESOLVE=0           # disable dns resolution
SNITCHX_DNS_CACHE=0         # disable dns caching
SNITCHX_NO_COLOR=1          # disable color output
SNITCHX_CONFIG=/path/to     # custom config file path
```

## platform notes

### Windows

snitchX uses the Windows IP helper API (`iphlpapi.dll`) to enumerate IPv4 and
IPv6 TCP/UDP endpoints with their owning PIDs, then resolves process names and
accounts best-effort. Protected/system processes may show an empty name. The
Windows UDP owner table does not expose a remote address or connection state,
so UDP sockets are reported as `LISTEN` with a wildcard remote. Run in an
elevated terminal for the most complete process information.

### Linux

snitchX reads from `/proc/net/*`. Root or `CAP_NET_ADMIN` may be required for
full process information.

### macOS

snitchX uses libproc and system APIs. Elevated privileges may be required for
full process information.

## requirements

- macOS, Linux, or Windows 10/11/Server
- Windows: amd64 or arm64; Linux: amd64, arm64, armv7; macOS: amd64 or arm64

snitchX does not make network connections, except for `snitchx upgrade`, which
fetches releases from GitHub. It does not write to system files or collect or
transmit any data.

## development

```bash
go build ./...
go test ./...

# cross-compile for Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o snitchx.exe .
```

## license

MIT. snitchX is a fork of [karol-broda/snitch](https://github.com/karol-broda/snitch);
the original copyright and license are retained in [LICENSE](LICENSE).
