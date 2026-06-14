# canvas-cli 领域术语表

## 核心守则

**canvas-cli 是 Canvas LMS 网页版的替代品**，额外提供自动化便利和 agent 友好。

## 核心概念

**canvas-cli** — Canvas LMS 的本地 CLI 工具，用于管理课程、作业、模块、文件、讨论、收件箱等。安装后的二进制文件是 `canvas`。

## 用户

**学生** — 需要快速拉取课程工作上下文的人。

**教学团队** — 教师、助教、评分员、课程设计者。

**Agent** — 自动化代理。需要确定性、可解析、可逆的行为。

## 命令设计决策

**安全门控** — `--read-only`、`--dry-run`、`--confirm` 三级安全机制。

**原始 API 逃生舱** — `canvas api` 允许直接调用任何 Canvas API 端点。
