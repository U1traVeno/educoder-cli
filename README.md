# Educoder OS Lab Helper

Small Go CLI for accessing an Educoder OS lab course without a browser dependency. It helps Codex inspect lab metadata, read task descriptions, start/resume lab tasks, execute commands in the lab VM through Educoder's SSH gateway, and trigger the official evaluation endpoint.

This repository intentionally does not store cookies, passwords, session tokens, API responses, or build artifacts.

## Requirements

- Go 1.22+
- `expect` available on `PATH` for `vm-exec`

## Usage

Run from the repository root:

```bash
go run ./cmd/educoder login --username <username>
go run ./cmd/educoder courses
go run ./cmd/educoder labs
```

## Login Model

`educoder login` signs and sends a username/password login request to Educoder, validates the returned session, and persists the minimal CLI credential outside the repository:

1. Extract Educoder request-signing keys from the public frontend bundle.
2. Call `POST /api/accounts/login.json` with the username, password, and signed headers.
3. Read the returned session header and optional autologin token.
4. Call `GET /api/users/get_user_info.json` to verify the session.
5. Save the session credential to the user config directory with `0600` permissions.

The plaintext password is not saved. After login, normal commands use the CLI credential file and no browser is needed. Run `go run ./cmd/educoder logout` to remove the saved credential.

No session cookie, password, API dump, or generated credential is written into this repository. Override the credential location with `--credentials <path>` if needed.

For automation, read the password from standard input instead of putting it on the command line:

```bash
printf '%s\n' "$EDUCODER_PASSWORD" | go run ./cmd/educoder login --username <username> --password-stdin
```

Typical workflow:

```bash
go run ./cmd/educoder courses
go run ./cmd/educoder start --shixun <shixun-id>
go run ./cmd/educoder task --task <game-id>
go run ./cmd/educoder vm-exec \
  --myshixun <myshixun-id> \
  --env <env-id> \
  --tab 4 \
  --game-id <numeric-game-id> \
  --homework-id <homework-id> \
  --cmd 'pwd; ls -la'
go run ./cmd/educoder api-post \
  --path '/api/tasks/<game-id>/game_build.json?homework_common_id=<homework-id>' \
  --body '{}'
go run ./cmd/educoder api-get \
  --path '/api/tasks/<game-id>/game_status.json?homework_common_id=<homework-id>'
```

Use `--json` before a subcommand for raw JSON where supported:

```bash
go run ./cmd/educoder --json task --task <game-id>
```

The helper intentionally does not hardcode a course id or classroom code. `labs` discovers the current user's courses and uses the only available course, the matching `--course-code`, or the OS-course match when unambiguous. If discovery is ambiguous, run `courses` and pass `--course-id <id>` or `--course-code <code>`.

## Security Notes

- The CLI sends the login password only to Educoder's account login API and never stores it.
- Commands after `login` read the saved CLI credential instead of asking for a browser session.
- `vm-exec` obtains temporary SSH gateway credentials from Educoder and passes the password to `expect` through an environment variable; it does not print the password itself.
- Do not commit local API dumps, generated credential files, curl cookie jars, terminal logs containing credentials, or compiled binaries.

## Project Skill

A project-local Codex skill is included under `.codex/skills/educoder-lab-access`. It documents the verified workflow and endpoint notes for future Codex sessions working inside this repository.
