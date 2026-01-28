---
name: gws-morning
version: 0.4.0
description: "AI-powered morning inbox briefing. Reads Gmail, Google Tasks, Calendar, and OKR sheets to produce a prioritized daily briefing with actionable recommendations. Triggers: /morning, morning briefing, inbox triage, email priorities, daily digest."
metadata:
  short-description: AI inbox briefing with OKR/task matching
  compatibility: claude-code
---

# Morning Inbox Briefing (gws morning)

An AI-powered workflow skill that reads your Gmail inbox, cross-references your OKRs, active tasks, and today's calendar, then produces a prioritized briefing with actionable recommendations.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Dependency Check

**Before executing any `gws` command**, verify the CLI is installed:
```bash
gws version
```

If not found, install: `go install github.com/omriariav/workspace-cli/cmd/gws@latest`

## Authentication

Requires OAuth2 credentials with access to Gmail, Tasks, Calendar, and Sheets.
Run `gws auth status` to check. If not authenticated: `gws auth login`.

## How This Skill Works

This is a **workflow skill** — it orchestrates multiple `gws` commands in sequence, feeds the results to Claude for AI classification, and produces a prioritized briefing.

**You are the AI classifier.** After gathering data from all sources, use the classification rules and priority scoring below to produce the briefing output.

## Step 1: Load Configuration

Read the config file:

```bash
cat ~/.config/gws/inbox-skill.yaml
```

If the file does not exist, run the **First-Run Setup** (see below).

