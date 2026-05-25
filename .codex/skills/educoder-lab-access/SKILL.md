---
name: educoder-lab-access
description: Access Educoder OS course lab data and task terminal/proxy APIs from this project's CLI credential store. Use when the user asks to inspect Educoder labs, list lab/homework identifiers, open or resume a shixun task, check task details, investigate web terminal access, or test whether a lab VM exposes an SSH/port proxy.
---

# Educoder Lab Access

## Quick Start

Use the Go helper in this project:

```bash
go run ./cmd/educoder labs
```

`login` uses an Educoder username/password request, verifies the returned session, and saves a local CLI credential outside the repository. Later commands read that credential, extract Educoder signing keys from the cached/public Umi bundle, and call `https://data.educoder.net/api` directly with the same signed headers as the web frontend. Do not print passwords, cookie values, session headers, or signing secrets.

## Common Commands

```bash
go run ./cmd/educoder login --username <username>
printf '%s\n' "$EDUCODER_PASSWORD" | go run ./cmd/educoder login --username <username> --password-stdin
go run ./cmd/educoder logout
go run ./cmd/educoder whoami
go run ./cmd/educoder courses
go run ./cmd/educoder labs
go run ./cmd/educoder start --shixun <shixun-id>
go run ./cmd/educoder task --task <game-id>
go run ./cmd/educoder active-pod --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id>
go run ./cmd/educoder vm-exec --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id> --homework-id <homework-id> --cmd 'pwd'
go run ./cmd/educoder proxy-list --task <game-id>
go run ./cmd/educoder port-proxy --task <game-id> --port 22
go run ./cmd/educoder api-post --path '/api/tasks/<game-id>/game_build.json?homework_common_id=<homework-id>' --body '{}'
```

Use `--json` for raw API responses when task fields are needed for follow-up automation.

## Workflow

1. Run `login` first after a fresh checkout or expired credential. It prompts for an Educoder password unless `--password-stdin` is used; the password is not stored.
2. Run `courses` when course selection is unclear. Do not rely on hardcoded course ids or classroom codes; pass `--course-id` or `--course-code` only after discovering them from the CLI.
3. Run `labs` to map lab names to `shixun_identifier`, `myshixun_identifier`, and `student_work_id`. It auto-discovers the course when unambiguous.
4. If a lab has no `myshixun_identifier`, run `start --shixun <id>` to create/resume a task and get the `game_identifier`.
5. Run `task --task <game_identifier>` to collect `game.id`, `myshixun.identifier`, `shixun_environments`, `wss_url`, and repository URLs.
6. Run `active-pod` with the task's `myshixun.identifier`, numeric `game.id`, and environment id when checking VM lifetime.
7. Run `vm-exec` with `myshixun.identifier`, numeric `game.id`, environment id, and `homework_common_id` to execute commands inside the lab VM through Educoder's SSH gateway.
8. Run `api-post .../game_build.json` to trigger evaluation/submission, then `api-get .../game_status.json` to fetch the result.
9. Run `proxy-list` before `port-proxy`; only create a port proxy when the user needs it or when verifying SSH feasibility.

## SSH And Terminal Notes

Educoder uses web terminal/WebSocket infrastructure and also returns an SSH gateway from `myshixuns/<id>/start.json` for these OS labs. Prefer `vm-exec` for Codex-driven terminal work because it hides the password and works from the local shell.

For the verified Lab05 task, `port-proxy --port 22` returned `status=-1` with message `未检测到服务启动，请先启动服务`, so arbitrary 22-port proxying is not available unless an SSH service is started inside the lab VM first. This is separate from Educoder's own SSH gateway used by `vm-exec`.

If the user asks for deeper endpoint details, read `references/educoder-api-notes.md`.
