package cmd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
	workspaceevents "google.golang.org/api/workspaceevents/v1"
)

func TestEventsCommands(t *testing.T) {
	var eventsCommand *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "events" {
			eventsCommand = cmd
			break
		}
	}
	if eventsCommand == nil {
		t.Fatal("events command not found")
	}

	expectedSubs := []string{"subscriptions", "decode"}
	for _, name := range expectedSubs {
		found := false
		for _, cmd := range eventsCommand.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected events subcommand '%s' not found", name)
		}
	}

	var subsCommand *cobra.Command
	for _, cmd := range eventsCommand.Commands() {
		if cmd.Name() == "subscriptions" {
			subsCommand = cmd
			break
		}
	}
	if subsCommand == nil {
		t.Fatal("events subscriptions command not found")
	}

	expectedLifecycle := []string{"list", "get", "create", "update", "renew", "delete", "reactivate"}
	for _, name := range expectedLifecycle {
		found := false
		for _, cmd := range subsCommand.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected events subscriptions subcommand '%s' not found", name)
		}
	}
}

func TestSubscriptionName(t *testing.T) {
	if got := subscriptionName("abc123"); got != "subscriptions/abc123" {
		t.Errorf("subscriptionName(abc123) = %q", got)
	}
	if got := subscriptionName("subscriptions/abc123"); got != "subscriptions/abc123" {
		t.Errorf("subscriptionName(subscriptions/abc123) = %q", got)
	}
}

func TestSplitEventTypes(t *testing.T) {
	got := splitEventTypes(" a ,b,, c ")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildListFilter(t *testing.T) {
	t.Run("event type only", func(t *testing.T) {
		f, err := buildListFilter("", "", "google.workspace.chat.message.v1.created")
		if err != nil {
			t.Fatal(err)
		}
		if f != `event_types:"google.workspace.chat.message.v1.created"` {
			t.Errorf("filter = %s", f)
		}
	})

	t.Run("event type and target", func(t *testing.T) {
		f, err := buildListFilter("", "//chat.googleapis.com/spaces/AAA", "google.workspace.chat.message.v1.created")
		if err != nil {
			t.Fatal(err)
		}
		want := `event_types:"google.workspace.chat.message.v1.created" AND target_resource="//chat.googleapis.com/spaces/AAA"`
		if f != want {
			t.Errorf("filter = %s, want %s", f, want)
		}
	})

	t.Run("target only rejected: API requires event_types", func(t *testing.T) {
		_, err := buildListFilter("", "//chat.googleapis.com/spaces/AAA", "")
		if err == nil || !strings.Contains(err.Error(), "event type") {
			t.Errorf("expected event-type requirement error, got %v", err)
		}
	})

	t.Run("no flags rejected", func(t *testing.T) {
		_, err := buildListFilter("", "", "")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("raw filter with event_types accepted", func(t *testing.T) {
		raw := `event_types:"google.workspace.chat.message.v1.created"`
		f, err := buildListFilter(raw, "", "")
		if err != nil || f != raw {
			t.Errorf("f = %q, err = %v", f, err)
		}
	})

	t.Run("raw filter without event_types rejected", func(t *testing.T) {
		_, err := buildListFilter(`target_resource="//chat.googleapis.com/spaces/AAA"`, "", "")
		if err == nil || !strings.Contains(err.Error(), "event_types") {
			t.Errorf("expected event_types requirement error, got %v", err)
		}
	})
}

// mockEventsServer routes requests by "METHOD PATH" and fails the test on
// unexpected requests.
func mockEventsServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if h, ok := handlers[key]; ok {
			h(w, r)
			return
		}
		t.Errorf("unexpected request: %s", key)
		http.NotFound(w, r)
	}))
}

func newTestEventsService(t *testing.T, url string) *workspaceevents.Service {
	t.Helper()
	svc, err := workspaceevents.NewService(context.Background(),
		option.WithoutAuthentication(), option.WithEndpoint(url))
	if err != nil {
		t.Fatalf("failed to create workspaceevents service: %v", err)
	}
	return svc
}

