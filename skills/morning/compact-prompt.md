# /morning Compact Prompt

Use this when context runs out during a `/morning` triage session. Copy the template below, fill in the state from the conversation, and paste it to resume.

---

## Template

```
Resume a /morning inbox triage session. Run /gws:morning to load the skill, then continue from the saved state below.

## Config

~/.config/gws/inbox-skill.yaml is already configured. Key settings:
- OKR sheet: <sheet_id> — <sheet_names>
- VIP senders: <list>
- Noise strategy: promotions
- max_unread: <N>

## Triage Agent Results

Parallel triage agents already ran. Here are the merged results (do NOT re-fetch or re-classify):

### Auto-Handled (<N> items)
- Noise archived: <N>
- Stale scheduling archived: <N>
- Invites accepted (no conflict): <N>
- Past events archived: <N>

### ACT NOW (<N> items)
| # | ID | From | Subject | Priority | OKR/Task Match | Status |
|---|-----|------|---------|----------|----------------|--------|
| 1 | <message_id> | <sender> | <subject> | <1-5> | <match or —> | <DONE: action taken / PENDING> |
...

### REVIEW (<N> items)
| # | ID | From | Subject | Priority | OKR/Task Match | Status |
|---|-----|------|---------|----------|----------------|--------|
...

## Actions Taken So Far

| Email | Action | Marked Read |
|-------|--------|-------------|
| <sender — subject> | <archived / starred / task created / opened / deleted> | Yes/No |
...

Tasks created:
- "<task title>" in <list name>
...

## Resume Point

Continue from: **<CATEGORY> item [<N>/<TOTAL>]** — <sender — subject>
<If mid-category, note what was the last item shown and user's last response>

## Corrections Applied

<List any classification corrections the user made during triage, e.g.:>
- <none, or: "Bar Agam was REVIEW not ACT NOW — she's CC'd, privacy team owns it">

## Notes

<Any other context: user preferences, side tasks started, meetings coming up, etc.>
```

---

## How to Fill This In

1. **Triage results**: Copy the merged output from parallel triage agents. The auto-handled section shows what was archived/accepted automatically. Mark ACT NOW / REVIEW items as DONE (with action) or PENDING.
2. **Actions taken**: List every email you acted on — archive, star, task, open, delete. Note if marked read.
3. **Resume point**: Identify exactly where triage stopped — category, item number, and whether the user was mid-decision.
4. **Corrections**: Any time the user corrected a classification (e.g., "that's not ACT NOW, I'm CC'd"), note it so the resumed session doesn't repeat the mistake.
5. **Notes**: Meeting prep started, docs opened, side conversations — anything the user might want to pick back up.
