# VK Rust server find bot

VK chat bot for finding Rust game servers.

## Building from source

1. [Install the Go compiler](https://golang.org/dl/)
2. In the project directory, run `go build`

## Config

Filename: `config.json`

Parameters:

| Parameter | Default value | Description
----------- | ------------- | -----------
| token | - | VK API group token (string)
| group_id | - | VK group id (string)
| start_commands | `["start","begin","hello"]` | Chat messages for bot activation (string array)
| welcome_message | `"Welcome!\n..."` | Help message (string)

## Server list

Filename: `server.tsv`

Format: TSV (Tab Separated Values)

Structure:

| Value | Example | Notes |
| ----- | ---- | ----- |
| IP:PORT | `127.0.0.1:28015` | Server address and port |
| TEAM_PLAYERS | `3` (MAX 3 server) | Players limit in team |
| RATES | `2` (X2 server) | Server gather rates |
| WIPE_INTERVAL | `7` (every week) | Server wipe interval in days |
| GENRE | `m` (modded server) | Server genre. `m`=modded, `c`=classic, `f`=fun |
| PROJECT_NAME | `FACEPUNCH` | Server project name or any other string. Used to display unique servers |

Example file:

```text
127.0.0.1:28015 3 2 7 m LOCAL
192.168.1.1:28015 100 1 30 c OTHER
```