func TestEventsSubscriptionCreate_RequestShape(t *testing.T) {
	var got workspaceevents.Subscription
	server := mockEventsServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"POST /v1/subscriptions": func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatalf("bad body: %v", err)
			}
			json.NewEncoder(w).Encode(&workspaceevents.Operation{Name: "operations/op1"})
		},
	})
	defer server.Close()

	svc := newTestEventsService(t, server.URL)
	sub := &workspaceevents.Subscription{
		TargetResource: "//chat.googleapis.com/spaces/AAA",
		EventTypes:     []string{"google.workspace.chat.message.v1.created"},
		NotificationEndpoint: &workspaceevents.NotificationEndpoint{
			PubsubTopic: "projects/p/topics/t",
		},
		PayloadOptions: &workspaceevents.PayloadOptions{IncludeResource: true},
		Ttl:            "86400s",
	}
	op, err := svc.Subscriptions.Create(sub).Do()
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if op.Name != "operations/op1" {
		t.Errorf("op = %+v", op)
	}

	if got.TargetResource != "//chat.googleapis.com/spaces/AAA" {
		t.Errorf("targetResource = %q", got.TargetResource)
	}
	if len(got.EventTypes) != 1 || got.EventTypes[0] != "google.workspace.chat.message.v1.created" {
		t.Errorf("eventTypes = %v", got.EventTypes)
	}
	if got.NotificationEndpoint == nil || got.NotificationEndpoint.PubsubTopic != "projects/p/topics/t" {
		t.Errorf("notificationEndpoint = %+v", got.NotificationEndpoint)
	}
	if got.PayloadOptions == nil || !got.PayloadOptions.IncludeResource {
		t.Errorf("payloadOptions = %+v", got.PayloadOptions)
	}
	if got.Ttl != "86400s" {
		t.Errorf("ttl = %q", got.Ttl)
	}
}

func TestEventsSubscriptionList_FilterAndPaging(t *testing.T) {
	var gotFilter string
	page := 0
	server := mockEventsServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"GET /v1/subscriptions": func(w http.ResponseWriter, r *http.Request) {
			gotFilter = r.URL.Query().Get("filter")
			page++
			resp := &workspaceevents.ListSubscriptionsResponse{
				Subscriptions: []*workspaceevents.Subscription{
					{Name: "subscriptions/sub-" + r.URL.Query().Get("pageToken")},
				},
			}
			if page == 1 {
				resp.NextPageToken = "next"
			}
			json.NewEncoder(w).Encode(resp)
		},
	})
	defer server.Close()

	svc := newTestEventsService(t, server.URL)
	filter := `event_types:"google.workspace.chat.message.v1.created"`

	resp, err := svc.Subscriptions.List().Filter(filter).PageSize(10).Do()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if gotFilter != filter {
		t.Errorf("filter = %q, want %q", gotFilter, filter)
	}
	if resp.NextPageToken != "next" {
		t.Errorf("nextPageToken = %q", resp.NextPageToken)
	}
}

func TestEventsSubscriptionPatch_RenewTTLAndMask(t *testing.T) {
	var got workspaceevents.Subscription
	var gotMask string
	server := mockEventsServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"PATCH /v1/subscriptions/sub-1": func(w http.ResponseWriter, r *http.Request) {
			gotMask = r.URL.Query().Get("updateMask")
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatalf("bad body: %v", err)
			}
			json.NewEncoder(w).Encode(&workspaceevents.Operation{Name: "operations/op2"})
		},
	})
	defer server.Close()

	svc := newTestEventsService(t, server.URL)
	_, err := svc.Subscriptions.Patch("subscriptions/sub-1", &workspaceevents.Subscription{Ttl: "0s"}).
		UpdateMask("ttl").Do()
	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}
	if gotMask != "ttl" {
		t.Errorf("updateMask = %q", gotMask)
	}
	if got.Ttl != "0s" {
		t.Errorf("ttl = %q, want 0s (maximum expiration)", got.Ttl)
	}
}