The config contains:
- `okr_sheet_id` — Google Sheets ID for the OKR document
- `okr_sheets` — Which sheet tabs to read (e.g., "Data Track (2026)")
- `task_lists` — Which Google Task lists to monitor (`"all"` or a list of names)
- `noise_strategy` — How to detect noise (`"promotions"` uses Gmail's Promotions category)
- `priority_signals` — Signals that boost priority:
  - `starred: true` — Starred emails get a priority boost
  - `vip_senders` — List of email addresses whose emails are always prioritized (manager, direct reports, department heads, key partners)
- `max_unread` — How many unread inbox emails to fetch (default 50, configurable)
- `inbox_query` — Gmail search query (default: `"is:unread in:inbox"`)
- `daily_log_doc_id` — Google Doc ID for the daily log (empty = skip logging)

## Step 2: Launch Background Data Gathering

Launch a **background agent** to run prefetch + pre-filter while the main agent shows the user immediate context.

### 2a. Start background agent

Spawn a background agent (using `run_in_background: true`) that runs:

```bash
skills/morning/scripts/morning-prefetch.sh "$SCRATCHPAD_DIR/morning"
skills/morning/scripts/morning-prefilter.sh "$SCRATCHPAD_DIR/morning"
```

The background agent:
1. Runs prefetch (fetches inbox, calendar, tasks, OKRs in parallel)
2. Runs pre-filter (archives OOO replies and calendar invite emails)
3. Reads `prefiltered.json` and all context files
4. Classifies all remaining emails inline using rules from `skills/morning/prompts/triage-agent.md`
5. Writes classification results to `$SCRATCHPAD_DIR/morning/classified.json`

**OKR caching:** OKR sheets are cached for 24 hours at `~/.cache/gws/morning/`. If the user says their OKRs changed, re-run with `--refresh-okr`.

### 2b. Show header immediately (main agent, while background works)

**Do not wait for the background agent.** While it runs, show the user what you already know from config:

```
/morning — <Day>, <Date>

Gathering inbox, calendar, tasks, and OKRs...

VIP senders: <names from config>
Noise strategy: <from config>
```

### 2c. Poll and narrate

Check the background agent's output file periodically. As data becomes available, update the user:

**After prefetch completes** (calendar.json exists):
```
Today's meetings:
  <time> <title>
  ...

Pre-filtering inbox... (removing OOO replies and calendar invites)
```

**After pre-filter completes** (prefiltered.json exists):
```
Inbox: <N> unread | <N> auto-archived (OOO/invites)
Classifying <N> remaining emails...
```

**After classification completes** (background agent done):
```
Classification done. <N> action | <N> review | <N> noise
Starting triage.
```

### What the scripts do internally

**Prefetch** (`morning-prefetch.sh`):
- `gws gmail list --max <max_unread> --query "is:unread in:inbox"` — **MUST use `in:inbox`**
- `gws calendar events --days 2` — today + tomorrow
- `gws tasks lists` → `gws tasks list <id>` for each list
- `gws sheets read <okr_sheet_id> "<sheet_name>!A1:Q100"` — OKR data (cached 24h)
- All fetched in parallel, results as JSON files in output directory

**Pre-filter** (`morning-prefilter.sh`):
- Archives OOO replies (Out of Office, OOO Re:, Automatic reply:)
- Archives calendar invite emails (Invitation:, Updated Invitation:, Canceled: from calendar-notification@google.com)
- Writes `prefiltered.json` (remaining) and `auto_handled.json` (archived with reasons)

### Output files

- `prefiltered.json` — remaining emails for AI classification (OOO/invites removed)
- `auto_handled.json` — items archived by pre-filter with reasons
- `classified.json` — classification results from background agent
- `inbox.json` — original unread inbox emails
- `calendar.json` — 2-day calendar events
- `tasks.json` / `tasks_<list_id>.json` — task data
- `okr_0.json` (etc.) — OKR sheet data

## Step 3: Classify (Background Agent)

The background agent classifies all remaining emails using the rules from `skills/morning/prompts/triage-agent.md`. This happens as part of the background agent launched in Step 2 — the main agent does NOT classify.

For each email, the background agent determines:
- **Classification:** ACT_NOW / REVIEW / NOISE
- **Priority score:** 1-5 (highest signal, not additive)
- **Summary:** 1-2 lines
- **Matches:** OKR, task, or calendar match if any
- **Recommended action**

The background agent writes results to `classified.json` and archives all NOISE items via `gws gmail archive-thread <thread_id> --quiet`.

## Step 4: Collect Results

When the background agent completes, the main agent reads `classified.json` and presents the auto-action summary.

### Classification Categories

| Category | Criteria |
|----------|----------|
| **ACT_NOW** | Direct question to the user, approval/review request. **User MUST be the blocker** — not just CC'd. |
| **REVIEW** | FYI-relevant, decision context, CC'd on active thread, relates to OKR/task/meeting |
| **NOISE** | Gmail Promotions category, newsletters, automated alerts, digests, resolved comments |

**No SCHEDULING category.** Calendar invites and OOO replies are handled by the pre-filter script (`scripts/morning-prefilter.sh`). Any remaining scheduling-related emails should be classified as REVIEW.

**Flat classification — no promo/non-promo split.** All noise is treated the same regardless of source (fixes Issue #31).

### Blocker Detection (Critical Rule)

When classifying multi-message threads or CC'd emails, determine **who owns the next action**:
- **User is the blocker** → ACT_NOW (someone is waiting on the user)
- **User is CC'd, someone else is the blocker** → REVIEW (user should be aware, but no action needed now)
- **User is TO'd but the ask is to a group** → REVIEW (unless user is explicitly named)

This prevents over-scoring threads where the user is just an observer.

### Google Docs/Slides/Sheets Comment Notifications

Triage agents parse comment notification emails for:
1. **Resolution status** — "N resolved" → NOISE (auto-archive)
2. **Who replied** — if someone already answered → REVIEW
3. **Open + user expected to respond** → ACT_NOW
4. **Include comment link** — for "Open doc/comment" option in triage

### Noise Detection

When `noise_strategy` is `"promotions"`:
- Emails in Gmail's Promotions category are automatically classified as noise
- Gmail's ML categorization handles this — no manual sender lists needed

Additional noise signals:
- Duplicate invitations (same event, multiple notifications)
- Duplicate alert emails (same error, repeated)
- Auto-generated meeting notes where user was just an attendee
- Resolved comment notifications

### Priority Scoring (1-5)

Each triage agent scores its batch using the HIGHEST signal (not additive):

| Signal | Score | How to detect |
|--------|-------|---------------|
| **Top 5 match** | 5 | Email relates to a task in "Top five things" |
| **Overdue task link** | 5 | Email relates to an overdue task |
| **Must Win match** | 4 | Email maps to an OKR Must Win or Objective |
| **VIP sender + action** | 4 | From `vip_senders` AND requires action |
| **Meeting prep (today)** | 4 | Relates to a meeting happening today |
| **Starred** | 4 | Has STARRED label (if `priority_signals.starred` is true) |
| **Task match** | 3 | Relates to an active task |
| **VIP sender (FYI)** | 3 | From `vip_senders`, informational |
| **Action required** | 3 | Explicitly asks the user to do something |
| **Time sensitivity** | 3 | Deadline mentioned, thread waiting |
| **Thread momentum** | 2 | Active multi-message thread |
| **FYI peripheral** | 1 | Org-wide, tangentially related |
| **Noise** | 0 | Newsletters, alerts, personal |

### Determinism Rules (MUST, not "should")

- **MUST** use `in:inbox` in all Gmail queries
- **MUST** run pre-filter script before AI classification — OOO/invites are handled deterministically
- **MUST** use haiku for classification — sonnet is reserved for deep-dive only
- **MUST** auto-archive noise without asking the user (noise items from AI classification are archived by main agent)
- **MUST** use a single flat ID list — no promo/non-promo separation
- **MUST** run post-triage cleanup (Step 8)

## Step 5: Collect Results and Present Auto-Action Summary

Combine pre-filter results (from `auto_handled.json`) with AI classification. Archive any NOISE items from the AI classifier:

```bash
# For each NOISE-classified email from the haiku agent:
gws gmail archive-thread <thread_id> --quiet
```

Present a summary of all autonomous actions:

```
Auto-handled: <N> emails
  Pre-filter (OOO replies): <N>
  Pre-filter (calendar invites): <N>
  AI-classified noise: <N>

Needs your input: <N> items (<N> action, <N> review)
```

Update the header with final counts:

```
/morning — <Day>, <Date>

Inbox: <N> unread | <N> auto-handled | <N> need input
OKR focus: <primary track name> | <N> Must Wins active
Overdue tasks: <N>

Today's meetings:
  <time> <title> [related email if applicable]
  ...

Starting guided triage (<N> action items, then <N> review items).
Say "digest" for the full printout.
```

## Step 6: Guided Triage (Default Mode)

Walk through **only ACT_NOW + REVIEW items** one at a time, highest priority first. Noise and stale scheduling were already auto-handled by triage agents.

### Action Items (one by one)

For each action-required email, present:

```
[1/<N>] <Sender> — <Subject> (<N> msgs if thread)
<1-2 line context: why this matters, what's being asked>
[TOP 5: <task name>]
[OKR: <objective>]
```

Use AskUserQuestion with **4 options**. Pick the best 4 from the pool based on context:

**Standard options (always include):**
- **Mark as read** — Mark read, keep in inbox: `gws gmail label <message_id> --remove UNREAD --quiet`
- **Archive** — Remove from inbox: `gws gmail archive-thread <thread_id> --quiet`
- **Skip** — Move to next item (keeps email **unread**)

**Rotate the 4th slot based on context:**
- **Dig Deeper** — Spawn deep-dive sub-agent (for complex threads, action items)
- **Label & archive** — Spawn label-resolver sub-agent (`skills/morning/prompts/label-resolver.md`) with `action=archive`
- **Add task & archive** — Ask for title, run `gws tasks create`, then archive
- **Open in browser** — Run `open "https://mail.google.com/mail/u/0/#inbox/<thread_id>"`

Free-form responses via "Other":
- "delete" / "trash" → `gws gmail trash <message_id> --quiet`
- "add task" → ask for title, run `gws tasks create`
- "add task & archive" → create task, then archive thread
- "label X" / "label & archive" → spawn label-resolver sub-agent
- "star" → `gws gmail label <message_id> --add STARRED`

### Mark-as-Read Rule

After any action **except Skip**, mark the email as read:
```bash
gws gmail label <message_id> --remove UNREAD --quiet
```

**Skip is the only action that preserves unread status.**

**Note:** `archive-thread` already marks all messages as read.

For bulk mark-as-read:
```bash
skills/morning/scripts/bulk-gmail.sh mark-read <id1> <id2> ...
```

### Pacing Rule

**Do NOT auto-advance to the next email.** Wait for the user to say "next" or indicate readiness. The user controls the pace.

### Deep-Dive Sub-Agent (on "Dig Deeper")

Spawn a sub-agent to fetch, summarize, and cross-reference the email.

**Prompt file:** `skills/morning/prompts/deep-dive.md`
**Model:** `sonnet` — **always use sonnet** (haiku is unreliable for email reading)
**Agent type:** `general-purpose`

Pass: email ID, message count (for `read` vs `thread`), OKR/task/calendar context.

The sub-agent returns a structured brief. Present it and ask what to do next:
- **Open comment/doc** — open direct link (if available)
- **Reply** — compose a reply
- **Open in browser** — open in Gmail
- **Archive** — remove from inbox
- **Add task** — create a Google Task
- **Delete** — trash the email
- **Move on** — go to the next item

### Transition to Review Items

After all action items:

```
Action items done. <N> review items remaining.
```

Ask: "Continue reviewing?" with options:
- **Yes, keep going** — continue through review items
- **Done for now** — end triage

### Review Items (one by one, same pattern)

Same flow as action items but with lighter urgency framing.

### Triage Complete

```
Triage complete.
  Auto-handled: <N> | User acted on: <N> | Archived: <N> | Skipped: <N>
  Remaining unread: <N>
```

## Step 7: Digest Mode (Alternative)

If the user says "digest" at any point, print the full briefing without interaction:

```
━━ ACT NOW ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

<numbered list, highest priority first>
  <N>. <Sender> — <Subject> (<message_count> msgs)
      <1-line context>
      [TOP 5: <task name>] [OKR: <objective>]
      → <Suggested action>
      → gws gmail read <message_id>  OR  gws gmail thread <thread_id>

━━ REVIEW ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

<numbered list continuing from above>

━━ AUTO-HANDLED (<N>) ━━━━━━━━━━━━━━━━━━━━━━━
  Noise archived: <N> | Scheduling handled: <N> | Invites accepted: <N>
```

After the digest, remain ready for follow-up commands:

| User says | Action |
|-----------|--------|
| "read item N" | Run `gws gmail read <message_id>` or `gws gmail thread <thread_id>` |
| "archive items N, M" | Run `skills/morning/scripts/bulk-gmail.sh archive-thread <thread_ids>` |
| "add task: <title>" | Run `gws tasks create --title "<title>" --tasklist "Incoming"` |
| "triage" | Start guided triage from the beginning |

## Step 8: Post-Triage Cleanup (MANDATORY)

**MUST run after triage completes.** This step is not optional.

After triage (and daily log if configured), check for new arrivals:

```bash
gws gmail list --max 10 --query "is:unread in:inbox"
```

If new unread emails arrived during triage:
```
<N> new emails arrived during triage.
```

Ask: "Handle new arrivals?" with options:
- **Quick triage** — Spawn a single triage agent on just the new items
- **Ignore** — Leave for next session
- **Done** — End session

### Daily Log (if configured)

If `daily_log_doc_id` is set in config, append the briefing summary before the cleanup check:

```bash
gws docs append <daily_log_doc_id> --text "<summary>" --newline
```

Summary format:
```
## <Day>, <Date>

**Auto-handled:** <N> | **Action items:** <N> | **Reviewed:** <N>
**Overdue tasks:** <N>

### Priority items:
1. <item summary> (<OKR/task match>) — <action taken>
...

### Overdue:
- <task title> (<due date>)
```

If `daily_log_doc_id` is empty, create a new doc:

```bash
gws docs create --title "Morning Briefing Log"
```

Save the returned doc ID back to the config file for future runs.

## First-Run Setup

When `~/.config/gws/inbox-skill.yaml` does not exist, guide the user through setup:

### 1. OKR Sheet

Ask: "What is the Google Sheets ID for your OKR/planning document?"

Once provided, read sheet names:
```bash
gws sheets info <sheet_id>
```

Show the sheet names and ask which ones to monitor.

### 2. Task Lists

Read all task lists:
```bash
gws tasks lists
```

Show the list names and ask which ones to monitor. Options:
- **All** — monitor every list
- **Pick specific lists** — user selects which ones

Recommend including "Top five things" or equivalent priority list.

### 3. Noise Strategy

Ask: "How should noise emails be detected?"

Options:
- **Gmail Promotions (Recommended)** — Use Gmail's built-in Promotions category as the noise signal. This leverages Google's ML classification and avoids maintaining sender lists.
- **Custom sender list** — Manually specify sender patterns to classify as noise.

### 4. Priority Signals

Ask: "Which signals should boost email priority?"

- **Starred emails** — treat starred emails as high priority
- **VIP senders** — specify email addresses of key people (manager, direct reports, department heads, key partners)

For VIP senders, if an employee directory or org-chart lookup skill is available, offer to look up the user's org structure (manager, direct reports, department heads). The user can add VIP senders from the lookup results. This can also be done later during triage sessions.

<!-- Internal: when ported to taboola-pm-marketplace, use the employee lookup skill here to auto-populate org structure -->

### 5. Save Config

Write the config to `~/.config/gws/inbox-skill.yaml` in YAML format:

```yaml
okr_sheet_id: "<sheet_id>"
okr_sheets:
  - "<sheet_name>"

task_lists: "all"  # or list of specific names

noise_strategy: "promotions"

priority_signals:
  starred: true
  vip_senders:
    # Manager
    - manager@company.com
    # Direct reports
    - report1@company.com
    # Department heads
    - dept_head@company.com

inbox_query: "is:unread in:inbox"
max_unread: 50

daily_log_doc_id: ""
```

## CLI Quick Reference

Common `gws` commands used during triage:

| Action | Command |
|--------|---------|
| List unread | `gws gmail list --max 50 --query "is:unread in:inbox"` |
| Read message | `gws gmail read <message-id>` |
| Read thread | `gws gmail thread <thread-id>` |
| Archive thread | `gws gmail archive-thread <thread-id> --quiet` |
| Archive message | `gws gmail archive <message-id> --quiet` |
| Trash message | `gws gmail trash <message-id> --quiet` |
| Mark read | `gws gmail label <message-id> --remove UNREAD --quiet` |
| Star message | `gws gmail label <message-id> --add STARRED` |
| Add label | `gws gmail label <message-id> --add "<label>"` |
| List labels | `gws gmail labels` |
| Create task | `gws tasks create --title "<title>" --tasklist "<list>"` |
| Update task | `gws tasks update <tasklist-id> <task-id> --title "<title>"` |
| Calendar events | `gws calendar events --days 2` |
| RSVP accept | `gws calendar rsvp <event-id> --response accepted` |
| Bulk archive (thread) | `skills/morning/scripts/bulk-gmail.sh archive-thread <thread_id1> <thread_id2> ...` |
| Bulk archive (message) | `skills/morning/scripts/bulk-gmail.sh archive <id1> <id2> ...` |
| Bulk trash | `skills/morning/scripts/bulk-gmail.sh trash <id1> <id2> ...` |
| Bulk mark-read | `skills/morning/scripts/bulk-gmail.sh mark-read <id1> <id2> ...` |

**Use `--quiet` on archive/trash/label actions** to suppress JSON output and save context tokens.

## Tips for AI Agents

### Architecture (v0.4.0 — Script Pre-filter + Lean AI Classification)
- **Pipeline: prefetch → pre-filter → AI classify.** Two bash scripts handle deterministic work (data gathering + OOO/invite archiving). Only semantic classification uses AI. Principle: whatever can be not-AI should be not-AI.
- **Pre-filter script** (`scripts/morning-prefilter.sh`) archives OOO replies and calendar invite emails deterministically — no AI tokens spent on pattern matching.
- **Background agent pattern** — a background agent handles prefetch + pre-filter + classification while the main agent shows the user immediate context (config, calendar, meetings). The main agent polls the background agent's output file and narrates progress. This eliminates dead wait time — the user sees useful content immediately.
- **Sonnet reserved for deep-dive only** (when user says "Dig Deeper" during guided triage).
- **Sub-agent types:**
  - **Deep-dive agent** (Step 6, "Dig Deeper") — fetches and summarizes a single email/thread. **Always use sonnet** (haiku failed in QA).
  - **Label resolver** (Step 6, "Label & archive") — fetches label list, fuzzy-matches, applies labels
- **Use bash scripts for bulk operations.** Archive/trash/mark-read across multiple emails is mechanical work — no AI reasoning needed. Use `skills/morning/scripts/bulk-gmail.sh`:
  - `bulk-gmail.sh archive-thread <thread_ids>` — archive threads + mark read (preferred)
  - `bulk-gmail.sh archive <message_ids>` — archive messages + mark read
  - `bulk-gmail.sh trash <ids>` — delete + mark read
  - `bulk-gmail.sh mark-read <ids>` — mark read only
- **Main agent runs both scripts in Step 2**, then reads JSON files to build compact context for the haiku classifier.
- **Deprecated sub-agents:** `batch-classifier.md` and `calendar-coordinator.md` are superseded by `triage-agent.md`.

### Classification
- **Blocker detection is the most important classification rule.** An email where the user is CC'd and someone else owns the action is REVIEW, not ACT NOW — even if the thread is 5 weeks old and high-priority.
- When matching emails to OKRs, use semantic understanding — don't rely on exact keyword matches. An email about "cross-device identifiers" matches the OKR "Improve cross-domain identity mapping".
- Noise classification via Gmail Promotions is preferred over sender-based lists. Gmail's ML is more accurate and requires no maintenance.
- Duplicate detection matters: multiple invitations for the same event, repeated alert emails, and auto-generated notes should be deduplicated or grouped.
- **Single flat ID list** — no promo vs non-promo separation. All noise is treated equally to prevent missed archives.

### Guided Triage
- **Guided triage is the default.** Only ACT_NOW + REVIEW items reach the user — noise and stale scheduling are auto-handled.
- Use AskUserQuestion with **compound options** (Label & archive, Add task & archive). Never present single-action-only choices.
- The "Top five things" task list is the most important signal for priority scoring.
- For multi-message threads, mention the message count and latest sender.
- When the user picks "Dig Deeper", spawn a deep-dive sub-agent — do NOT dump raw email content into the main conversation.
- After the deep-dive returns, immediately ask what to do next. Don't wait for a free-form prompt.
- **Support "pause and work on this" flow.** Help open relevant docs/emails and offer to resume triage later.
- Keep each triage step focused — show ONE item at a time.
- Track actions taken (auto-handled + user actions) and report them at the end.
- Overdue tasks should always appear in the summary header.
- Calendar cross-referencing: "you have a 1:1 with X at 2pm, and X sent you an email" is actionable prep context.

### VIP Senders
- VIP sender lists can be populated during first-run setup using an employee directory lookup if available.
- During triage, if the user mentions wanting to track a new person, offer to add them to `vip_senders` in the config.

### Task Management
- `gws tasks update` modifies title, notes, or due date — it does NOT support moving tasks between lists or reordering. To move a task to a different list, create a new task in the target list and complete the old one.
- When creating follow-up tasks from triage, always ask the user which task list to use. Default to `@default` if they don't specify.

### Label Operations
- Gmail labels are resolved by **display name** (case-insensitive), not by internal ID.
- For label operations during triage, use the **label-resolver sub-agent** (`skills/morning/prompts/label-resolver.md`) to avoid loading the full label list (4000+ labels) into the main context.
- Common label patterns: `gws gmail label <id> --add "STARRED"`, `gws gmail label <id> --remove "UNREAD"`
