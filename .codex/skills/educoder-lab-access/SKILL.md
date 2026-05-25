---
name: educoder-lab-access
description: Access Educoder OS course lab data and task terminal/proxy/evaluation APIs from this project's CLI credential store. Use when the user asks to inspect Educoder labs, list lab/homework identifiers, open or resume a shixun task, check task details, execute or download files from a lab VM, submit a task, check official evaluation status, investigate web terminal access, or test whether a lab VM exposes an SSH/port proxy.
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
go run ./cmd/educoder reset-pod --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id> --homework-id <homework-id>
go run ./cmd/educoder delete-pod --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id> --homework-id <homework-id>
go run ./cmd/educoder vm-exec --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id> --homework-id <homework-id> --cmd 'pwd'
go run ./cmd/educoder vm-download --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id> --homework-id <homework-id> --remote /data/workspace/myshixun --local ../labs/labx/repo
go run ./cmd/educoder vm-download --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id> --homework-id <homework-id> --remote /data/workspace/myshixun/<subdir> --local ../labs/labx/repo/<repo-root>/<subdir>
go run ./cmd/educoder vm-upload --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id> --homework-id <homework-id> --local ../labs/labx/repo/result.txt --remote /data/workspace/myshixun/result.txt
go run ./cmd/educoder update-file --myshixun <myshixun-id> --game-id <numeric-game-id> --homework-id <homework-id> --path result.txt --local ../labs/labx/repo/result.txt
go run ./cmd/educoder evaluate-file --task <game-id> --myshixun <myshixun-id> --env <env-id> --game-id <numeric-game-id> --homework-id <homework-id> --path result.txt --local ../labs/labx/repo/result.txt
go run ./cmd/educoder proxy-list --task <game-id>
go run ./cmd/educoder port-proxy --task <game-id> --port 22
go run ./cmd/educoder submit --task <game-id> --homework-id <homework-id>
go run ./cmd/educoder status --task <game-id> --homework-id <homework-id>
```

Use `--json` for raw API responses when task fields, submission details, or test-set outputs are needed for follow-up automation.

## Workflow

1. Run `login` first after a fresh checkout or expired credential. It prompts for an Educoder password unless `--password-stdin` is used; the password is not stored.
2. Run `courses` when course selection is unclear. Do not rely on hardcoded course ids or classroom codes; pass `--course-id` or `--course-code` only after discovering them from the CLI.
3. Run `labs` to map lab names to `shixun_identifier`, `myshixun_identifier`, and `student_work_id`. It auto-discovers the course when unambiguous.
4. If a lab has no `myshixun_identifier`, run `start --shixun <id>` to create/resume a task and get the `game_identifier`.
5. Run `task --task <game_identifier>` to collect `game.id`, `myshixun.identifier`, `shixun_environments`, `wss_url`, repository URLs, and any homework id needed for later save/evaluate/status calls.
6. Run `active-pod` with the task's `myshixun.identifier`, numeric `game.id`, and environment id when checking VM lifetime.
7. Run `reset-pod` only when the active VM appears unhealthy and the user accepts that uncommitted VM files may be lost. Run `delete-pod` only when intentionally destroying the active Pod so Educoder can create a fresh one later.
8. Run `vm-exec` with `myshixun.identifier`, numeric `game.id`, environment id, and `homework_common_id` to execute commands inside the lab VM through Educoder's SSH gateway.
9. Run `vm-download` when a remote file or source tree must be copied to a local destination. If used from this workspace, keep experiment files under the matching `labs/labx/` folder. Downloading a directory can create an extra top-level directory at the destination; inspect the resulting layout before editing. If a local copy appears incomplete, verify the remote tree with `vm-exec` and download the missing subdirectory explicitly.
10. Run `vm-upload` when a local result file or source tree must be copied back into the lab VM.
11. Run `update-file` when an allowed challenge file must be saved through Educoder's editor API without triggering evaluation. If the API says the file is not an editable student code file, do not retry the same save; use `vm-upload` for VM-state consistency and save/evaluate only the permitted artifact.
12. Prefer `evaluate-file` when saving a permitted challenge file and triggering official evaluation in one browser-equivalent flow; it preserves the `sec_key`, `resubmit`, `content_modified`, and commit metadata returned by the save API.
13. Run `submit --task <game.identifier> --homework-id <homework_common_id>` only when no file save metadata is needed.
14. Run `status --task <game.identifier> --homework-id <homework_common_id>` to check completion. Use `--json status ...` if the concise summary is not enough.
15. Run `proxy-list` before `port-proxy`; only create a port proxy when the user needs it or when verifying SSH feasibility.

## Result Artifact Notes

- `testCasesExp` and test-set metadata describe how Educoder matches judge output; they do not always describe the literal contents expected in `result.txt`.
- Some labs require `result.txt` to contain encoded or encrypted structured data that the hidden judge decodes before printing the final pass marker. If official status shows a hidden judge traceback for base64 decoding, decryption, or JSON parsing, inspect the hidden verifier format and regenerate the artifact instead of submitting a plain `[PASS]` token.
- Keep hidden verifier files, temporary decoded artifacts, and key material in a private temporary directory only. Never commit them, never place them under this repository, and remove the temporary directory after generating the final artifact.

## SSH And Terminal Notes

Educoder uses web terminal/WebSocket infrastructure and also returns an SSH gateway from `myshixuns/<id>/start.json` for these OS labs. Prefer `vm-exec` for Codex-driven terminal work because it hides the password and works from the local shell.

Arbitrary port proxying is separate from Educoder's own SSH gateway used by `vm-exec`; only create a port proxy when the target service is known to be running inside the lab VM.

If the user asks for deeper endpoint details, read `references/educoder-api-notes.md`.
