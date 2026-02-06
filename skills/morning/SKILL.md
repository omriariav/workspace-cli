---
name: gws-morning
version: 0.2.0
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
- `inbox_query` — Gmail search query (default: `"is:unread in:inbox"`)
- `max_emails` — How many unread emails to analyze (default 50)
- `daily_log_doc_id` — Google Doc ID for the daily log (empty = skip logging)

## Step 2: Gather Context for Summary Header

The main agent gathers **only calendar data** for the summary header. All other data (inbox, tasks, OKRs) is gathered by the batch classifier sub-agent in Step 3 to keep raw JSON out of the main context.

### Calendar

```bash
gws calendar events --days 2
```

Fetch **2 days** (today + tomorrow) for meeting prep context. Extract: event title, start time, attendees, description. These are used to:
- Populate the summary header with today's meetings
- Surface prep context ("you have a meeting about X, review email Y first")
- Tomorrow's meetings provide early prep opportunities

**Do NOT** gather inbox, tasks, or OKRs here. The batch classifier sub-agent handles all data gathering and classification in a single call (Step 3).

## Step 3: Classify Emails (Batch Sub-Agent)

**Do not classify emails inline.** Spawn a sub-agent to gather all data and classify all primary emails in a single batch. The sub-agent runs `gws` commands itself, keeping raw JSON out of the main conversation context.

**Prompt file:** `skills/morning/prompts/batch-classifier.md`
**Model:** `sonnet` | **Agent type:** `general-purpose`

Follow the prompt template in the file. Pass config values only:
- `max_emails` — from config
- Task list IDs — from config (`task_lists`)
- OKR sheet ID and sheet names — from config
- VIP senders — from config (`priority_signals.vip_senders`)
- Noise strategy — from config
- `inbox_query` — from config (default: `"is:unread in:inbox"`)

The sub-agent fetches inbox, tasks, and OKRs itself. **Do NOT pass raw data.**

### Sub-Agent Output

The sub-agent returns a structured classification for each email:

```
ID | CATEGORY | PRIORITY (1-5) | SUMMARY (2-3 lines) | OKR_MATCH | TASK_MATCH | CALENDAR_MATCH | SUGGESTED_ACTION
```

Grouped into: **ACT NOW**, **REVIEW**, **SCHEDULING**, **PERIPHERAL**, **NOISE**.

### Classification Categories

| Category | Criteria |
|----------|----------|
| **Action Required** | Direct question to the user, approval/review request, explicit ask. **The user must be the blocker** — not just CC'd on a thread where someone else owns the action. |
| **Decision Needed** | Options presented, deadline mentioned, waiting for user's call |
| **FYI — Relevant** | Relates to an OKR objective, active task, or today's meeting |
| **FYI — Peripheral** | Org-wide, tangentially related, informational |
| **Scheduling** | Calendar invites, meeting updates, reschedules |
| **Noise** | Gmail Promotions category, newsletters, automated alerts, digests |
| **Personal** | Non-work personal emails (payments, account notifications) |

### Blocker Detection (Critical Rule)

When classifying multi-message threads or CC'd emails, determine **who owns the next action**:
- **User is the blocker** → ACTION_REQUIRED (someone is waiting on the user)
- **User is CC'd, someone else is the blocker** → REVIEW (user should be aware, but no action needed now)
- **User is TO'd but the ask is to a group** → REVIEW (unless user is explicitly named)

This prevents over-scoring threads where the user is just an observer.

### Google Docs/Slides/Sheets Comment Notifications

When the email is a Google Workspace comment notification (from `comments-noreply@docs.google.com` or `drive-shares-dm-noreply@google.com`), the deep-dive sub-agent must parse the email body for:

1. **Comment resolution status** — look for "N resolved" in the email body. If the comment is resolved, it's NOT an action item.
2. **Who replied** — check if someone already answered the question. If a direct report or colleague already responded, the user may not need to act.
3. **Open vs. resolved** — only classify as ACTION_REQUIRED if the comment is still open AND the user is expected to respond.
4. **Suggest opening the comment link** — the email contains a direct link to the comment in the document. Offer to open that link (not just the email).
5. **Propose an answer** — if the comment is still open and asks a question the user can answer, draft a suggested response using available OKR/task context.

