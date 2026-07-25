# canvas-cli Domain Glossary

## Core Creed

**canvas-cli is a replacement for the Canvas LMS web interface**, adding automation convenience and agent-friendliness on top.

## Core Concepts

**canvas-cli** — A local CLI tool for Canvas LMS, used to manage courses, assignments, modules, files, discussions, the inbox, and more. Once installed, the binary is `canvas`.

## Users

**Student** — Someone who needs to quickly pull the context of their coursework.

**Teaching team** — Teachers, teaching assistants, graders, and course designers.

**Agent** — An automation agent. Needs deterministic, parseable, reversible behavior.

## Command Design Decisions

**Safety gates** — The three-tier safety mechanism: `--read-only`, `--dry-run`, `--confirm`.

**Raw API escape hatch** — `canvas api` allows calling any Canvas API endpoint directly.