func TestEventsSubscriptionDeleteGetReactivate(t *testing.T) {
	server := mockEventsServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"GET /v1/subscriptions/sub-1": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&workspaceevents.Subscription{
				Name:           "subscriptions/sub-1",
				TargetResource: "//chat.googleapis.com/spaces/AAA",
				EventTypes:     []string{"google.workspace.chat.message.v1.created"},
				State:          "SUSPENDED",
			})
		},
		"DELETE /v1/subscriptions/sub-1": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&workspaceevents.Operation{Name: "operations/op3", Done: true})
		},
		"POST /v1/subscriptions/sub-1:reactivate": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&workspaceevents.Operation{Name: "operations/op4"})
		},
	})
	defer server.Close()

	svc := newTestEventsService(t, server.URL)

	sub, err := svc.Subscriptions.Get("subscriptions/sub-1").Do()
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if sub.State != "SUSPENDED" || sub.TargetResource != "//chat.googleapis.com/spaces/AAA" {
		t.Errorf("sub = %+v", sub)
	}

	if _, err := svc.Subscriptions.Delete("subscriptions/sub-1").Do(); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := svc.Subscriptions.Reactivate("subscriptions/sub-1", &workspaceevents.ReactivateSubscriptionRequest{}).Do(); err != nil {
		t.Fatalf("reactivate failed: %v", err)
	}
}

// setAuthModeConfig sets the auth-mode viper keys for a test and restores
// clean config afterwards.
func setAuthModeConfig(t *testing.T, mode, keyFile, impersonate string) {
	t.Helper()
	viper.Set("auth_mode", mode)
	viper.Set("service_account_key_file", keyFile)
	viper.Set("impersonate_user", impersonate)
	t.Cleanup(func() {
		viper.Set("auth_mode", "")
		viper.Set("service_account_key_file", "")
		viper.Set("impersonate_user", "")
	})
}

func TestAuthLogin_FailsFastInServiceAccountModes(t *testing.T) {
	for _, mode := range []string{"service-account-dwd", "service-account-app"} {
		t.Run(mode, func(t *testing.T) {
			impersonate := ""
			if mode == "service-account-dwd" {
				impersonate = "user@example.com"
			}
			setAuthModeConfig(t, mode, "/tmp/key.json", impersonate)

			err := runLogin(loginCmd, nil)
			if err == nil {
				t.Fatal("expected fail-fast error, got nil")
			}
			if !strings.Contains(err.Error(), mode) || !strings.Contains(err.Error(), "service-account modes need no login") {
				t.Errorf("error = %q, want service-account guidance mentioning %q", err.Error(), mode)
			}
		})
	}
}

func TestAuthLogin_InvalidModeConfigFailsFast(t *testing.T) {
	// DWD without impersonation is incomplete config: login must surface
	// the validation error instead of starting OAuth.
	setAuthModeConfig(t, "service-account-dwd", "/tmp/key.json", "")

	err := runLogin(loginCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "impersonate_user") {
		t.Errorf("error = %v, want impersonate_user validation error", err)
	}
}