Example classification:
- Comment resolved by someone else → **FYI — Peripheral** ("Tomer already answered, Vib resolved it")
- Comment open, user is asked a question → **Action Required** ("Yaniv asks about geo availability, here's a suggested answer based on your OKR data...")

### Noise Detection

When `noise_strategy` is `"promotions"`:
- Emails in Gmail's Promotions category are automatically classified as noise
- No manual sender-based filtering needed — Gmail's ML categorization handles this

Additional noise signals (regardless of strategy):
- Duplicate invitations (same event, multiple notifications)
- Duplicate alert emails (same error, repeated)
- Auto-generated meeting notes where user was just an attendee (Gemini notes)
- Google Docs/Slides/Sheets comment notifications where the comment is already resolved

## Step 4: Score Priority

Priority scoring is done by the batch sub-agent as part of Step 3. Pass these rules to the sub-agent.

Each actionable email (not noise) gets scored 1-5. Use these signals:

| Signal | Score | How to detect |
|--------|-------|---------------|
| **Top 5 match** | 5 | Email subject/content relates to a task in "Top five things" |
| **Overdue task link** | 5 | Email relates to an overdue task |
| **Must Win match** | 4 | Email topic maps to an OKR Must Win or Objective |
| **VIP sender + action** | 4 | Email is from a `vip_senders` address AND requires action |
| **Meeting prep (today)** | 4 | Email relates to a meeting happening today |
| **Starred** | 4 | Email has the STARRED label (if `priority_signals.starred` is true) |
| **Task match** | 3 | Email relates to an active task in any monitored list |
| **VIP sender (FYI)** | 3 | Email is from a `vip_senders` address, informational |
| **Action required** | 3 | Email explicitly asks the user to do something |
| **Time sensitivity** | 3 | Deadline mentioned, or thread has been waiting |
| **Thread momentum** | 2 | Multi-message thread where others are actively discussing |
| **FYI peripheral** | 1 | Org-wide, tangentially related |
| **Noise / personal** | 0 | Newsletters, alerts, personal account notifications |

**Score aggregation:** When multiple signals apply, use the highest signal score (not additive). A Top 5 match with VIP sender is still priority 5, not 9.

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

Then ask the user what to do using AskUserQuestion (max 4 options). Rotate options based on context:

**For action/review items:**
- **Dig Deeper** — Spawn deep-dive sub-agent (see below)
- **Open in browser** — Run `open "https://mail.google.com/mail/u/0/#inbox/<thread_id>"`
- **Archive** — Run `gws gmail archive-thread <thread_id> --quiet` (archives all messages in thread + marks read)
- **Skip** — Move to next item (keeps email **unread**)

**Compound options** (rotate based on context):
- **Label & archive** — Spawn label-resolver sub-agent (`skills/morning/prompts/label-resolver.md`) with `action=archive`
- **Add task & archive** — Ask for title, run `gws tasks create`, then archive the thread
- **Accept & archive** — For scheduling items, RSVP accept + archive (via calendar-coordinator)

**For noise/peripheral items:**
- **Archive** — Run `gws gmail archive-thread <thread_id> --quiet`
- **Delete** — Run `gws gmail trash <message_id> --quiet`
- **Open in browser** — Run `open "https://mail.google.com/mail/u/0/#inbox/<thread_id>"`
- **Skip** — Move to next item (keeps email **unread**)

The user can always type a free-form response (e.g., "delete", "add task", "star this") via the "Other" option. Handle these naturally:
- "delete" / "trash" → `gws gmail trash <message_id> --quiet`
- "add task" → ask for title, run `gws tasks create`
- "add task & archive" → create task, then `gws gmail archive-thread <thread_id> --quiet`
- "label X" / "label & archive" → spawn label-resolver sub-agent
- "star" → `gws gmail label <message_id> --add STARRED`
- "open" → open in browser

### Mark-as-Read Rule

After any action **except Skip**, mark the email as read:
```bash
gws gmail label <message_id> --remove UNREAD --quiet
```

