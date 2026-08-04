// Package events implements the domain rules for the Workspace Events API:
// GA-only target and event-type validation, least-privilege scope selection
// per subscription class and auth mode, Pub/Sub topic format validation,
// and CloudEvent notification decoding.
package events

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/omriariav/workspace-cli/internal/auth"
)

// Subscription target kinds (GA only — customer-wide Chat targets are
// Developer Preview and rejected).
const (
	TargetChatSpace     = "chat-space"      // //chat.googleapis.com/spaces/{space}
	TargetChatAllSpaces = "chat-all-spaces" // //chat.googleapis.com/spaces/-
	TargetUser          = "user"            // //cloudidentity.googleapis.com/users/{user}
	TargetDriveFile     = "drive-file"      // //drive.googleapis.com/files/{file}
	TargetSharedDrive   = "shared-drive"    // //drive.googleapis.com/drives/{drive}
	TargetMeetSpace     = "meet-space"      // //meet.googleapis.com/spaces/{space}
)

// Event classes group event types that share one least-privilege scope.
const (
	classChatSpace       = "chat-space"
	classChatMembership  = "chat-membership"
	classChatMessage     = "chat-message"
	classChatReaction    = "chat-reaction"
	classChatReadState   = "chat-read-state"
	classDriveFile       = "drive-file"
	classDrivePermission = "drive-permission"
	classMeet            = "meet"
)

// gaEventClasses maps every supported GA event type valid for subscription
// creation to its class. Preview event families (Drive
// comment/reply/approval/access-proposal, Chat availability, customer-wide
// events) are intentionally absent. Chat batch events are also absent:
// they are output-only and delivered automatically once the base event is
// subscribed, so they are decodable but never requestable.
var gaEventClasses = map[string]string{
	"google.workspace.chat.message.v1.created": classChatMessage,
	"google.workspace.chat.message.v1.updated": classChatMessage,
	"google.workspace.chat.message.v1.deleted": classChatMessage,

	"google.workspace.chat.reaction.v1.created": classChatReaction,
	"google.workspace.chat.reaction.v1.deleted": classChatReaction,

	"google.workspace.chat.membership.v1.created": classChatMembership,
	"google.workspace.chat.membership.v1.updated": classChatMembership,
	"google.workspace.chat.membership.v1.deleted": classChatMembership,

	"google.workspace.chat.space.v1.updated": classChatSpace,
	"google.workspace.chat.space.v1.deleted": classChatSpace,

	"google.workspace.chat.spaceReadState.v1.updated":  classChatReadState,
	"google.workspace.chat.threadReadState.v1.updated": classChatReadState,

	"google.workspace.drive.file.v3.added":          classDriveFile,
	"google.workspace.drive.file.v3.moved":          classDriveFile,
	"google.workspace.drive.file.v3.contentChanged": classDriveFile,
	"google.workspace.drive.file.v3.deleted":        classDriveFile,
	"google.workspace.drive.file.v3.trashed":        classDriveFile,
	"google.workspace.drive.file.v3.untrashed":      classDriveFile,
	"google.workspace.drive.file.v3.renamed":        classDriveFile,

	"google.workspace.drive.permission.v3.created": classDrivePermission,
	"google.workspace.drive.permission.v3.edited":  classDrivePermission,
	"google.workspace.drive.permission.v3.deleted": classDrivePermission,

	"google.workspace.meet.conference.v2.started":       classMeet,
	"google.workspace.meet.conference.v2.ended":         classMeet,
	"google.workspace.meet.participant.v2.joined":       classMeet,
	"google.workspace.meet.participant.v2.left":         classMeet,
	"google.workspace.meet.recording.v2.started":        classMeet,
	"google.workspace.meet.recording.v2.ended":          classMeet,
	"google.workspace.meet.recording.v2.fileGenerated":  classMeet,
	"google.workspace.meet.transcript.v2.started":       classMeet,
	"google.workspace.meet.transcript.v2.ended":         classMeet,
	"google.workspace.meet.transcript.v2.fileGenerated": classMeet,
	"google.workspace.meet.smartNote.v2.started":        classMeet,
	"google.workspace.meet.smartNote.v2.ended":          classMeet,
	"google.workspace.meet.smartNote.v2.fileGenerated":  classMeet,
}

