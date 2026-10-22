package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/omriariav/workspace-cli/internal/config"
	"golang.org/x/oauth2"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/chat/v1"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	driveactivity "google.golang.org/api/driveactivity/v2"
	"google.golang.org/api/forms/v1"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/keep/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"
	"google.golang.org/api/tasks/v1"
	workspaceevents "google.golang.org/api/workspaceevents/v1"
)

var (
	rateLimitMu       sync.Mutex
	rateLimitInterval time.Duration // 0 means unlimited
	rateLimitNext     time.Time
)

// SetRateLimit caps outgoing API requests at perSecond requests per second
// across all service clients created afterwards. Zero or negative disables
// the limit. The limit is client-side only and does not inspect responses.
func SetRateLimit(perSecond float64) {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()
	if perSecond <= 0 {
		rateLimitInterval = 0
		return
	}
	rateLimitInterval = time.Duration(float64(time.Second) / perSecond)
	rateLimitNext = time.Time{}
}

// rateLimitWait reserves the next request slot and returns how long the
// caller must wait before sending.
func rateLimitWait() time.Duration {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()
	if rateLimitInterval <= 0 {
		return 0
	}
	now := time.Now()
	slot := rateLimitNext
	if slot.Before(now) {
		slot = now
	}
	rateLimitNext = slot.Add(rateLimitInterval)
	return slot.Sub(now)
}

type rateLimitedTransport struct {
	base http.RoundTripper
}

func (t *rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if d := rateLimitWait(); d > 0 {
		timer := time.NewTimer(d)
		select {
		case <-timer.C:
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		}
	}
	return t.base.RoundTrip(req)
}

// authOption authenticates a service client with ts. When a rate limit is
// set, requests go through an HTTP client that throttles them first.
func authOption(ctx context.Context, ts oauth2.TokenSource) option.ClientOption {
	rateLimitMu.Lock()
	limited := rateLimitInterval > 0
	rateLimitMu.Unlock()
	if !limited {
		return option.WithTokenSource(ts)
	}
	return option.WithHTTPClient(&http.Client{
		Transport: &oauth2.Transport{
			Source: ts,
			Base:   &rateLimitedTransport{base: http.DefaultTransport},
		},
	})
}

// Factory provides lazy-initialized Google API service clients.
type Factory struct {
	mu              sync.Mutex
	ctx             context.Context
	authMode        *auth.AuthModeConfig
	tokenSource     oauth2.TokenSource
	grantedServices []string        // services granted at login time
	scopeWarned     map[string]bool // track warnings to avoid repeats

	gmail         *gmail.Service
	calendar      *calendar.Service
	drive         *drive.Service
	docs          *docs.Service
	sheets        *sheets.Service
	slides        *slides.Service
	tasks         *tasks.Service
	chat          *chat.Service
	forms         *forms.Service
	people        *people.Service
	admin         *admin.Service
	keep          *keep.Service
	driveActivity *driveactivity.Service
}

// NewFactory creates a new client factory. The token source depends on the
// configured auth mode: the default OAuth flow reuses the stored user token,
// while the service-account modes mint credentials from the configured JSON
// key (with the DWD mode impersonating the configured Workspace user).
func NewFactory(ctx context.Context) (*Factory, error) {
	modeCfg, err := auth.ResolveAuthMode(
		config.GetAuthMode(),
		config.GetServiceAccountKeyFile(),
		config.GetImpersonateUser(),
	)
	if err != nil {
		return nil, err
	}

	if modeCfg.Mode == auth.AuthModeApp {
		// App mode never pre-mints credentials: only the Workspace Events
		// surface may request a token, scoped exactly to the granular
		// chat.app scopes its subscription class needs. Every other
		// service getter is rejected via requireUserAuth.
		return &Factory{
			ctx:         ctx,
			authMode:    modeCfg,
			scopeWarned: make(map[string]bool),
		}, nil
	}

	if modeCfg.Mode == auth.AuthModeDWD {
		ts, err := auth.ServiceAccountTokenSource(ctx, modeCfg, serviceAccountScopes(modeCfg))
		if err != nil {
			return nil, err
		}
		return &Factory{
			ctx:         ctx,
			authMode:    modeCfg,
			tokenSource: ts,
			scopeWarned: make(map[string]bool),
		}, nil
	}

	token, err := auth.LoadToken()
	if err != nil {
		return nil, err
	}

	clientID := config.GetClientID()
	clientSecret := config.GetClientSecret()

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("missing OAuth credentials")
	}

	ts := auth.GetTokenSource(ctx, clientID, clientSecret, token)

	// Check if token is valid by trying to get a token
	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("token expired, run: gws auth login")
	}

	// Save refreshed token if it changed
	if newToken.AccessToken != token.AccessToken {
		_ = auth.SaveToken(newToken)
	}

	return &Factory{
		ctx:             ctx,
		authMode:        modeCfg,
		tokenSource:     ts,
		grantedServices: auth.LoadGrantedServices(),
		scopeWarned:     make(map[string]bool),
	}, nil
}

