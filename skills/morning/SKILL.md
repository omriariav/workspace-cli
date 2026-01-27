---
name: gws-morning
version: 0.1.0
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
- `task_lists` — Which Google Task lists to monitor
- `noise_senders` — Email senders to auto-classify as noise
- `max_emails` — How many unread emails to analyze (default 50)
- `daily_log_doc_id` — Google Doc ID for the daily log (empty = skip logging)

## Step 2: Gather Data

Run these `gws` commands to collect all context. Run them in parallel where possible.

### 2a. Inbox

```bash
gws gmail list --max <max_emails> --query "is:unread"
```

This returns threads with `thread_id`, `message_id`, `message_count`, `subject`, `from`, `date`, `snippet`.

For threads with `message_count > 1`, fetch the full thread to understand conversation context:

```bash
gws gmail thread <thread_id>
```

### 2b. Tasks

For each task list in the config:

```bash
gws tasks list <task-list-id>
```

Pay special attention to:
- Tasks in "Top five things" — these are the user's current highest priorities
- Tasks with due dates that are past or today — flag as overdue
- Subtasks (tasks with `parent` field) — group under their parent

### 2c. Calendar

```bash
gws calendar events --days 1
```

Extract: event title, start time, attendees, description. These are used to:
- Cross-reference with inbox items (emails from attendees, about meeting topics)
- Surface prep context ("you have a meeting about X, review email Y first")

### 2d. OKRs

For each sheet in `okr_sheets`:

```bash
gws sheets read <okr_sheet_id> "<sheet_name>!A1:Q100"
```

Extract the OKR hierarchy:
- **Must Wins** — the top-level strategic bets
- **Objectives** — quarterly goals under each Must Win
- **Key Results** — measurable milestones
- **Initiatives** — current work items
- **Status** — On Track, At Risk, Not Started
- **Recent updates** — the latest bi-weekly update column with content

## Step 3: Classify Emails

For each unread email, classify into one of these categories:

| Category | Criteria |
|----------|----------|
| **Action Required** | Direct question to the user, approval/review request, explicit ask |
| **Decision Needed** | Options presented, deadline mentioned, waiting for user's call |
| **FYI — Relevant** | Relates to an OKR objective, active task, or today's meeting |
| **FYI — Peripheral** | Org-wide, tangentially related, informational |
| **Scheduling** | Calendar invites, meeting updates, reschedules |
| **Noise** | Sender matches `noise_senders`, or is: newsletter, automated alert, digest, JIRA watcher notification |

### Noise Detection

Auto-classify as noise if:
- Sender matches any pattern in `noise_senders` config
- Email is from a mailing list with no direct mention of the user
- Email is an automated notification (JIRA watcher, GitHub notification, build alerts) that doesn't require action
- Email is a newsletter or digest

## Step 4: Score Priority

Each actionable email (not noise) gets scored. Use these signals:

| Signal | Weight | How to detect |
|--------|--------|---------------|
| **Top 5 match** | Highest | Email subject/content relates to a task in "Top five things" |
| **Must Win match** | High | Email topic maps to an OKR Must Win or Objective |
| **Task match** | High | Email relates to an active task in any monitored list |
| **Meeting prep** | Medium-High | Email relates to a meeting happening today |
| **Sender is attendee** | Medium | Email sender is an attendee of today's meeting |
| **Action required** | Medium | Email explicitly asks the user to do something |
| **Time sensitivity** | Medium | Deadline mentioned, or thread has been waiting |
| **Thread momentum** | Low-Medium | Multi-message thread where others are actively discussing |
| **Overdue task link** | High | Email relates to an overdue task |

## Step 5: Summary Header

Start with a compact summary header so the user sees the big picture before diving in:

```
/morning — <Day>, <Date>

Inbox: <N> unread | <N> action needed | <N> review | <N> noise
OKR focus: <primary track name> | <N> Must Wins active
Overdue tasks: <N>

Today's meetings:
  <time> <title> [⚠ overdue task / 📬 related email if applicable]
  ...

Starting guided triage (<N> action items, then <N> review items).
Say "digest" for the full printout, or "skip" to jump to noise.
```

Keep this short — no more than 15-20 lines. The detail comes in the guided triage.

## Step 6: Guided Triage (Default Mode)

Walk through items one at a time, highest priority first. For each item, use the AskUserQuestion tool to present options.

### Action Items (one by one)

For each action-required email, present it as a question:

```
[1/<N>] ★ <Sender> — <Subject> (<N> msgs if thread)
<1-2 line context: why this matters, what's being asked>
[TOP 5: <task name>]
[OKR: <objective>]
```