// previewEventHints maps known Preview event families to the reason they
// are rejected, so the error explains the boundary instead of just saying
// "unknown".
var previewEventHints = []struct{ prefix, reason string }{
	{"google.workspace.drive.comment.", "Drive comment events are a Developer Preview feature"},
	{"google.workspace.drive.reply.", "Drive reply events are a Developer Preview feature"},
	{"google.workspace.drive.approval.", "Drive approval events are a Developer Preview feature"},
	{"google.workspace.drive.accessproposal.", "Drive access-proposal events are a Developer Preview feature"},
	{"google.workspace.chat.availability.", "Chat availability events are not in the supported GA set"},
}

const scopePrefix = "https://www.googleapis.com/auth/"

// Least-privilege user-auth scope per event class.
var userScopeByClass = map[string]string{
	classChatSpace:      scopePrefix + "chat.spaces.readonly",
	classChatMembership: scopePrefix + "chat.memberships.readonly",
	classChatMessage:    scopePrefix + "chat.messages.readonly",
	classChatReaction:   scopePrefix + "chat.messages.reactions.readonly",
	classChatReadState:  scopePrefix + "chat.users.readstate.readonly",
	// Drive subscriptions accept drive, drive.file, drive.readonly, and
	// drive.metadata — drive.metadata.readonly is not in the supported
	// set, so drive.metadata is the least-broad supported scope.
	classDriveFile:       scopePrefix + "drive.metadata",
	classDrivePermission: scopePrefix + "drive.metadata",
	classMeet:            scopePrefix + "meetings.space.readonly",
}

// MeetCreatedScope is the narrower Meet scope covering only meeting spaces
// the authenticated user created.
const MeetCreatedScope = scopePrefix + "meetings.space.created"

// Least-privilege app-auth (admin-approved Chat app) scope per event class.
// Only individual-space Chat subscriptions are supported with app auth;
// reactions share the chat.app.messages.readonly scope. Read-state has no
// GA app-auth scope (chat.app.all.users.readstate.readonly is Preview).
var appScopeByClass = map[string]string{
	classChatSpace:      scopePrefix + "chat.app.spaces.readonly",
	classChatMembership: scopePrefix + "chat.app.memberships.readonly",
	classChatMessage:    scopePrefix + "chat.app.messages.readonly",
	classChatReaction:   scopePrefix + "chat.app.messages.readonly",
}

var topicRe = regexp.MustCompile(`^projects/[^/\s]+/topics/[^/\s]+$`)

// ValidateTopic checks the Pub/Sub topic resource name format. gws never
// creates or replaces topics — the topic must already exist in the same
// Cloud project as the subscription.
func ValidateTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("missing Pub/Sub topic: pass --topic projects/PROJECT/topics/TOPIC (the topic must already exist in the subscription's Cloud project)")
	}
	if !topicRe.MatchString(topic) {
		return fmt.Errorf("invalid Pub/Sub topic %q: expected format projects/PROJECT/topics/TOPIC", topic)
	}
	return nil
}

// targetPrefixes maps each supported target resource prefix to its kind.
// The resource ID after the prefix must be non-empty.
var targetPrefixes = []struct{ prefix, kind string }{
	{"//chat.googleapis.com/spaces/", TargetChatSpace},
	{"//cloudidentity.googleapis.com/users/", TargetUser},
	{"//drive.googleapis.com/files/", TargetDriveFile},
	{"//drive.googleapis.com/drives/", TargetSharedDrive},
	{"//meet.googleapis.com/spaces/", TargetMeetSpace},
}

