package events

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/omriariav/workspace-cli/internal/auth"
)

func TestValidateTopic(t *testing.T) {
	valid := []string{
		"projects/my-project/topics/chat-events",
		"projects/p1/topics/t_1",
	}
	for _, topic := range valid {
		if err := ValidateTopic(topic); err != nil {
			t.Errorf("ValidateTopic(%q) = %v, want nil", topic, err)
		}
	}

	invalid := map[string]string{
		"":                           "missing Pub/Sub topic",
		"my-topic":                   "invalid Pub/Sub topic",
		"projects/p/subscriptions/s": "invalid Pub/Sub topic",
		"projects//topics/t":         "invalid Pub/Sub topic",
		"projects/p/topics/":         "invalid Pub/Sub topic",
		"projects/p/topics/t/extra":  "invalid Pub/Sub topic",
		"projects/p /topics/t":       "invalid Pub/Sub topic",
	}
	for topic, want := range invalid {
		err := ValidateTopic(topic)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ValidateTopic(%q) = %v, want error containing %q", topic, err, want)
		}
	}
}

func TestClassifyTarget(t *testing.T) {
	valid := map[string]string{
		"//chat.googleapis.com/spaces/AAA":           TargetChatSpace,
		"//chat.googleapis.com/spaces/-":             TargetChatAllSpaces,
		"//cloudidentity.googleapis.com/users/12345": TargetUser,
		"//drive.googleapis.com/files/FILE_ID":       TargetDriveFile,
		"//drive.googleapis.com/drives/DRIVE_ID":     TargetSharedDrive,
		"//meet.googleapis.com/spaces/MEET_SPACE":    TargetMeetSpace,
	}
	for target, want := range valid {
		kind, err := ClassifyTarget(target)
		if err != nil {
			t.Errorf("ClassifyTarget(%q) error: %v", target, err)
			continue
		}
		if kind != want {
			t.Errorf("ClassifyTarget(%q) = %q, want %q", target, kind, want)
		}
	}

	invalid := map[string]string{
		"":                                      "missing subscription target",
		"//admin.googleapis.com/customers/C123": "Developer Preview",
		"spaces/AAA":                            "unsupported subscription target",
		"//example.com/things/1":                "unsupported subscription target",
		// prefix-only targets with empty resource IDs
		"//chat.googleapis.com/spaces/":         "missing resource ID",
		"//cloudidentity.googleapis.com/users/": "missing resource ID",
		"//drive.googleapis.com/files/":         "missing resource ID",
		"//drive.googleapis.com/drives/":        "missing resource ID",
		"//meet.googleapis.com/spaces/":         "missing resource ID",
	}
	for target, want := range invalid {
		_, err := ClassifyTarget(target)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ClassifyTarget(%q) = %v, want error containing %q", target, err, want)
		}
	}
}

