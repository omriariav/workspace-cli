package spacecache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/usercache"
	"google.golang.org/api/chat/v1"
	"google.golang.org/api/people/v1"
)

// SpaceEntry stores cached space metadata and member emails.
type SpaceEntry struct {
	Type        string   `json:"type"`
	DisplayName string   `json:"display_name,omitempty"`
	Members     []string `json:"members"`
	MemberCount int      `json:"member_count"`
}

// CacheData is the on-disk format for the space-members cache.
type CacheData struct {
	LastUpdated time.Time              `json:"last_updated"`
	Spaces      map[string]SpaceEntry  `json:"spaces"`
}

// DefaultPath returns the default cache file location.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gws", "space-members-cache.json")
}

// Load reads the cache from disk. Returns empty CacheData if file doesn't exist.
func Load(path string) (*CacheData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CacheData{Spaces: make(map[string]SpaceEntry)}, nil
		}
		return nil, err
	}

	var cache CacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return &CacheData{Spaces: make(map[string]SpaceEntry)}, nil
	}
	if cache.Spaces == nil {
		cache.Spaces = make(map[string]SpaceEntry)
	}
	return &cache, nil
}

// Save writes the cache atomically to disk.
func Save(path string, cache *CacheData) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// FindByMembers returns spaces where ALL specified emails are members.
func FindByMembers(cache *CacheData, emails []string) map[string]SpaceEntry {
	results := make(map[string]SpaceEntry)
	for name, entry := range cache.Spaces {
		if containsAll(entry.Members, emails) {
			results[name] = entry
		}
	}
	return results
}

// Build iterates spaces, fetches members, resolves emails, and saves the cache.
// spaceTypeFilter can be "GROUP_CHAT", "SPACE", "DIRECT_MESSAGE", or "all".
// progress is called with (current, total) for progress reporting (may be nil).
func Build(ctx context.Context, chatSvc *chat.Service, peopleSvc *people.Service, spaceTypeFilter string, progress func(current, total int)) (*CacheData, error) {
	// List spaces with optional filter
	var spaces []*chat.Space
	var pageToken string
	for {
		call := chatSvc.Spaces.List().PageSize(100).Context(ctx)
		if spaceTypeFilter != "all" {
			call = call.Filter(fmt.Sprintf(`spaceType = "%s"`, spaceTypeFilter))
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list spaces: %w", err)
		}
		spaces = append(spaces, resp.Spaces...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	// Load user cache for email resolution
	uc, _ := usercache.New()

	cache := &CacheData{
		LastUpdated: time.Now(),
		Spaces:      make(map[string]SpaceEntry),
	}

	total := len(spaces)
	for i, space := range spaces {
		if progress != nil {
			progress(i+1, total)
		}

		// Fetch members for this space
		var members []*chat.Membership
		var memberPageToken string
		for {
			mCall := chatSvc.Spaces.Members.List(space.Name).PageSize(100).Context(ctx)
			if memberPageToken != "" {
				mCall = mCall.PageToken(memberPageToken)
			}

			mResp, err := mCall.Do()
			if err != nil {
				break // best-effort: skip space on error
			}
			members = append(members, mResp.Memberships...)
			if mResp.NextPageToken == "" {
				break
			}
			memberPageToken = mResp.NextPageToken
		}

		// Resolve user IDs to emails
		var userIDs []string
		for _, m := range members {
			if m.Member != nil && strings.HasPrefix(m.Member.Name, "users/") {
				userIDs = append(userIDs, m.Member.Name)
			}
		}
		if uc != nil && peopleSvc != nil {
			uc.ResolveMany(peopleSvc, userIDs)
		}

		var emails []string
		for _, id := range userIDs {
			if uc != nil {
				if info, ok := uc.Get(id); ok && info.Email != "" {
					emails = append(emails, info.Email)
					continue
				}
			}
			emails = append(emails, id) // fallback to user ID
		}

		cache.Spaces[space.Name] = SpaceEntry{
			Type:        space.SpaceType,
			DisplayName: space.DisplayName,
			Members:     emails,
			MemberCount: len(emails),
		}
	}

	return cache, nil
}

func containsAll(haystack []string, needles []string) bool {
	set := make(map[string]bool, len(haystack))
	for _, h := range haystack {
		set[strings.ToLower(h)] = true
	}
	for _, n := range needles {
		if !set[strings.ToLower(n)] {
			return false
		}
	}
	return true
}