// ClassifyTarget parses a subscription target resource and returns its kind.
// Prefix-only targets with an empty resource ID are rejected.
func ClassifyTarget(target string) (string, error) {
	if target == "" {
		return "", fmt.Errorf("missing subscription target: pass --target with a resource like //chat.googleapis.com/spaces/SPACE, //drive.googleapis.com/files/FILE, or //meet.googleapis.com/spaces/SPACE")
	}
	if target == "//chat.googleapis.com/spaces/-" {
		return TargetChatAllSpaces, nil
	}
	if strings.HasPrefix(target, "//admin.googleapis.com/customers/") {
		return "", fmt.Errorf("customer-wide Chat subscriptions (%s) are a Developer Preview feature and not supported", target)
	}
	for _, tp := range targetPrefixes {
		if strings.HasPrefix(target, tp.prefix) {
			if strings.TrimPrefix(target, tp.prefix) == "" {
				return "", fmt.Errorf("invalid subscription target %q: missing resource ID after %s", target, tp.prefix)
			}
			return tp.kind, nil
		}
	}
	return "", fmt.Errorf("unsupported subscription target %q: supported targets are //chat.googleapis.com/spaces/{space|-}, //cloudidentity.googleapis.com/users/USER, //drive.googleapis.com/files/FILE, //drive.googleapis.com/drives/DRIVE, //meet.googleapis.com/spaces/SPACE", target)
}

// classesForTarget lists the event classes valid on each target kind.
var classesForTarget = map[string]map[string]bool{
	TargetChatSpace:     {classChatSpace: true, classChatMembership: true, classChatMessage: true, classChatReaction: true},
	TargetChatAllSpaces: {classChatSpace: true, classChatMembership: true, classChatMessage: true, classChatReaction: true},
	TargetUser:          {classChatMembership: true, classChatReadState: true, classMeet: true},
	TargetDriveFile:     {classDriveFile: true, classDrivePermission: true},
	TargetSharedDrive:   {classDriveFile: true, classDrivePermission: true},
	TargetMeetSpace:     {classMeet: true},
}

// Plan is a validated subscription request: the target kind, the event
// classes involved, and the least-privilege scopes the credentials need.
type Plan struct {
	TargetKind string
	EventTypes []string
	Scopes     []string
}

// PlanSubscription validates a target + event-type combination against the
// GA boundary and the auth mode, returning the least-privilege scopes.
// meetCreatedScope narrows the Meet scope to meetings.space.created.
func PlanSubscription(target string, eventTypes []string, authMode string, meetCreatedScope bool) (*Plan, error) {
	kind, err := ClassifyTarget(target)
	if err != nil {
		return nil, err
	}
	if len(eventTypes) == 0 {
		return nil, fmt.Errorf("missing event types: pass --event-types with one or more GA event types (see gws events subscriptions create --help)")
	}

	if authMode == auth.AuthModeApp && kind != TargetChatSpace {
		return nil, fmt.Errorf("auth_mode %q only supports subscriptions on an individual Chat space (//chat.googleapis.com/spaces/SPACE); target %q requires user credentials (OAuth or %q)",
			auth.AuthModeApp, target, auth.AuthModeDWD)
	}

	scopes := map[string]bool{}
	for _, et := range eventTypes {
		class, ok := gaEventClasses[et]
		if !ok {
			for _, hint := range previewEventHints {
				if strings.HasPrefix(et, hint.prefix) {
					return nil, fmt.Errorf("unsupported event type %q: %s", et, hint.reason)
				}
			}
			return nil, fmt.Errorf("unknown event type %q: run gws events subscriptions create --help for the supported GA event types", et)
		}
		if !classesForTarget[kind][class] {
			return nil, fmt.Errorf("event type %q is not valid for target %q", et, target)
		}

		if authMode == auth.AuthModeApp {
			scope, ok := appScopeByClass[class]
			if !ok {
				return nil, fmt.Errorf("event type %q requires user credentials: its only app-auth variant is a Developer Preview feature; use auth_mode %q or the default OAuth login", et, auth.AuthModeDWD)
			}
			scopes[scope] = true
			continue
		}

		scope := userScopeByClass[class]
		if class == classMeet && meetCreatedScope {
			scope = MeetCreatedScope
		}
		scopes[scope] = true
	}

	out := make([]string, 0, len(scopes))
	for s := range scopes {
		out = append(out, s)
	}
	sort.Strings(out)

	return &Plan{TargetKind: kind, EventTypes: eventTypes, Scopes: out}, nil
}

// ChatBotScope is the Chat app scope used for read/delete lifecycle
// operations under app auth.
const ChatBotScope = scopePrefix + "chat.bot"

