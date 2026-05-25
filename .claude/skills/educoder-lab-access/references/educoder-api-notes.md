# Educoder API Notes

Verified on 2026-05-25 against the Educoder APIs.

## Defaults

- Workspace helper: `go run ./cmd/educoder`
- Course id/code: discover with `go run ./cmd/educoder courses`; do not hardcode user-specific course identifiers in source or docs.
- API base: `https://data.educoder.net/api`
- Frontend origin: `https://www.educoder.net`
- Web terminal base observed in task JSON: `wss://webssh.educoder.net`

## Login Model

Use `go run ./cmd/educoder login --username <username>` to initialize or refresh CLI access. The helper sends a signed `POST /api/accounts/login.json` request, reads the returned `cs` session header plus optional autologin token, verifies the session with `GET /api/users/get_user_info.json`, then saves the minimal session credential outside the repository with `0600` permissions. Later commands load the saved CLI credential and do not need a browser. Do not commit or print passwords, session values, or signing secrets.

## Signing

Requests that hit `data.educoder.net/api` need frontend-style signing:

- Decode `ak` and `sk` from Umi module `51459`; values in the bundle are base64 encoded twice.
- Timestamp is `Date.now()` style milliseconds.
- Raw string: `method=<METHOD>&ak=<ak>&sk=<sk>&time=<timestamp>`
- Signature: `md5(base64(raw string))`
- Required headers include `X-EDU-Type: pc`, `X-EDU-Timestamp`, `X-EDU-Signature`, `Pc-Authorization`, `Origin`, `Referer`, and `X-Original-*` headers.

Use direct `https://data.educoder.net/api/...` URLs. The `www.educoder.net/api` paths redirect to `data.educoder.net` and can lose cookie behavior in simple clients.

## Useful Endpoints

- `POST /api/accounts/login.json` with `{"login":"<username>","password":"<password>","autologin":true}` to create a CLI session; read the response `cs` header as the Educoder session.
- `GET /api/users/get_user_info.json`
- `GET /api/users/<login>/courses.json?page=1&per_page=100` to discover course ids and classroom codes.
- `GET /api/courses/<course_id>/homework_commons.json?homework_type=practice&page=1&limit=100&zzud=<login>`
- `GET /api/shixuns/<shixun_identifier>/shixun_exec.json?zzud=<login>`
- `GET /api/tasks/<game_identifier>.json?zzud=<login>`
- `GET /api/myshixuns/<myshixun_identifier>/active_pod.json?shixun_environment_id=<id>&tab_type=<n>&game_id=<numeric_game_id>&zzud=<login>`
- `GET /api/myshixuns/<myshixun_identifier>/start.json?shixun_environment_id=<id>&tab_type=<n>&game_id=<numeric_game_id>&homework_common_id=<homework_id>&zzud=<login>`
- `POST /api/tasks/<game_identifier>/game_build.json?homework_common_id=<homework_id>` with `{}` to trigger evaluation/submission.
- `GET /api/tasks/<game_identifier>/game_status.json?homework_common_id=<homework_id>` to fetch evaluation results.
- `POST /api/tasks/<game_identifier>/proxy_list`
- `POST /api/tasks/<game_identifier>/port_proxy` with JSON body `{"port":22}`

## Terminal Access

`start.json` returns an SSH gateway (`ssh_address`, `port`, `username`, `password`) plus WebSocket fields. The helper command below was verified to enter a lab VM and run read-only commands:

```bash
go run ./cmd/educoder vm-exec --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id> --homework-id <homework-id> --cmd 'pwd; ls -1 | head'
```

Do not print the returned password. `vm-exec` passes it through an environment variable to `expect`.

## Submission Access

`game_build.json` was verified to start evaluation and `game_status.json` returned the test result. A failing result means the current lab code/token output did not satisfy the judge; it does not imply the submission/evaluation route is unavailable.
