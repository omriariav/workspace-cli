# `/morning` — Gmail AI Inbox Skill

## Problem Statement

Email is an unstructured, unprioritized input stream. Cognitive load is spent on triage instead of execution. No system maps the inbox to the user's actual priority framework (OKRs, active tasks, today's meetings) and tells them *what matters right now*.

## Product Concept

A Claude Code skill that reads the user's inbox, cross-references their OKRs (Google Sheets), active tasks (Google Tasks), and today's meetings (Google Calendar), then produces an actionable prioritized briefing.

Two modes:
- **Morning briefing** (`/morning`) — daily digest of unread inbox
- **On-demand triage** (`/inbox`) — anytime snapshot of current inbox state

Advisory only (v1) — recommends actions, user executes. Interactive follow-up in the same session.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌────────────────┐     ┌─────────────────┐
│  Gmail Inbox     │     │  OKR Sheet       │     │  Google Tasks   │     │  Google Calendar │
│  gws gmail list  │     │  gws sheets read │     │  gws tasks list │     │  gws calendar    │
│  gws gmail thread│     │                  │     │                 │     │  events          │
└────────┬────────┘     └────────┬─────────┘     └───────┬────────┘     └───────┬─────────┘
         │                       │                        │                      │
         └───────────┬───────────┴────────────────────────┴──────────────────────┘
                     ▼
           ┌─────────────────┐
           │  AI Classifier   │
           │  (Claude)        │
           │                  │
           │  - Categorize    │
           │  - Score priority│
           │  - Match to OKRs │
           │  - Match to tasks│
           │  - Cross-ref cal │
           └────────┬────────┘
                    ▼
         ┌─────────────────────┐
         │  Prioritized Brief  │
         │  Terminal + G Doc   │
         └─────────────────────┘
```

## Data Sources

| Source | Access | What it provides |
|--------|--------|-----------------|
| Gmail inbox | `gws gmail list`, `gws gmail thread` | Unread/recent emails, threads, senders |
| Google Tasks | `gws tasks lists`, `gws tasks list` | Active tasks, due dates, subtasks |
| Google Sheets | `gws sheets read` | OKR document — Must Wins, Objectives, Key Results, Initiatives, Status |
| Google Calendar | `gws calendar events --days 1` | Today's meetings for cross-referencing |

### OKR Sheet Structure

The OKR sheet (`Data Track (2026)`) has this hierarchy:

```
Track (col 1) → Sub Track/Product → Must Win → Objective → Key Result → Initiatives
```

Plus: Start Date, Due Date, Due Q, Status, KPIs, bi-weekly update columns.

### Task Lists

| List | Purpose |
|------|---------|
| "Top five things" | Highest priority — current focus items |
| "Incoming" | Triage inbox |
| Domain lists (Privacy, Attribution, Audience Toolkit, etc.) | Topic-specific tasks |

## Core Intelligence

### Email Classification

| Category | Signal | Example |
|----------|--------|---------|
| Action Required | Direct ask, question, approval request | "Can you review this PR?" |
| Decision Needed | Options presented, deadline mentioned | "We need to choose vendor by Friday" |
| FYI — Relevant | Relates to OKRs/tasks | Status update on a project you own |
| FYI — Peripheral | Org-wide, tangentially related | All-hands recap |
| Scheduling | Calendar invites, meeting changes | "Updated invitation: ..." |
| Noise | Newsletters, automated alerts, digests | Medium daily digest, JIRA watchers |

### Priority Scoring (1-5)

| Signal | Weight | Example |
|--------|--------|---------|
| **Top 5 match** | Highest | Email directly relates to a "Top five things" task |
| **Must Win match** | High | Email about sequence modeling → matches Must Win |
| **Task list match** | High | Email maps to an active task |
| **Sender signal** | Medium | Direct report, manager, cross-functional partner |
| **Action required** | Medium | Direct question, approval request, review ask |
| **Time sensitivity** | Medium | Explicit deadline, aging in inbox |
| **Thread momentum** | Low-Medium | Others waiting on your reply |
| **Meeting prep** | Medium | Email relates to a meeting happening today |
| **FYI/broadcast** | Low | All-hands, org-wide announcements |
| **Automated/noise** | Lowest | Newsletters, JIRA watchers, Medium digests |

### Task Matching

- Fuzzy match email subject/content against task titles
- Surface when an email creates an implicit task that doesn't exist yet
- Flag overdue tasks even if no matching email exists

### Calendar Cross-Reference

- Today's meetings shown with prep context
- Emails from meeting attendees get a priority boost
- Overdue tasks linked to meeting participants are flagged

## Configuration

```yaml
# ~/.config/gws/inbox-skill.yaml
okr_sheet_id: "14qwO-5DxkVT1GfzxB6DLuwmS_tJ113g-fnYL5yMjhMA"
okr_sheets:
  - "Data Track (2026)"