// ReadLifecycleScopes returns the scopes for read/delete lifecycle
// operations (list, get, delete), where the subscription's event class is
// irrelevant. User modes use the union of least-privilege user-auth event
// scopes; app mode uses chat.bot per the official contract.
//
// Mutating operations (create, update/renew, reactivate) never use this:
// they must go through PlanSubscription with the subscription's exact
// target and event types so only the matching granular scope(s) are
// minted — under app auth, callers first Get the existing subscription
// with chat.bot to derive that plan.
func ReadLifecycleScopes(authMode string) []string {
	if authMode == auth.AuthModeApp {
		return []string{ChatBotScope}
	}
	return uniqueSortedScopes(userScopeByClass)
}

func uniqueSortedScopes(byClass map[string]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range byClass {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// CloudEvent is a decoded Pub/Sub notification from a Workspace Events
// subscription. Subscription comes from the ce-source attribute (the
// Workspace subscription resource); Target comes from ce-subject (the
// changed target resource).
type CloudEvent struct {
	EventType          string            `json:"event_type,omitempty"`
	Subscription       string            `json:"subscription,omitempty"`
	Target             string            `json:"target,omitempty"`
	Time               string            `json:"time,omitempty"`
	Attributes         map[string]string `json:"attributes,omitempty"`
	MessageID          string            `json:"message_id,omitempty"`
	PublishTime        string            `json:"publish_time,omitempty"`
	PubsubSubscription string            `json:"pubsub_subscription,omitempty"`
	Data               json.RawMessage   `json:"data,omitempty"`
}

// pubsubEnvelope matches both a Pub/Sub push delivery ({"message": {...},
// "subscription": ...}) and a bare Pub/Sub message object.
type pubsubEnvelope struct {
	Message      *pubsubMessage `json:"message"`
	Subscription string         `json:"subscription"`

	// bare-message fields (used when "message" is absent)
	Attributes  map[string]string `json:"attributes"`
	Data        string            `json:"data"`
	MessageID   string            `json:"messageId"`
	PublishTime string            `json:"publishTime"`
}

type pubsubMessage struct {
	Attributes  map[string]string `json:"attributes"`
	Data        string            `json:"data"`
	MessageID   string            `json:"messageId"`
	PublishTime string            `json:"publishTime"`
}

// DecodeCloudEvent reads a Pub/Sub JSON notification (push envelope or bare
// message) and decodes the CloudEvent attributes plus the base64 event data.
func DecodeCloudEvent(r io.Reader) (*CloudEvent, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read event from stdin: %w", err)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, fmt.Errorf("no event on stdin: pipe a Pub/Sub message JSON (push envelope or bare message) to gws events decode")
	}

	var env pubsubEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("invalid Pub/Sub message JSON: %w", err)
	}

	msg := env.Message
	if msg == nil {
		msg = &pubsubMessage{
			Attributes:  env.Attributes,
			Data:        env.Data,
			MessageID:   env.MessageID,
			PublishTime: env.PublishTime,
		}
	}

	ev := &CloudEvent{
		Attributes:  msg.Attributes,
		MessageID:   msg.MessageID,
		PublishTime: msg.PublishTime,
		// The push envelope's "subscription" is the Pub/Sub subscription,
		// not the Workspace Events subscription.
		PubsubSubscription: env.Subscription,
	}
	if msg.Attributes != nil {
		ev.EventType = msg.Attributes["ce-type"]
		ev.Time = msg.Attributes["ce-time"]
		ev.Subscription = msg.Attributes["ce-source"]
		ev.Target = msg.Attributes["ce-subject"]
	}

	if msg.Data != "" {
		decoded, err := base64.StdEncoding.DecodeString(msg.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to base64-decode event data: %w", err)
		}
		if json.Valid(decoded) {
			ev.Data = json.RawMessage(decoded)
		} else {
			// Non-JSON payloads are re-encoded as a JSON string.
			quoted, _ := json.Marshal(string(decoded))
			ev.Data = json.RawMessage(quoted)
		}
	}

	return ev, nil
}

// SupportedEventTypes returns the sorted list of supported GA event types
// (for help text and docs).
func SupportedEventTypes() []string {
	out := make([]string, 0, len(gaEventClasses))
	for et := range gaEventClasses {
		out = append(out, et)
	}
	sort.Strings(out)
	return out
}
