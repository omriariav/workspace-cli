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

task_lists:
  - "Top five things"
  - "Incoming"
  - "Taboola User Profile"
  - "Algo (Intent & User modeling)"
  - "Privacy"
  - "Attribution/Tracking/Measurements"

inbox_query: "is:unread"
max_emails: 50

daily_log_doc_id: ""    # Created on first run if empty

noise_senders:
  - "noreply@medium.com"
  - "noreply@linkedin.com"
  - "notification@github.com"
```

### First-Run Setup

When no config exists, the skill runs an interactive setup:

1. **OKR source** — reads sheet names from the configured spreadsheet, user picks which to monitor
2. **Task lists** — shows all task lists, user picks which to monitor
3. **Noise senders** — suggests common patterns, user can add custom ones
4. **Saves config** to `~/.config/gws/inbox-skill.yaml`

## Output Format

### Terminal Briefing

```
/morning — Mon Jan 27, 2026

Inbox: 23 unread | 3 action needed | 4 relevant | 16 noise
OKR focus: Data Track H1-2026 | 3 Must Wins active

━━ ACT NOW ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. ★ Jenny Liu — Yahoo/Taboola Sync canceled
   Meeting canceled (Corp Holiday). You own contextual signals.
   TOP 5: Intent research
   OKR: Improve cross-domain identity mapping
   → Reschedule the sync. Reply or create new invite.
   → gws gmail read 19bfd0a0fe192673

2. Peter Cimring / Aneil Pai — Legal question (7 msgs)
   Aneil replied yesterday. Thread active, may need your input.
   → Read latest and decide if follow-up needed.
   → gws gmail thread 19b4b3135a0732bb

3. JIRA DEV-209634 — TMT incident (2 msgs)
   Investigation thread, team may be waiting on direction.
   → Review and comment.
   → gws gmail thread 19b926796e7261a6

━━ REVIEW ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

4. Data Track all-hands — catering request (8 msgs)
   Logistics for your team event. No action unless change needed.
   → gws gmail read 19bfb267d937fa12

━━ TODAY'S MEETINGS (4) ━━━━━━━━━━━━━━━━━━━━━━
 9:00  Team standup
10:00  1:1 with Tomer Tunitsky
       ⚠ Related: annual review prep (overdue task)
14:00  Data Track sync
       📬 Prep: read JIRA DEV-209634 thread (item #3)
16:00  Intent research deep-dive
       📬 Prep: Yahoo/Taboola sync canceled (item #1)

━━ TASKS DUE ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ⚠ Tomer Tunitsky - annual review prep (due Jan 22 — overdue)
   ⚠ Adi Oz - annual review prep (due Jan 22 — overdue)

━━ NOISE (16) ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   8 newsletters | 5 JIRA watchers | 3 calendar auto-updates
   → Safe to bulk-archive
```

### Interactive Follow-Up

After the briefing, the user stays in the Claude session:

```
User: "read item 2"
→ Runs gws gmail thread 19b4b3135a0732bb

User: "archive the noise"
→ Runs gws gmail archive for each noise message_id

User: "add task: follow up with Jenny on sync reschedule"
→ Runs gws tasks create --title "..." --tasklist "Incoming"
```

### Daily Log (Google Doc)

Each briefing appends to a Google Doc:

```
## Mon Jan 27, 2026

**Action items:** 3 | **Relevant:** 4 | **Noise:** 16
**Overdue tasks:** 2

### Priority items:
1. Yahoo/Taboola Sync — reschedule (Intent research)
2. Legal question thread — review latest
3. TMT incident — review and comment

### Overdue:
- Tomer Tunitsky - annual review prep (Jan 22)
- Adi Oz - annual review prep (Jan 22)
```

## Implementation Phases

### P0: Core Briefing (this PR)
- [x] PLAN.md
- [ ] `skills/morning/SKILL.md` — skill definition with full workflow instructions
- [ ] First-run config setup flow
- [ ] Gather inbox (gmail list + thread for multi-message)
- [ ] Gather tasks (all configured lists)
- [ ] Gather calendar (today's events)
- [ ] Gather OKRs (configured sheets)
- [ ] AI classification and priority scoring
- [ ] Terminal output format
- [ ] Register skill in marketplace.json

### P1: Daily Log
- [ ] Create daily log Google Doc on first run
- [ ] Append briefing summary after each run

### P2: Interactive Follow-Up
- [ ] "read item N" → fetch and display
- [ ] "archive noise" → batch archive
- [ ] "add task" → create task in Incoming list

### P3: Generalize for Shipping
- [ ] Config-driven (no hardcoded sheet IDs)
- [ ] Documentation for other users
- [ ] Add to plugin marketplace

## Key Design Decisions

1. **Advisory only** — no auto-labeling, no auto-archiving. User executes recommended actions.
2. **OKR matching is semantic** — Claude reads the OKR sheet and uses judgment to match emails to objectives. No keyword lookup.
3. **Calendar cross-reference** — meetings boost priority of related emails and surface prep context.
4. **First-run setup** — interactive wizard creates config, so no manual YAML editing needed.
5. **Personal first** — hardcode to user's sheet/lists initially. Generalize in P3.
