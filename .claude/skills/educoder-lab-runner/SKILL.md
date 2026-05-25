---
name: educoder-lab-runner
description: Complete an Educoder OS course experiment end to end and write its course-design report. Use when the user asks Claude Code to finish a lab, continue the next lab, create a labs/labx workspace, inspect lab requirements, download or work with the lab VM/source tree, build and test the experiment locally or in Docker, prepare the required result artifact, trigger official evaluation, confirm completion status, or write/update a report using the course-required personal/group report format.
---

# Educoder Lab Runner

## Overview

Use this skill for the high-level lab workflow. Use the local Educoder access CLI, and the `educoder-lab-access` skill when available, only for platform operations such as discovering lab identifiers, VM access, file transfer, save/evaluate, and status checks.

The lab report must describe the experiment itself. Do not mention the access CLI, API endpoints, cookies, signed requests, VM gateway plumbing, or other platform automation details in the report.

## Workspace Rules

- Create one directory per experiment under `labs/`, named `labx/` where `x` is the lab number or the closest course-provided label.
- Keep the downloaded or edited experiment repository under `labs/labx/repo/` unless the user asks for a different layout.
- If a platform download creates an extra `myshixun/` directory under `labs/labx/repo/`, either treat that directory consistently as the repository root for all later commands or move its contents into `labs/labx/repo/`. Do not mix both layouts in reports and commands.
- Keep the report at `labs/labx/report.md` unless the course requires another filename.
- Do not put lab-specific identifiers, tokens, API dumps, private keys, cookies, session values, or raw secret repository contents inside `educoder-cli/`; that repository must remain safe to publish.
- Treat existing files as user work. Read them before editing, and do not overwrite unrelated changes.

## Workflow

1. Discover the target lab from the user's wording, `labs/`, and Educoder metadata. If platform access is needed, use the local CLI from `educoder-cli/`; prefer the existing binary when fresh, otherwise run tests and rebuild it.
2. Create or reuse the matching `labs/labx/` directory. Record working notes in that directory, not in `educoder-cli/`.
3. Collect requirements from the task description, repository README, scripts, tests, and any visible judge or result-file instructions. Do not assume the submission artifact format; infer it from the lab materials and validate it. Treat platform test expressions such as `string.contains(actual, "[PASS]<<...>>")` as output-matching rules, not necessarily as the literal contents expected in `result.txt`.
4. Download or synchronize the lab source tree into `labs/labx/repo/` when local work is needed. Use VM execution for small checks and local/Docker execution for repeatable builds, debugging, and reports.
5. Build the experiment in the environment that matches the lab. Prefer an existing Dockerfile/image or course scripts; create a lab-local Docker helper only when the environment is otherwise not reproducible. If the VM lacks build dependencies, do not install packages by default; first look for an existing local/Docker environment that can reproduce the build.
6. Implement the required lab changes with narrow edits. Run the smallest meaningful checks first, then the full local or Docker validation that proves the experiment behavior.
7. Prepare the required result artifact. Validate its format locally or inside the VM when possible. If a judge reads encoded or encrypted content, preserve the exact expected format; never replace it with a guessed plain token. If official evaluation fails with decoding errors such as base64 padding, JSON parse, RSA decrypt, or hidden `judger.py` traceback, inspect the hidden verifier format before changing the experiment code.
8. Synchronize the final files back to the Educoder VM or editor as required. Prefer the browser-equivalent save-and-evaluate flow when a file must be saved before evaluation; use plain submission only when no save metadata is needed. If the platform refuses to save a source file as a non-student code file, use VM upload for VM-state consistency and save/evaluate only the allowed challenge artifact.
9. Poll official status until the outcome is clear. Treat traceback, compile output, and failed test-set output as actionable debugging evidence rather than as generic service failure.
10. Write or update `labs/labx/report.md` after validation. Read `references/report-guidelines.md` before writing the report.

## Validation Expectations

- Capture enough evidence to support the final answer: local build/test result, VM self-check when available, and official evaluation status.
- Prefer exact status fields such as completion state, grade, compile result, and failing test-set output. Do not paste credentials, cookies, private keys, or secret file contents.
- If local checks pass but official evaluation fails, compare the saved platform file, the file evaluated by the official request, and the latest commit/save metadata before changing the lab solution.
- If a VM appears unhealthy, reset or delete the Pod only after the user has agreed or has already authorized that action.
- For the HDU OS course repository, `make user` failures that only show `cd: can't cd to .../build/u_objs` are often hiding an earlier CMake configuration error. Run CMake directly to expose the real error; if the assembler is not detected, configure with `-DCMAKE_ASM_COMPILER=riscv64-unknown-elf-gcc`.

## Report

Before writing the report, read:

```text
references/report-guidelines.md
```

The report should be a student-facing course-design report following `references/report-guidelines.md`. Use placeholders for missing student number, name, group member, main contributor, and division-of-work fields. It should not describe the automation toolchain used to access Educoder.