Then ask the user what to do using AskUserQuestion with these options:
- **Read it** — Run `gws gmail read <message_id>` or `gws gmail thread <thread_id>`, show content, then ask again what to do next (Reply, Archive, Add task, Move on)
- **Archive** — Run `gws gmail archive <message_id>`
- **Add task** — Ask for task title, run `gws tasks create`
- **Skip** — Move to next item

After the user acts or skips, move to the next action item.

### Transition to Review Items

After all action items:

```
Action items done. <N> review items remaining.
```

Ask: "Continue reviewing?" with options:
- **Yes, keep going** — continue guided triage through review items
- **Skip to noise** — jump to noise handling
- **Done for now** — end triage

### Review Items (one by one, same pattern)

Same flow as action items but with lighter urgency framing. Options:
- **Read it** — fetch and display
- **Archive** — remove from inbox
- **Skip** — move on

### Noise Handling

After action and review items (or when user skips to noise):

```
<N> noise items:
  <N> newsletters | <N> automated alerts | <N> scheduling | <N> other
```

Ask: "Archive all noise?" with options:
- **Archive all** — Run `gws gmail archive` for each noise message_id
- **Let me pick** — Show noise items one by one, user decides each
- **Leave them** — Skip, do nothing

### Triage Complete

```
Triage complete.
  Acted on: <N> | Archived: <N> | Skipped: <N>
  Remaining unread: <N>
```

## Step 7: Digest Mode (Alternative)

If the user says "digest" at any point, switch to printing the full briefing without interaction:

```
━━ ACT NOW ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

<numbered list of action-required emails, highest priority first>
Each item:
  <N>. [★ if Top 5 match] <Sender> — <Subject> (<message_count> msgs if thread)
      <1-line context: why this matters>
      [TOP 5: <task name> if matched]
      [OKR: <objective name> if matched]
      → <Suggested action>
      → gws gmail read <message_id>  OR  gws gmail thread <thread_id>

━━ REVIEW ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

<numbered list continuing from above, FYI-relevant items>

━━ NOISE (<N>) ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  <N> newsletters | <N> JIRA watchers | <N> calendar auto-updates
  → Safe to bulk-archive
```

After the digest, remain ready for follow-up commands:

| User says | Action |
|-----------|--------|
| "read item N" | Run `gws gmail read <message_id>` or `gws gmail thread <thread_id>` for that item |
| "archive the noise" | Run `gws gmail archive <message_id>` for each noise email |
| "archive items N, M" | Run `gws gmail archive` for specified items |
| "add task: <title>" | Run `gws tasks create --title "<title>" --tasklist "Incoming"` |
| "triage" | Start guided triage from the beginning |

## Step 8: Daily Log (if configured)

If `daily_log_doc_id` is set in config, append the briefing summary after triage completes:

```bash
gws docs append <daily_log_doc_id> --text "<summary>" --newline
```

Summary format:
```
## <Day>, <Date>

**Action items:** <N> | **Reviewed:** <N> | **Noise:** <N> (archived: <N>)
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

Show the list names and ask which ones to monitor. Recommend including "Top five things" or equivalent priority list.

### 3. Noise Senders

Suggest common noise patterns:
- `noreply@medium.com`
- `noreply@linkedin.com`
- `notification@github.com`
- `digest-noreply@quora.com`

Ask the user if they want to add more patterns (e.g., JIRA watcher address, internal notification systems).

### 4. Save Config

Write the config to `~/.config/gws/inbox-skill.yaml` in YAML format.

## Tips for AI Agents

- Always run all data-gathering commands (Step 2) before classification. You need the full picture.
- **Guided triage is the default.** Use AskUserQuestion to present each item with options. Only switch to digest mode if the user explicitly asks.
- The "Top five things" task list is the most important signal for priority scoring.
- When matching emails to OKRs, use semantic understanding — don't rely on exact keyword matches. An email about "cross-device identifiers" matches the OKR "Improve cross-domain identity mapping".
- For multi-message threads, mention the message count and latest sender to give the user context on thread activity.
- When the user picks "Read it", fetch the email/thread, display it, then immediately ask what to do next (Reply, Archive, Add task, Move on). Don't wait for a free-form prompt.
- Noise classification should be generous — when in doubt, classify as noise. Users prefer to review a few false positives than miss important emails.
- Overdue tasks should always appear in the summary header even if there's no matching email.
- Calendar cross-referencing is valuable: "you have a 1:1 with X at 2pm, and X sent you an email" is actionable prep context.
- Keep each triage step focused — show ONE item at a time. Never dump multiple items between questions.
- Track actions taken (archived, read, tasks created) and report them at the end.
