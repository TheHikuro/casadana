---
name: commit
description: Commit staged changes using conventional commit format and GitHub CLI. Splits related changes into multiple focused commits. Applies when committing code changes.
---

# Commit Command

Guide for creating well-structured git commits using conventional commit format.

## When to Apply

Use this skill when:

- Committing staged changes to the current branch
- Multiple related changes need splitting into logical commits
- Creating commit history for a feature or bug fix
- Following conventional commit standards

## Commit Format

Use conventional commit format:

```
<type>(<scope>): <subject>

[optional body]
```

### Types

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only
- `style` - Code style (formatting, missing semi-colons, etc)
- `refactor` - Code change that neither fixes a bug nor adds a feature
- `perf` - Performance improvement
- `test` - Adding or updating tests
- `chore` - Build process, tooling, dependencies

### Scope

Optional context (component, file, module name):

- `feat(auth): add password reset flow`
- `fix(api): handle null response from GraphQL`
- `docs(readme): update installation steps`

## Rules

1. **Use GitHub CLI** - Create commits via `gh` command
2. **Commit to current branch** - Don't switch branches
3. **Split related changes** - Multiple logical commits > one large commit
4. **No AI mentions** - Don't reference Claude Code or AI generation in messages
5. **Keep subjects short** - 50 chars or less
6. **Use imperative mood** - "add" not "added", "fix" not "fixed"

## Workflow

1. Review staged changes: `git status`, `git diff --staged`
2. Group related changes logically
3. Create commits for each logical unit
4. Verify commit history: `git log --oneline`
5. **Check for open PR** — run `gh pr view --json url,title 2>/dev/null`
   - If a PR exists: load the `pull-request` skill and run its **Update Workflow** (re-run self-check, update PR body via `gh pr edit`)
   - If no PR: done

## Examples

See README.md for detailed examples of common commit patterns.