**Skip is the only action that preserves unread status.** This ensures triaged emails don't reappear as unread in the next run.

**Note:** `archive-thread` already marks all messages as read, so no separate mark-read step is needed when archiving threads.

For bulk mark-as-read (e.g., after archiving multiple items), use the bulk script:
```bash
skills/morning/scripts/bulk-gmail.sh mark-read <id1> <id2> ...
```

### Pacing Rule

**Do NOT auto-advance to the next email.** After the user takes an action (or the deep-dive brief is presented and acted on), wait for the user to say "next" or otherwise indicate they're ready. The user controls the pace — they may want to discuss, take a side action, or pause before continuing.

### Deep-Dive Sub-Agent (on "Dig Deeper")

Spawn a sub-agent to fetch, summarize, and cross-reference the email.

**Prompt file:** `skills/morning/prompts/deep-dive.md`
**Agent type:** `general-purpose`

**Model selection by complexity:**
- **`haiku`** — Single-message emails, FYI items, newsletters, simple questions (fast + cheap)
- **`sonnet`** — Multi-message threads (3+ messages), comment notifications with resolution context, action items with cross-references

Choose the model when spawning the deep-dive agent based on the email's message count and classification category.

Follow the prompt template in the file. Pass it the email ID, message count (to decide `read` vs `thread`), and the OKR/task/calendar context for cross-referencing.

The sub-agent returns a structured brief. **The main agent presents the brief** and asks what to do next:
- **Open comment/doc** — open the direct link to the document or comment (if available)
- **Reply** — compose a reply (or post the suggested answer)
- **Open in browser** — open in Gmail for manual handling
- **Archive** — remove from inbox
- **Add task** — create a Google Task
- **Delete** — trash the email
- **Move on** — go to the next item

This keeps the main conversation lean — only the structured brief enters the main context, not the raw email content.

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

Same flow as action items but with lighter urgency framing. Same option rotation and free-form handling applies. Mark-as-read rule: any action except Skip marks the email as read.

### Scheduling Step

After review items, handle all SCHEDULING category emails together:

```
<N> scheduling items (calendar invites, meeting updates).
```

Ask: "Handle scheduling items?" with options:
- **Auto-accept all** — Spawn calendar-coordinator sub-agent (`skills/morning/prompts/calendar-coordinator.md`) with all scheduling items. It RSVPs, checks conflicts, and archives handled items. Returns a structured summary.
- **One by one** — Walk through scheduling items individually with options: Accept & archive, Decline, Open in browser, Skip
- **Skip** — Leave all scheduling items unread

### Noise Handling

After action and review items (or when user skips to noise), present a **numbered list** so the user can select by number:

```
<N> noise items:
  1. <Sender> — <Subject> (newsletter)
  2. <Sender> — <Subject> (automated alert)
  3. <Sender> — <Subject> (JIRA watcher)
  ...
```

Ask: "Archive all noise, or select by number?" with options:
- **Archive all** — Run: `skills/morning/scripts/bulk-gmail.sh archive-thread <thread_id1> <thread_id2> ...` (archives + marks read in one pass)
- **Delete all** — Run: `skills/morning/scripts/bulk-gmail.sh trash <id1> <id2> ...`
- **Let me pick** — User can type "archive 1,3,5" or "keep 2,4" to selectively handle items
- **Leave them** — Skip, do nothing

When the user selects by number (e.g., "archive 2,5,8"), archive only those items and leave the rest unread.

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
| "archive the noise" | Run `skills/morning/scripts/bulk-gmail.sh archive-thread <noise_thread_ids>` |
| "archive items N, M" | Run `skills/morning/scripts/bulk-gmail.sh archive-thread <selected_thread_ids>` |
| "delete the noise" | Run `skills/morning/scripts/bulk-gmail.sh trash <noise_ids>` |
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

## Step 9: Post-Triage Cleanup

After triage completes (and daily log is written), check for new arrivals:

```bash
gws gmail list --max 10 --query "is:unread in:inbox"
```

If new unread emails arrived during triage:
```
<N> new emails arrived during triage.
```

Ask: "Handle new arrivals?" with options:
- **Quick classify** — Run the batch classifier on just the new items (lighter pass)
- **Ignore** — Leave for next session
- **Done** — End session

