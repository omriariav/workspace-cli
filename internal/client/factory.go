package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/omriariav/workspace-cli/internal/auth"
	"github.com/omriariav/workspace-cli/internal/config"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/chat/v1"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/forms/v1"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"
	"google.golang.org/api/tasks/v1"
)

// Factory provides lazy-initialized Google API service clients.
type Factory struct {
	mu          sync.Mutex
	ctx         context.Context
	tokenSource oauth2.TokenSource

	gmail    *gmail.Service
	calendar *calendar.Service
	drive    *drive.Service
	docs     *docs.Service
	sheets   *sheets.Service
	slides   *slides.Service
	tasks    *tasks.Service
	chat     *chat.Service
	forms    *forms.Service
}

// NewFactory creates a new client factory.
func NewFactory(ctx context.Context) (*Factory, error) {
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
		auth.SaveToken(newToken)
	}

	return &Factory{
		ctx:         ctx,
		tokenSource: ts,
	}, nil
}

// Gmail returns the Gmail service client.
func (f *Factory) Gmail() (*gmail.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.gmail != nil {
		return f.gmail, nil
	}

	svc, err := gmail.NewService(f.ctx, option.WithTokenSource(f.tokenSource))
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

	if f.calendar != nil {
		return f.calendar, nil
	}

	svc, err := calendar.NewService(f.ctx, option.WithTokenSource(f.tokenSource))
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

	if f.drive != nil {
		return f.drive, nil
	}

	svc, err := drive.NewService(f.ctx, option.WithTokenSource(f.tokenSource))
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

	if f.docs != nil {
		return f.docs, nil
	}

	svc, err := docs.NewService(f.ctx, option.WithTokenSource(f.tokenSource))
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

	if f.sheets != nil {
		return f.sheets, nil
	}

	svc, err := sheets.NewService(f.ctx, option.WithTokenSource(f.tokenSource))
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

	if f.slides != nil {
		return f.slides, nil
	}

	svc, err := slides.NewService(f.ctx, option.WithTokenSource(f.tokenSource))
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

	if f.tasks != nil {
		return f.tasks, nil
	}

	svc, err := tasks.NewService(f.ctx, option.WithTokenSource(f.tokenSource))
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

	if f.chat != nil {
		return f.chat, nil
	}

	svc, err := chat.NewService(f.ctx, option.WithTokenSource(f.tokenSource))
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

	if f.forms != nil {
		return f.forms, nil
	}

	svc, err := forms.NewService(f.ctx, option.WithTokenSource(f.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Forms client: %w", err)
	}

	f.forms = svc
	return svc, nil
}
