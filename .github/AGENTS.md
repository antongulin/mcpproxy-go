# Workflow Contracts

## Purpose

Maintain CI, review, packaging, and publication workflows. Read the [repository contract](../AGENTS.md) first.

## Ownership

This contract covers `.github/`, including workflows and their configuration. Application behavior and generated assets stay with their source owners.

## Local Contracts

- Check actual event, branch, path, runner, and permission filters before relying on a workflow. In `unit-tests.yml` and `pr-build.yml`, the `"*"` branch filter does not match a slash in a branch name. Several other workflows target only `main` or `next`.
- Keep review setup separate from runtime changes. When Robin is missing from the base, land a minimal bootstrap under the maintained workflow's bootstrap exception before relying on it for a substantive PR.
- `workflows/robin.yml` uses the pinned Robin action, the `LLM_API_KEY`, `LLM_BASE_URL`, and `LLM_MODEL` secret names, and explicit `actions: read`, `contents: read`, and `pull-requests: write` permissions. Secret names in YAML do not prove configured values or provider access.
- Robin reviews opened, reopened, and ready-for-review PRs. Authorized PR slash commands use `issue_comment`; pushes do not request reviews because `synchronize` is absent. Inspect existing runs before requesting another review and retain the task's review budget across rounds and repositories.
- GitHub loads `issue_comment` workflows from the default branch. `workflow_dispatch` also requires a workflow on the default branch; the Robin workflow has no dispatch trigger. A workflow present only on a maintenance branch does not establish manual re-review availability.
- Retain existing provider configuration. Keep credentials in GitHub Secrets, restrict command execution to authorized PR commenters, and do not use `pull_request_target` for Robin.
- Source publication, review, artifact builds, and release/deployment are separate actions. A workflow definition or static validation does not prove a successful run or published artifact.

## Work Guidance

Use the canonical Robin action contract when changing its inputs. Preserve unrelated workflows and release triggers. Check the destination's archive state, actual branch rules, allowed merge methods, secret-name metadata, and runner availability before publication. Keep task-specific admission decisions in the private task record.

## Verification

- `actionlint -shellcheck= -pyflakes= .github/workflows/robin.yml` validates the review workflow without running it.
- Run `git diff --check` and verify pinned action inputs against their official `action.yml`.
- For future PR delivery, verify exact-head required checks, usable review results, finding dispositions, and handled thread resolution under the maintained workflow. Report unavailable or failed checks as gaps.

## Child DOX Index

None. This contract owns the `.github/` subtree.
