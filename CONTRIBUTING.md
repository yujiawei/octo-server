# Contributing to DMWork IM

Thank you for your interest in contributing! Please follow these guidelines.

## Before You Start

### Claim the Issue First

**Before writing any code, you must:**

1. **Comment on the issue** to claim it — say you're working on it
2. **Describe your fix/approach** in the same comment
3. **Wait for acknowledgment** if the issue has active discussion

This prevents duplicate work. If someone else has already claimed an issue, coordinate with them or pick a different one.

Example comment:
```
I'll take this one.

**Approach:** Replace `ioutil.ReadAll` with `io.ReadAll` across all files.
Will also update imports to remove `io/ioutil`.
```

### Check for Existing PRs

Before starting work on an issue, check if there's already an open PR for it.

## Fork Workflow

**Do NOT clone the main repository directly. Use the fork workflow:**

```bash
# 1. Fork the repository on GitHub (click "Fork" button)

# 2. Clone YOUR fork
git clone git@github.com:YOUR_USERNAME/dmworkim.git
cd dmworkim

# 3. Add upstream remote
git remote add upstream https://github.com/Mininglamp-OSS/octo-server.git

# 4. Create a feature branch from upstream/main
git fetch upstream
git checkout -b fix/issue-XX upstream/main

# 5. Make your changes, commit, and push to YOUR fork
git push origin fix/issue-XX

# 6. Open a PR from your fork to Mininglamp-OSS/octo-server main branch
gh pr create --repo Mininglamp-OSS/octo-server
```

### Keeping Your Fork Updated

```bash
git fetch upstream
git checkout main
git merge upstream/main
git push origin main
```

## Branch Naming

- Bug fixes: `fix/issue-XX-short-description`
- Features: `feat/short-description`
- Chores: `chore/short-description`

## Pull Requests

### Requirements

- **One PR, one concern** — don't mix unrelated changes
- **Reference the issue** — use `Closes #XX` in the PR description
- **Test your changes** — build and verify locally
- **Describe what and why** — not just what you changed, but why

### PR Template

The repository provides a PR template. Please fill it out completely.

### AI-Assisted Contributions

If you used AI tools:
- Note it in the PR description
- Indicate testing level (untested / lightly tested / fully tested)
- Confirm you understand the code changes

## Bug Reports vs Feature Requests

- **Bugs** → Fix directly via PR (after claiming the issue)
- **Features / Architecture changes** → Open a Discussion first, get agreement, then implement

## Code Style

- Go code follows standard `gofmt` formatting
- Keep changes minimal and focused
- Don't introduce new dependencies without discussion

## Security

For security vulnerabilities, **do not open a public issue**. See [SECURITY.md](SECURITY.md).
