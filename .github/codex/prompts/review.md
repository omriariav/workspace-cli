You are reviewing a pull request for the gws Google Workspace CLI project (Go, Cobra/Viper).

## Step 1: Compute the PR diff

IMPORTANT: You are reviewing a PR, not the entire repository. Run this command first to get ONLY the changes in this PR:

```bash
git diff origin/$BASE_BRANCH...HEAD
```

Where $BASE_BRANCH is the base branch (usually `main`). If that fails, try:

```bash
git log --oneline origin/main..HEAD
git diff origin/main...HEAD
```

Only review the files and lines that appear in this diff. Do NOT review unchanged files.

## Step 2: Review the diff

Check the changed code for:
- **Correctness**: API usage, error handling, edge cases
- **Pattern consistency** with existing `cmd/*.go` files (Cobra command registration, flag definitions, `runXxx` functions, printer usage)
- **Test coverage**: new commands need httptest mocking and command structure tests in `commands_test.go`
- **Security**: no credential leaks, proper input validation
- **Code quality**: naming, formatting (gofmt), idiomatic Go
- **Docs**: README.md command table and `skills/*/references/commands.md` updated for new commands

Reference CLAUDE.md for project conventions.

## Step 3: Output

Provide a structured review with:
1. **Summary**: What changed (1-2 sentences based on the diff, not the whole repo)
2. **What looks good**: Patterns followed correctly
3. **Issues found**: List each issue with severity:
   - **Critical**: Bugs, security issues, broken functionality — must fix before merge
   - **Warning**: Missing validation, incomplete docs, inconsistencies — should fix
   - **Suggestion**: Style improvements, optional enhancements — nice to have

If no issues are found, explicitly state the PR is clean and ready to merge.
