# Workflow Contracts

## Purpose

Maintain CI, review, packaging, and publication workflows. Read the [repository contract](../AGENTS.md) first.

## Ownership

This contract covers `.github/`, including workflows and their configuration. Application behavior and generated assets stay with their source owners.

## Local Contracts

- Check actual event, branch, path, runner, and permission filters before relying on a workflow. In `unit-tests.yml` and `pr-build.yml`, the `"*"` branch filter does not match a slash in a branch name. Several other workflows target only `main` or `next`.
- The existing unit-test workflows select Go 1.25; `go.mod` requires Go 1.25.5. Consult the manifests and selected toolchain when changing that contract.
- Review integration is separate from runtime changes. Verify the workflow on the actual PR base before relying on it. `workflows/robin.yml` pins Robin v2.12.3 and consumes `LLM_API_KEY`, `LLM_BASE_URL`, and `LLM_MODEL` from GitHub Secrets with `actions: read`, `contents: read`, and `pull-requests: write` permissions.
- Robin reviews opened, reopened, and ready-for-review PRs. Authorized owner, member, or collaborator slash commands use `issue_comment`. Pushes do not request reviews because `synchronize` is absent; inspect existing runs and the persistent task review budget before requesting another review.
- Adding workflow YAML is an executable change. When Robin is missing from the base, land a minimal separate bootstrap PR under the maintained workflow's bootstrap exception before the substantive PR. Secret-name references do not establish configured values, runner access, or a usable review.
- GitHub loads `issue_comment` workflows from the default branch. `workflow_dispatch` also requires a workflow on the default branch. A workflow only on another branch does not establish manual re-review availability.
- Retain existing provider configuration and task review budgets. Use GitHub Secrets for credentials and verify actual event/permission gates before any review request.
- Source publication, review, artifact builds, and release/deployment are separate actions. A workflow definition or static validation does not prove a successful run or published artifact.

## Work Guidance

Preserve unrelated workflows and release triggers. Check the destination's archive state, actual branch rules, allowed merge methods, secret-name metadata, and runner availability before publication. Keep task-specific admission decisions in the private task record.

## Verification

- Run `git diff --check` and the installed `actionlint` on any changed workflow YAML. Check pinned action inputs against their official contract.
- For future PR delivery, verify exact-head required checks, usable review results, finding dispositions, and handled thread resolution under the maintained workflow. Report unavailable or failed checks as gaps.

## Child DOX Index

None. This contract owns the `.github/` subtree.