This prevents the "endless inbox" problem where emails arrive faster than they're triaged.

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
max_emails: 50

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

### Architecture
- **Use sub-agents to manage context.** The main agent orchestrates; sub-agents do the heavy lifting:
  - **Batch classifier** (Step 3) — classifies all emails in one call, returns structured data
  - **Deep-dive agent** (Step 6, "Dig Deeper") — fetches and summarizes a single email/thread
  - **Calendar coordinator** (Step 6, "Scheduling") — matches invites to events, checks conflicts, RSVPs
  - **Label resolver** (Step 6, "Label & archive") — fetches label list, fuzzy-matches, applies labels
  - This keeps the main conversation lean and prevents context overflow on large inboxes.
- **Use bash scripts for bulk operations.** Archive/trash/mark-read across multiple emails is mechanical work — no AI reasoning needed. Use `skills/morning/scripts/bulk-gmail.sh` instead of spawning a sub-agent or running sequential commands:
  - `bulk-gmail.sh archive-thread <thread_ids>` — archive all messages in threads + mark read (preferred)
  - `bulk-gmail.sh archive <message_ids>` — archive individual messages + mark read
  - `bulk-gmail.sh trash <ids>` — delete + mark read
  - `bulk-gmail.sh mark-read <ids>` — mark read only
- The batch classifier sub-agent gathers its own data (inbox, tasks, OKRs). The main agent only gathers calendar data (Step 2) for the summary header. Pass config values — not raw data — when spawning the sub-agent.

### Classification
- **Blocker detection is the most important classification rule.** An email where the user is CC'd and someone else owns the action is REVIEW, not ACT NOW — even if the thread is 5 weeks old and high-priority.
- When matching emails to OKRs, use semantic understanding — don't rely on exact keyword matches. An email about "cross-device identifiers" matches the OKR "Improve cross-domain identity mapping".
- Noise classification via Gmail Promotions is preferred over sender-based lists. Gmail's ML is more accurate and requires no maintenance.
- Duplicate detection matters: multiple invitations for the same event, repeated alert emails, and auto-generated notes should be deduplicated or grouped.

### Guided Triage
- **Guided triage is the default.** Use AskUserQuestion to present each item with options. Only switch to digest mode if the user explicitly asks.
- The "Top five things" task list is the most important signal for priority scoring.
- For multi-message threads, mention the message count and latest sender to give the user context on thread activity.
- When the user picks "Dig Deeper", spawn a deep-dive sub-agent — do NOT dump raw email content into the main conversation.
- After the deep-dive returns, immediately ask what to do next (Reply, Archive, Add task, Open in browser, Move on). Don't wait for a free-form prompt.
- **Support "pause and work on this" flow.** If the user says they want to look into something now (e.g., prep for an upcoming meeting), help them open relevant docs/emails in the browser and offer to resume triage later.
- Keep each triage step focused — show ONE item at a time. Never dump multiple items between questions.
- Track actions taken (archived, read, tasks created) and report them at the end.
- Overdue tasks should always appear in the summary header even if there's no matching email.
- Calendar cross-referencing is valuable: "you have a 1:1 with X at 2pm, and X sent you an email" is actionable prep context.

### VIP Senders
- VIP sender lists can be populated during first-run setup using an employee directory lookup if available.
- During triage, if the user mentions wanting to track a new person, offer to add them to `vip_senders` in the config.

### Task Management
- `gws tasks update` modifies title, notes, or due date — it does NOT support moving tasks between lists or reordering. To move a task to a different list, create a new task in the target list and complete the old one.
- When creating follow-up tasks from triage, always ask the user which task list to use. Default to `@default` if they don't specify.

### Label Operations
- Gmail labels are resolved by **display name** (case-insensitive), not by internal ID. Use `gws gmail labels` to see all available label names.
- For label operations during triage, use the **label-resolver sub-agent** (`skills/morning/prompts/label-resolver.md`) to avoid loading the full label list (4000+ labels) into the main context.
- Common label patterns: `gws gmail label <id> --add "STARRED"`, `gws gmail label <id> --remove "UNREAD"`
