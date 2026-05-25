# Lab Report Guidelines

Use these guidelines when writing `labs/labx/report.md`.

## Required Shape

Write in Chinese unless the user or course requires another language. Use this structure unless the course gives a stricter template:

```markdown
# Lab X 实验报告：<实验主题>

## 一、实验原理

详细说明本实验验证的系统机制、关键模块、关键数据结构或执行流程。需要解释为什么这些机制能完成实验目标，而不只是罗列文件名。

## 二、实验方法

详细说明完成实验的主要步骤、关键代码修改、构建运行方式和验证方法。需要覆盖从理解任务、修改实现到运行验证的完整链路。

## 三、实验结果

详细列出本地或容器验证结果、VM 自检结果、官方评测状态，以及能证明通过的关键输出。需要说明结果如何对应实验目标。
```

## Writing Rules

- Describe the experiment, not the platform automation. Do not mention `educoder-cli`, API paths, cookies, signed headers, temporary SSH passwords, or save/evaluate implementation details.
- Keep the report technically specific: name the edited modules, scripts, important commands, expected behavior, and observed behavior.
- Do not write a minimal checklist report. Each required section must contain enough explanation for a reader to understand what was tested, how it was done, and why the result proves completion.
- Include command output snippets only when they support the result. Prefer short excerpts over full logs.
- Do not include passwords, session cookies, private keys, temporary SSH passwords, or raw secret repository content.
- The report may include the original token or result value when it is an experiment output, course-required evidence, or useful for documenting the result.
- State the final official evaluation result using stable fields such as pass/fail, score, compile status, and test-set result.