func TestPlanSubscription_UserAuthScopes(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		eventTypes []string
		wantScopes []string
	}{
		{
			"chat space messages",
			"//chat.googleapis.com/spaces/AAA",
			[]string{"google.workspace.chat.message.v1.created"},
			[]string{"https://www.googleapis.com/auth/chat.messages.readonly"},
		},
		{
			"chat space mixed classes",
			"//chat.googleapis.com/spaces/AAA",
			[]string{
				"google.workspace.chat.message.v1.created",
				"google.workspace.chat.reaction.v1.created",
				"google.workspace.chat.membership.v1.updated",
				"google.workspace.chat.space.v1.updated",
			},
			[]string{
				"https://www.googleapis.com/auth/chat.memberships.readonly",
				"https://www.googleapis.com/auth/chat.messages.reactions.readonly",
				"https://www.googleapis.com/auth/chat.messages.readonly",
				"https://www.googleapis.com/auth/chat.spaces.readonly",
			},
		},
		{
			"all chat spaces",
			"//chat.googleapis.com/spaces/-",
			[]string{"google.workspace.chat.message.v1.created"},
			[]string{"https://www.googleapis.com/auth/chat.messages.readonly"},
		},
		{
			"user read-state",
			"//cloudidentity.googleapis.com/users/123",
			[]string{
				"google.workspace.chat.spaceReadState.v1.updated",
				"google.workspace.chat.threadReadState.v1.updated",
			},
			[]string{"https://www.googleapis.com/auth/chat.users.readstate.readonly"},
		},
		{
			// Locks the exact Drive mapping: drive.metadata, NOT
			// drive.metadata.readonly (unsupported for subscriptions).
			"drive file and permission events use drive.metadata",
			"//drive.googleapis.com/files/F1",
			[]string{
				"google.workspace.drive.file.v3.contentChanged",
				"google.workspace.drive.permission.v3.created",
			},
			[]string{"https://www.googleapis.com/auth/drive.metadata"},
		},
		{
			"shared drive",
			"//drive.googleapis.com/drives/D1",
			[]string{"google.workspace.drive.file.v3.added"},
			[]string{"https://www.googleapis.com/auth/drive.metadata"},
		},
		{
			"meet space",
			"//meet.googleapis.com/spaces/M1",
			[]string{
				"google.workspace.meet.conference.v2.started",
				"google.workspace.meet.recording.v2.fileGenerated",
				"google.workspace.meet.transcript.v2.fileGenerated",
				"google.workspace.meet.smartNote.v2.fileGenerated",
				"google.workspace.meet.participant.v2.joined",
			},
			[]string{"https://www.googleapis.com/auth/meetings.space.readonly"},
		},
		{
			"meet events on user target",
			"//cloudidentity.googleapis.com/users/123",
			[]string{"google.workspace.meet.conference.v2.started"},
			[]string{"https://www.googleapis.com/auth/meetings.space.readonly"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := PlanSubscription(tt.target, tt.eventTypes, auth.AuthModeOAuth, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(plan.Scopes, tt.wantScopes) {
				t.Errorf("scopes = %v, want %v", plan.Scopes, tt.wantScopes)
			}
		})
	}
}

func TestPlanSubscription_MeetCreatedScope(t *testing.T) {
	plan, err := PlanSubscription("//meet.googleapis.com/spaces/M1",
		[]string{"google.workspace.meet.conference.v2.started"}, auth.AuthModeDWD, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"https://www.googleapis.com/auth/meetings.space.created"}
	if !reflect.DeepEqual(plan.Scopes, want) {
		t.Errorf("scopes = %v, want %v", plan.Scopes, want)
	}
}