task_lists: "all"   # or list of specific names

noise_strategy: "promotions"   # uses Gmail's Promotions category

priority_signals:
  starred: true
  vip_senders:
    - manager@company.com
    - report1@company.com
    - dept_head@company.com

inbox_query: "is:unread"
max_emails: 50

daily_log_doc_id: ""    # Created on first run if empty
```

### First-Run Setup

When no config exists, the skill runs an interactive setup:

1. **OKR source** — reads sheet names from the configured spreadsheet, user picks which to monitor
2. **Task lists** — shows all task lists, user picks which to monitor (or "all")
3. **Noise strategy** — recommend Gmail Promotions category over manual sender lists
4. **Priority signals** — starred emails, VIP senders (populate via `/taboolar` org lookup if available)
5. **Saves config** to `~/.config/gws/inbox-skill.yaml`

## Interaction Model

### Default: Guided Triage

The default mode walks the user through items one at a time using AskUserQuestion, so they never lose context.

**Flow:**

1. **Batch classification** — spawn sub-agent with all email snippets + OKR/tasks/calendar context. Returns structured classification for every email.
2. **Summary header** — compact overview (inbox counts, today's meetings, overdue tasks). ~15 lines max.
3. **Action items** — one at a time, each with options: Read it, Open in browser, Archive, Add task, Skip.
   - If "Read it": spawn deep-dive sub-agent to fetch and summarize, then ask: Reply, Archive, Add task, Open in browser, Move on.
3. **Transition** — "Action items done. Continue reviewing?" (Yes / Skip to noise / Done)
4. **Review items** — same one-at-a-time pattern with lighter urgency.
5. **Noise handling** — "N noise items. Archive all?" (Archive all / Let me pick / Leave them)
6. **Triage complete** — summary of actions taken.

### Alternative: Digest Mode

User says "digest" at any point to get the full printout:

```
━━ ACT NOW ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. ★ <Sender> — <Subject>
   <context>
   → gws gmail read <message_id>
...
━━ REVIEW ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
...
━━ NOISE (<N>) ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

After the digest, follow-up via free-form commands: "read item N", "archive noise", "add task: ...".

### Daily Log (Google Doc)

Each briefing appends a summary to a Google Doc:

```
## Mon Jan 27, 2026

**Action items:** 3 | **Reviewed:** 4 | **Noise:** 16 (archived: 16)
**Overdue tasks:** 2

### Priority items:
1. Yahoo/Taboola Sync — reschedule (Intent research) — read
2. Legal question thread — review latest — skipped
3. TMT incident — review and comment — added task

### Overdue:
- Tomer Tunitsky - annual review prep (Jan 22)
- Adi Oz - annual review prep (Jan 22)
```

## Implementation Phases

### P0: Core Skill (this PR)
- [x] PLAN.md
- [x] `skills/morning/SKILL.md` — skill definition with full workflow instructions
- [x] Guided triage flow with AskUserQuestion
- [x] Digest mode as alternative
- [x] Register skill in marketplace.json
- [x] First-run config setup flow (noise strategy, VIP senders, starred signal)
- [x] Sub-agent architecture: batch classifier + per-item deep-dive
- [x] Blocker detection: distinguish "you own the action" vs "you're CC'd"
- [x] Gmail Promotions as noise signal (replaces sender-based lists)
- [ ] Live testing — complete full triage cycle
- [ ] Daily log integration

### P1: Daily Log
- [ ] Create daily log Google Doc on first run
- [ ] Append briefing summary after each run

### P2: Generalize for Shipping
- [ ] Config-driven (no hardcoded sheet IDs)
- [ ] Documentation for other users
- [ ] Plugin distribution

## Key Design Decisions

1. **Guided triage by default** — one item at a time with options. Digest mode available on demand.
2. **User controls all actions** — Claude recommends, user executes via AskUserQuestion choices.
3. **OKR matching is semantic** — Claude reads the OKR sheet and uses judgment to match emails to objectives. No keyword lookup.
4. **Calendar cross-reference** — meetings boost priority of related emails and surface prep context.
5. **First-run setup** — interactive wizard creates config, so no manual YAML editing needed.
6. **Personal first** — hardcode to user's sheet/lists initially. Generalize in P2.
7. **Sub-agent architecture** — batch classifier for initial scoring, per-item deep-dive on "Read it". Keeps main conversation lean and prevents context overflow.
8. **Blocker detection** — the most important classification rule. Emails where the user is CC'd and someone else owns the action are REVIEW, not ACT NOW.
9. **Gmail Promotions as noise** — replaces manual sender-based lists. Gmail's ML categorization is more accurate and requires no maintenance.
10. **VIP senders from org data** — `/taboolar` integration populates manager, reports, dept heads as priority signals during setup.
11. **Pause-and-resume** — user can stop triage to work on something (prep for a meeting, open a doc), then resume later.
