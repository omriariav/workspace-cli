# Release Procedure

Canonical release procedure for `gws`. CLAUDE.md points here; if the two
ever diverge, this document wins. Release notes history lives in
[RELEASES.md](RELEASES.md).

Releases are PR-only. Never push release commits, version bumps, tags, or
GitHub releases directly from local `main`.

## Hard gates

No release may be tagged, drafted, or published until ALL of these hold:

1. **Release PR** exists and contains the version bump, release notes/docs,
   and related issue links where applicable (never fabricate links).
2. **CI green** — GitHub Actions pass on the latest PR commit.
3. **Automated reviews inspected and resolved** — a green check status
   alone is NOT sufficient. Before merge or release, a reviewer must open
   and read every automated PR-review result and comment on the latest
   commit — the Codex review and every GitHub Actions review/feedback
   comment (e.g. `post_feedback`) included — and confirm zero unresolved
   actionable warnings, errors, or critical findings. A `SUCCESS`
   conclusion on the check does not satisfy this gate: the comment bodies
   must be inspected. Any actionable finding must be fixed on the PR
   branch and the automated reviews re-inspected on the updated latest
   commit before the gate passes.
4. **`make release-check` passes** locally on the release branch or merged
   result. This runs fmt, vet, and the full test suite — which includes the
   skill integrity gates below — and requires a clean tree.
5. **No existing artifacts** — no local tag, remote tag, draft release, or
   published release already exists for `vX.Y.Z`.
6. **Merged first** — the release PR is merged and the tag points at the
   merged commit on `main`.

If any gate is missing, do not run `make release`. If release automation
already started from an unreviewed commit, stop it, delete any draft
release/tag it created, restore `main` with a revert if needed, and route
the release through a PR.

## Skills are a release gate

The plugin bundle (skills and their mirrors) ships with every release and
is part of the release surface. All of the following are mandatory and
enforced by tests that `make release-check` runs:

1. **Uniform source skill versions** — every `skills/*/SKILL.md` `version`
   field equals the release version in the Makefile and in BOTH plugin
   manifests (`.claude-plugin/plugin.json`,
   `plugins/gws/.codex-plugin/plugin.json`).
   Enforced by `TestSkillVersions_MatchPackageVersion` and the manifest
   version tests.
2. **Codex mirrors are version-free by design** — `plugins/gws/skills/*`
   SKILL.md files must NOT carry a top-level `version` field, and must be
   byte-identical to their source counterpart after stripping only the
   source's top-level `version` line. Bundled reference files must match
   exactly. Enforced by `TestCodexSKILLmd_HasCodexFrontmatter` and
   `TestCodexSkills_MirrorClaudeSkills`.
3. **Skill content audit** — every release audits the skills affected by
   the change set for: newly supported or newly restricted CLI capability,
   accurate examples, correct scopes/auth-mode statements, and precise
   exclusions (e.g. Developer Preview boundaries). Source, mirror, and
   agent metadata (`agents/openai.yaml`) must stay in parity, and the
   inventory tests (`expectedSkills`, `servicesWithCommands`, file counts)
   must cover any new skill.
4. **`make release-check` enforces the above** — it runs the full test
   suite, so a version drift, mirror divergence, or inventory gap fails
   the release gate before any tag or draft exists.
5. **No publish before the bundle check** — a release is not published
   until the plugin bundle has passed these checks on the exact commit
   being tagged.

## Checklist

```
1.  git checkout -b release/vX.Y.Z main
2.  Edit Makefile: bump VERSION (e.g. 1.43.0 -> 1.43.1)
3.  Set every skills/*/SKILL.md version to X.Y.Z and both plugin
    manifests to X.Y.Z; re-sync plugins/gws mirrors if skills changed
4.  Edit CLAUDE.md: update "Current Version" line
5.  Audit skills touched by this release (capabilities, examples,
    scopes/auth, exclusions) — see "Skills are a release gate"
6.  Update release notes/docs (RELEASES.md) and issue links
7.  git add -A && git commit -m "Release vX.Y.Z"
8.  git push -u origin release/vX.Y.Z and open a PR against main
9.  Wait for green GitHub Actions, then open and read EVERY automated
    review/comment body on the latest commit (Codex review, GitHub
    Actions feedback comments such as post_feedback) — zero unresolved
    actionable findings; a green check alone is not enough. Fix issues
    only by pushing to the PR branch, then re-read all automated
    reviews on the new head
10. Merge the PR only after all hard gates are satisfied
11. git checkout main && git pull --ff-only
12. Verify the release tag/release does not already exist
13. make release VERSION=X.Y.Z
    — runs release-check (fmt/vet/test incl. skill gates, clean tree),
      tags, cross-compiles, uploads binaries, verifies
    — creates a DRAFT release — edit notes and publish:
      gh release edit vX.Y.Z --draft=false --notes "<release notes>"
14. make install
    — installs to $GOPATH/bin with the correct version embedded
15. Announce (e.g. /tweet) — optional comms step
```

## Makefile targets

| Target | What it does |
|--------|-------------|
| `make release-check` | Pre-release gate: fmt + vet + full tests (incl. skill version/mirror/inventory gates) + clean-tree check |
| `make release VERSION=x.y.z` | Full release: check → tag → cross-compile → upload → verify |
| `make install` | Install to `$GOPATH/bin` with proper `-ldflags` (never use `go install` directly) |

**Never use `go install ./cmd/gws` directly** — it produces a
pseudo-version like `v1.27.1-0.xxxxx` instead of the clean version.
Always use `make install`.