// serviceAccountScopes returns the scopes a DWD token source is minted
// with. DWD acts as a user, so it follows the same service-based scope
// resolution as an OAuth login (config `services` list, or all services).
func serviceAccountScopes(modeCfg *auth.AuthModeConfig) []string {
	if services := config.GetServices(); len(services) > 0 {
		return auth.ScopesForServices(services)
	}
	return auth.AllScopes
}

// requireUserAuth rejects service access under service-account-app mode.
// App credentials are limited to Workspace Events subscriptions on
// individual Chat spaces; every regular service client needs user
// credentials (OAuth or DWD impersonation).
func (f *Factory) requireUserAuth(service string) error {
	if f.authMode != nil && f.authMode.Mode == auth.AuthModeApp {
		return fmt.Errorf("%s is not available with auth_mode %q: Chat app credentials only support `gws events` subscriptions on individual Chat spaces; use auth_mode %q or the default OAuth login for %s",
			service, auth.AuthModeApp, auth.AuthModeDWD, service)
	}
	return nil
}

// AuthMode returns the resolved authentication mode config for this factory.
func (f *Factory) AuthMode() *auth.AuthModeConfig {
	return f.authMode
}

// TokenSource exposes the factory's token source so commands can construct
// clients for services not cached on the factory.
func (f *Factory) TokenSource() oauth2.TokenSource {
	return f.tokenSource
}

// serviceAliases maps every accepted alias for a service to its canonical
// name. Aliases share the same OAuth scope set in auth.ServiceScopes — see
// `people` ↔ `contacts` (both surface the People API).
var serviceAliases = map[string]string{
	"contacts": "contacts",
	"people":   "contacts",
}

// checkServiceScopes checks if the required service was granted during login.
// Prints a warning to stderr if the service was not included in the scoped login.
// Treats `people` and `contacts` (and any other entry in serviceAliases) as
// equivalent — a login with one grants the other.
func (f *Factory) checkServiceScopes(service string) {
	if len(f.grantedServices) == 0 {
		return // Full auth or no metadata — skip check
	}

	canonical := canonicalService(service)
	for _, s := range f.grantedServices {
		if canonicalService(s) == canonical {
			return // Granted (directly or via alias)
		}
	}

	// Only warn once per service per session
	if f.scopeWarned[service] {
		return
	}
	f.scopeWarned[service] = true

	allServices := append(f.grantedServices, service)
	fmt.Fprintf(os.Stderr, "%s requires additional permissions. Re-authorize with:\n  gws auth login --services %s\n",
		service, strings.Join(allServices, ","))
}

// canonicalService returns the canonical name for a service, resolving any
// alias (e.g. "people" → "contacts"). Unknown services pass through.
func canonicalService(s string) string {
	if c, ok := serviceAliases[s]; ok {
		return c
	}
	return s
}

// Gmail returns the Gmail service client.
func (f *Factory) Gmail() (*gmail.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("gmail"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("gmail")

	if f.gmail != nil {
		return f.gmail, nil
	}

	svc, err := gmail.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail client: %w", err)
	}

	f.gmail = svc
	return svc, nil
}

// Calendar returns the Calendar service client.
func (f *Factory) Calendar() (*calendar.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("calendar"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("calendar")

	if f.calendar != nil {
		return f.calendar, nil
	}

	svc, err := calendar.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Calendar client: %w", err)
	}

	f.calendar = svc
	return svc, nil
}

// Drive returns the Drive service client.
func (f *Factory) Drive() (*drive.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("drive"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("drive")

	if f.drive != nil {
		return f.drive, nil
	}

	svc, err := drive.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive client: %w", err)
	}

	f.drive = svc
	return svc, nil
}

// Docs returns the Docs service client.
func (f *Factory) Docs() (*docs.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("docs"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("docs")

	if f.docs != nil {
		return f.docs, nil
	}

	svc, err := docs.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Docs client: %w", err)
	}

	f.docs = svc
	return svc, nil
}