func TestPlanSubscription_AppAuth(t *testing.T) {
	t.Run("individual space allowed with granular scopes", func(t *testing.T) {
		plan, err := PlanSubscription("//chat.googleapis.com/spaces/AAA",
			[]string{
				"google.workspace.chat.message.v1.created",
				"google.workspace.chat.reaction.v1.created",
				"google.workspace.chat.membership.v1.created",
				"google.workspace.chat.space.v1.updated",
			}, auth.AuthModeApp, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{
			"https://www.googleapis.com/auth/chat.app.memberships.readonly",
			"https://www.googleapis.com/auth/chat.app.messages.readonly",
			"https://www.googleapis.com/auth/chat.app.spaces.readonly",
		}
		if !reflect.DeepEqual(plan.Scopes, want) {
			t.Errorf("scopes = %v, want %v", plan.Scopes, want)
		}
	})

	t.Run("all-spaces target rejected", func(t *testing.T) {
		_, err := PlanSubscription("//chat.googleapis.com/spaces/-",
			[]string{"google.workspace.chat.message.v1.created"}, auth.AuthModeApp, false)
		if err == nil || !strings.Contains(err.Error(), "individual Chat space") {
			t.Errorf("expected app-mode target rejection, got %v", err)
		}
	})

	t.Run("user target rejected", func(t *testing.T) {
		_, err := PlanSubscription("//cloudidentity.googleapis.com/users/1",
			[]string{"google.workspace.chat.spaceReadState.v1.updated"}, auth.AuthModeApp, false)
		if err == nil || !strings.Contains(err.Error(), "individual Chat space") {
			t.Errorf("expected app-mode target rejection, got %v", err)
		}
	})

	t.Run("drive target rejected", func(t *testing.T) {
		_, err := PlanSubscription("//drive.googleapis.com/files/F1",
			[]string{"google.workspace.drive.file.v3.added"}, auth.AuthModeApp, false)
		if err == nil || !strings.Contains(err.Error(), "individual Chat space") {
			t.Errorf("expected app-mode target rejection, got %v", err)
		}
	})
}

func TestPlanSubscription_GABoundary(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		eventTypes []string
		mode       string
		want       string
	}{
		{"empty event types", "//chat.googleapis.com/spaces/AAA", nil, auth.AuthModeOAuth, "missing event types"},
		{"drive comment preview", "//drive.googleapis.com/files/F1",
			[]string{"google.workspace.drive.comment.v3.created"}, auth.AuthModeOAuth, "Developer Preview"},
		{"drive reply preview", "//drive.googleapis.com/files/F1",
			[]string{"google.workspace.drive.reply.v3.created"}, auth.AuthModeOAuth, "Developer Preview"},
		{"drive approval preview", "//drive.googleapis.com/files/F1",
			[]string{"google.workspace.drive.approval.v3.created"}, auth.AuthModeOAuth, "Developer Preview"},
		{"drive access proposal preview", "//drive.googleapis.com/files/F1",
			[]string{"google.workspace.drive.accessproposal.v3.created"}, auth.AuthModeOAuth, "Developer Preview"},
		{"chat availability excluded", "//cloudidentity.googleapis.com/users/1",
			[]string{"google.workspace.chat.availability.v1.updated"}, auth.AuthModeOAuth, "not in the supported GA set"},
		{"unknown event type", "//chat.googleapis.com/spaces/AAA",
			[]string{"google.workspace.chat.message.v9.exploded"}, auth.AuthModeOAuth, "unknown event type"},
		{"batch events are output-only", "//chat.googleapis.com/spaces/AAA",
			[]string{"google.workspace.chat.message.v1.batchCreated"}, auth.AuthModeOAuth, "unknown event type"},
		{"event class invalid for target", "//drive.googleapis.com/files/F1",
			[]string{"google.workspace.chat.message.v1.created"}, auth.AuthModeOAuth, "not valid for target"},
		{"read-state invalid on space target", "//chat.googleapis.com/spaces/AAA",
			[]string{"google.workspace.chat.spaceReadState.v1.updated"}, auth.AuthModeOAuth, "not valid for target"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PlanSubscription(tt.target, tt.eventTypes, tt.mode, false)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestReadLifecycleScopes(t *testing.T) {
	t.Run("app mode uses chat.bot for read ops", func(t *testing.T) {
		got := ReadLifecycleScopes(auth.AuthModeApp)
		want := []string{"https://www.googleapis.com/auth/chat.bot"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ReadLifecycleScopes(app) = %v, want %v", got, want)
		}
	})

	t.Run("user modes use union of user scopes", func(t *testing.T) {
		for _, mode := range []string{auth.AuthModeOAuth, auth.AuthModeDWD} {
			got := ReadLifecycleScopes(mode)
			want := []string{
				"https://www.googleapis.com/auth/chat.memberships.readonly",
				"https://www.googleapis.com/auth/chat.messages.reactions.readonly",
				"https://www.googleapis.com/auth/chat.messages.readonly",
				"https://www.googleapis.com/auth/chat.spaces.readonly",
				"https://www.googleapis.com/auth/chat.users.readstate.readonly",
				"https://www.googleapis.com/auth/drive.metadata",
				"https://www.googleapis.com/auth/meetings.space.readonly",
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("ReadLifecycleScopes(%s) = %v, want %v", mode, got, want)
			}
		}
	})

	t.Run("app mutations never get a scope union from read lifecycle", func(t *testing.T) {
		// Mutations must derive scopes via PlanSubscription; the read set
		// for app mode is exactly chat.bot and contains no chat.app scope.
		for _, s := range ReadLifecycleScopes(auth.AuthModeApp) {
			if strings.Contains(s, "chat.app") {
				t.Errorf("read lifecycle scopes must not include chat.app scopes, got %v", s)
			}
		}
	})
}

func TestDecodeCloudEvent_PushEnvelope(t *testing.T) {
	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"name": "spaces/AAA/messages/BBB",
		},
	}
	data, _ := json.Marshal(payload)

	envelope := map[string]interface{}{
		"message": map[string]interface{}{
			"attributes": map[string]string{
				"ce-type":    "google.workspace.chat.message.v1.created",
				"ce-source":  "//workspaceevents.googleapis.com/subscriptions/sub-123",
				"ce-subject": "//chat.googleapis.com/spaces/AAA",
				"ce-time":    "2026-08-04T10:00:00Z",
			},
			"data":        base64.StdEncoding.EncodeToString(data),
			"messageId":   "msg-1",
			"publishTime": "2026-08-04T10:00:01Z",
		},
		"subscription": "projects/p/subscriptions/pubsub-sub",
	}
	raw, _ := json.Marshal(envelope)

	ev, err := DecodeCloudEvent(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ev.EventType != "google.workspace.chat.message.v1.created" {
		t.Errorf("EventType = %q", ev.EventType)
	}
	// ce-source is the Workspace Events subscription resource.
	if ev.Subscription != "//workspaceevents.googleapis.com/subscriptions/sub-123" {
		t.Errorf("Subscription = %q, want ce-source value", ev.Subscription)
	}
	// ce-subject is the changed target resource, not the subscription.
	if ev.Target != "//chat.googleapis.com/spaces/AAA" {
		t.Errorf("Target = %q, want ce-subject value", ev.Target)
	}
	if ev.PubsubSubscription != "projects/p/subscriptions/pubsub-sub" {
		t.Errorf("PubsubSubscription = %q", ev.PubsubSubscription)
	}
	if ev.MessageID != "msg-1" {
		t.Errorf("MessageID = %q", ev.MessageID)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(ev.Data, &decoded); err != nil {
		t.Fatalf("data is not JSON: %v", err)
	}
	msg, _ := decoded["message"].(map[string]interface{})
	if msg["name"] != "spaces/AAA/messages/BBB" {
		t.Errorf("decoded data = %v", decoded)
	}
}

func TestDecodeCloudEvent_BareMessageAndBatchType(t *testing.T) {
	// Batch events are output-only: they can never be subscribed to, but
	// decode must still accept their CloudEvent attributes.
	bare := map[string]interface{}{
		"attributes": map[string]string{
			"ce-type":   "google.workspace.chat.message.v1.batchCreated",
			"ce-source": "//workspaceevents.googleapis.com/subscriptions/sub-9",
		},
		"messageId": "msg-2",
	}
	raw, _ := json.Marshal(bare)

	ev, err := DecodeCloudEvent(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.EventType != "google.workspace.chat.message.v1.batchCreated" {
		t.Errorf("EventType = %q", ev.EventType)
	}
	if ev.Subscription != "//workspaceevents.googleapis.com/subscriptions/sub-9" {
		t.Errorf("Subscription = %q", ev.Subscription)
	}
	if ev.Data != nil {
		t.Errorf("Data = %s, want empty", ev.Data)
	}
}

func TestDecodeCloudEvent_NonJSONData(t *testing.T) {
	bare := map[string]interface{}{
		"data": base64.StdEncoding.EncodeToString([]byte("plain text")),
	}
	raw, _ := json.Marshal(bare)

	ev, err := DecodeCloudEvent(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var s string
	if err := json.Unmarshal(ev.Data, &s); err != nil || s != "plain text" {
		t.Errorf("Data = %s, want JSON string \"plain text\"", ev.Data)
	}
}

func TestDecodeCloudEvent_Errors(t *testing.T) {
	if _, err := DecodeCloudEvent(strings.NewReader("")); err == nil || !strings.Contains(err.Error(), "no event on stdin") {
		t.Errorf("empty input: %v", err)
	}
	if _, err := DecodeCloudEvent(strings.NewReader("{not json")); err == nil || !strings.Contains(err.Error(), "invalid Pub/Sub message JSON") {
		t.Errorf("bad json: %v", err)
	}
	if _, err := DecodeCloudEvent(strings.NewReader(`{"data":"!!!not-base64!!!"}`)); err == nil || !strings.Contains(err.Error(), "base64") {
		t.Errorf("bad base64: %v", err)
	}
}

func TestSupportedEventTypes(t *testing.T) {
	types := SupportedEventTypes()
	if len(types) != len(gaEventClasses) {
		t.Fatalf("len = %d, want %d", len(types), len(gaEventClasses))
	}
	for _, et := range types {
		if strings.Contains(et, "batch") {
			t.Errorf("batch event %q must not be creatable", et)
		}
	}
}