// writeTestSAKey writes a syntactically valid service-account JSON key
// (fresh test-only RSA key, never a real credential) and returns its path.
func writeTestSAKey(t *testing.T) string {
	t.Helper()

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
	})
	key := map[string]string{
		"type":         "service_account",
		"project_id":   "test-project",
		"private_key":  string(keyPEM),
		"client_email": "gws-test@test-project.iam.gserviceaccount.com",
		"token_uri":    "https://oauth2.googleapis.com/token",
	}
	data, err := json.Marshal(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sa-key.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAuthStatus_ReportsServiceAccountMode(t *testing.T) {
	keyFile := writeTestSAKey(t)
	setAuthModeConfig(t, "service-account-dwd", keyFile, "user@example.com")

	out, err := captureStdout(t, func() error {
		return runStatus(statusCmd, nil)
	})
	if err != nil {
		t.Fatalf("runStatus error: %v", err)
	}

	for _, want := range []string{"service-account-dwd", "user@example.com", keyFile, `"configured": true`, `"key_valid": true`} {
		if !strings.Contains(out, want) {
			t.Errorf("status output missing %q: %s", want, out)
		}
	}
	// Configured is not authenticated: no network validation happened, so
	// the status must not claim authenticated.
	if strings.Contains(out, `"authenticated": true`) {
		t.Errorf("status must not report authenticated:true for a merely configured service-account mode: %s", out)
	}
	if strings.Contains(out, "Not authenticated") {
		t.Errorf("status must not claim unauthenticated in service-account mode: %s", out)
	}
}

func TestAuthStatus_SurfacesInvalidServiceAccountKey(t *testing.T) {
	t.Run("missing key file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing.json")
		setAuthModeConfig(t, "service-account-app", missing, "")

		out, err := captureStdout(t, func() error {
			return runStatus(statusCmd, nil)
		})
		if err != nil {
			t.Fatalf("runStatus error: %v", err)
		}
		for _, want := range []string{`"key_valid": false`, "not found"} {
			if !strings.Contains(out, want) {
				t.Errorf("status output missing %q: %s", want, out)
			}
		}
	})

	t.Run("malformed key file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(path, []byte(`{"type":"service_account"`), 0600); err != nil {
			t.Fatal(err)
		}
		setAuthModeConfig(t, "service-account-app", path, "")

		out, err := captureStdout(t, func() error {
			return runStatus(statusCmd, nil)
		})
		if err != nil {
			t.Fatalf("runStatus error: %v", err)
		}
		for _, want := range []string{`"key_valid": false`, "not a valid service-account JSON key"} {
			if !strings.Contains(out, want) {
				t.Errorf("status output missing %q: %s", want, out)
			}
		}
	})
}

func TestEventsUpdate_EmptyEventTypesRejected(t *testing.T) {
	// --event-types provided but containing no usable entries (e.g. ",,")
	// must fail locally as a usage error, never reach a PATCH with an
	// empty update mask.
	if err := eventsSubsUpdateCmd.Flags().Set("event-types", ",,"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = eventsSubsUpdateCmd.Flags().Set("event-types", "")
		_ = eventsSubsUpdateCmd.Flags().Set("ttl", "")
	})

	err := runEventsSubsUpdate(eventsSubsUpdateCmd, []string{"sub-1"})
	if err == nil {
		t.Fatal("expected usage error, got nil")
	}
	if !strings.Contains(err.Error(), "contains no event types") {
		t.Errorf("error = %q, want empty-event-types usage error", err.Error())
	}
}

func TestEventsList_MaxValidation(t *testing.T) {
	// Non-positive --max must fail locally as a usage error before any
	// filter building or client/HTTP work; a valid max proceeds past the
	// local validation (and then fails on auth in this environment, not
	// on usage).
	setFlags := func(t *testing.T, max string) {
		t.Helper()
		if err := eventsSubsListCmd.Flags().Set("event-type", "google.workspace.chat.message.v1.created"); err != nil {
			t.Fatal(err)
		}
		if err := eventsSubsListCmd.Flags().Set("max", max); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = eventsSubsListCmd.Flags().Set("event-type", "")
			_ = eventsSubsListCmd.Flags().Set("max", "50")
		})
	}

	tests := []struct {
		name    string
		max     string
		wantErr string
	}{
		{"zero", "0", "--max must be greater than 0"},
		{"negative", "-1", "--max must be greater than 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFlags(t, tt.max)
			err := runEventsSubsList(eventsSubsListCmd, nil)
			if err == nil {
				t.Fatal("expected usage error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
			var ue *usageError
			if !errors.As(err, &ue) {
				t.Errorf("error should be a usage error, got %T", err)
			}
		})
	}
}