// Sheets returns the Sheets service client.
func (f *Factory) Sheets() (*sheets.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("sheets"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("sheets")

	if f.sheets != nil {
		return f.sheets, nil
	}

	svc, err := sheets.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Sheets client: %w", err)
	}

	f.sheets = svc
	return svc, nil
}

// Slides returns the Slides service client.
func (f *Factory) Slides() (*slides.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("slides"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("slides")

	if f.slides != nil {
		return f.slides, nil
	}

	svc, err := slides.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Slides client: %w", err)
	}

	f.slides = svc
	return svc, nil
}

// Tasks returns the Tasks service client.
func (f *Factory) Tasks() (*tasks.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("tasks"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("tasks")

	if f.tasks != nil {
		return f.tasks, nil
	}

	svc, err := tasks.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Tasks client: %w", err)
	}

	f.tasks = svc
	return svc, nil
}

// Chat returns the Chat service client.
func (f *Factory) Chat() (*chat.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("chat"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("chat")

	if f.chat != nil {
		return f.chat, nil
	}

	svc, err := chat.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Chat client: %w", err)
	}

	f.chat = svc
	return svc, nil
}

// Forms returns the Forms service client.
func (f *Factory) Forms() (*forms.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("forms"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("forms")

	if f.forms != nil {
		return f.forms, nil
	}

	svc, err := forms.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Forms client: %w", err)
	}

	f.forms = svc
	return svc, nil
}

// Admin returns the Admin Directory API service client.
func (f *Factory) Admin() (*admin.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("groups"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("groups")

	if f.admin != nil {
		return f.admin, nil
	}

	svc, err := admin.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Admin Directory client: %w", err)
	}

	f.admin = svc
	return svc, nil
}

// Keep returns the Keep API service client.
func (f *Factory) Keep() (*keep.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("keep"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("keep")

	if f.keep != nil {
		return f.keep, nil
	}

	svc, err := keep.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Keep client: %w", err)
	}

	f.keep = svc
	return svc, nil
}

// DriveActivity returns the Drive Activity API v2 service client.
func (f *Factory) DriveActivity() (*driveactivity.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("driveactivity"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("driveactivity")

	if f.driveActivity != nil {
		return f.driveActivity, nil
	}

	svc, err := driveactivity.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive Activity client: %w", err)
	}

	f.driveActivity = svc
	return svc, nil
}

// People returns the People API service client. Used by Contacts commands
// and any path that needs the directory/contacts scopes.
func (f *Factory) People() (*people.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.requireUserAuth("contacts"); err != nil {
		return nil, err
	}
	f.checkServiceScopes("contacts")

	return f.peopleLocked()
}

// PeopleProfile returns the People API service client for profile-only paths
// (e.g. self-identity lookups via people/me) that work with the userinfo
// scope and do not require the contacts scope. This avoids printing a
// misleading "requires contacts" warning to users who authenticated with a
// scoped login that omitted contacts.
func (f *Factory) PeopleProfile() (*people.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.requireUserAuth("people"); err != nil {
		return nil, err
	}
	return f.peopleLocked()
}

// peopleLocked builds (or returns the cached) People service. Caller must
// hold f.mu.
func (f *Factory) peopleLocked() (*people.Service, error) {
	if f.people != nil {
		return f.people, nil
	}
	svc, err := people.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create People client: %w", err)
	}
	f.people = svc
	return svc, nil
}

// WorkspaceEvents returns a Workspace Events API client. In the
// service-account modes a dedicated token source is minted with exactly the
// provided least-privilege scopes (computed by the events command from the
// subscription class); the default OAuth mode reuses the stored user token,
// whose scopes were granted at login. Not cached: scopes vary per call.
func (f *Factory) WorkspaceEvents(scopes []string) (*workspaceevents.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.authMode != nil && f.authMode.Mode != auth.AuthModeOAuth {
		ts, err := auth.ServiceAccountTokenSource(f.ctx, f.authMode, scopes)
		if err != nil {
			return nil, err
		}
		svc, err := workspaceevents.NewService(f.ctx, authOption(f.ctx, ts))
		if err != nil {
			return nil, fmt.Errorf("failed to create Workspace Events client: %w", err)
		}
		return svc, nil
	}

	f.checkServiceScopes("events")

	svc, err := workspaceevents.NewService(f.ctx, authOption(f.ctx, f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Workspace Events client: %w", err)
	}
	return svc, nil
}
