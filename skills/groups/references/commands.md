# Groups Commands Reference

Complete flag and option reference for `gws groups` commands -- 2 commands total.

> **Disclaimer:** `gws` is not the official Google CLI. This is an independent, open-source project not endorsed by or affiliated with Google.

## Global Flags

These flags apply to all `gws groups` commands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.config/gws/config.yaml` | Config file path |
| `--format` | string | `json` | Output format: `json`, `yaml`, or `text` |
| `--quiet` | bool | `false` | Suppress output (useful for scripted actions) |

---

## gws groups list

Lists Google Groups in your domain.

```
Usage: gws groups list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int64 | 50 | Maximum number of groups to return |
| `--domain` | string | | Filter by domain |
| `--user-email` | string | | Filter groups for a specific user |

**Constraint:** `--domain` and `--user-email` are mutually exclusive. If neither is provided, groups for the entire customer account are returned.

### Output Fields (JSON)

Returns an object with:
- `groups` -- Array of group objects
- `count` -- Number of groups returned

Each group includes:
- `id` -- Group identifier
- `email` -- Group email address
- `name` -- Group display name
- `description` -- Group description (only present if set)
- `member_count` -- Number of direct members (only present if non-zero)

### Examples

```bash
# List default 50 groups
gws groups list

# List up to 200 groups
gws groups list --max 200

# Filter by domain
gws groups list --domain example.com

# Filter groups a specific user belongs to
gws groups list --user-email alice@example.com

# List groups with text output
gws groups list --format text

# Extract just names and emails
gws groups list --format json | jq '.groups[] | {name, email}'

# Find groups with more than 10 members
gws groups list --format json | jq '.groups[] | select(.member_count > 10)'
```

### Notes

- Requires Admin SDK API enabled in the Google Cloud project
- Requires Google Workspace admin privileges
- When no filter is specified, uses `customer=my_customer` to list all groups in the organization
- The `--domain` flag restricts results to groups in a specific domain
- The `--user-email` flag returns only groups the specified user belongs to

---

## gws groups members

Lists members of a Google Group by group email address.

```
Usage: gws groups members <group-email> [flags]
```

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `group-email` | string | Yes | Group email address |

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int64 | 50 | Maximum number of members to return |
| `--role` | string | | Filter by role: `OWNER`, `MANAGER`, or `MEMBER` |

### Output Fields (JSON)

Returns an object with:
- `group` -- The group email address queried
- `members` -- Array of member objects
- `count` -- Number of members returned

Each member includes:
- `id` -- Member identifier
- `email` -- Member email address
- `role` -- Member role (`OWNER`, `MANAGER`, or `MEMBER`)
- `type` -- Member type (e.g., `USER`, `GROUP`)
- `status` -- Member status (only present if set)

### Examples

```bash
# List all members of a group
gws groups members engineering@example.com

# List up to 200 members
gws groups members engineering@example.com --max 200

# List only group owners
gws groups members engineering@example.com --role OWNER

# List only group managers
gws groups members engineering@example.com --role MANAGER

# List only regular members
gws groups members engineering@example.com --role MEMBER

# Extract just emails and roles
gws groups members engineering@example.com --format json | jq '.members[] | {email, role}'

# Count members by role
gws groups members engineering@example.com --format json | jq '.members | group_by(.role) | map({role: .[0].role, count: length})'
```

### Notes

- Requires Admin SDK API enabled in the Google Cloud project
- Requires Google Workspace admin privileges
- The group email address is a required positional argument
- The `--role` flag accepts: `OWNER`, `MANAGER`, or `MEMBER` (case-sensitive, uppercase)
- Members of type `GROUP` indicate nested group membership

---

## Common Workflows

### Find All Owners Across Groups

```bash
# List groups, then get owners for each
gws groups list --format json | jq -r '.groups[].email' | while read group; do
  echo "=== $group ==="
  gws groups members "$group" --role OWNER --format json | jq -r '.members[].email'
done
```

### Export Group Membership

```bash
# Export all members of a group to JSON
gws groups members engineering@example.com --max 500 --format json > members.json
```

### Check User's Group Memberships

```bash
# List all groups a user belongs to
gws groups list --user-email alice@example.com --format json | jq '.groups[] | {name, email}'
```
