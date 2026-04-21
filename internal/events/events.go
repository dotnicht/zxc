package events

import (
	"encoding/json"

	"github.com/google/uuid"
)

type AccountCreated struct {
	AccountID uuid.UUID `json:"account_id"`
	Name      string    `json:"name"`
	RequestID uuid.UUID `json:"request_id"`
}

func (AccountCreated) Kind() string             { return "account_created" }
func (AccountCreated) AggregateType() string    { return "account" }
func (e AccountCreated) AggregateID() uuid.UUID { return e.AccountID }

type RequestAccountLinked struct {
	RequestID uuid.UUID `json:"request_id"`
	ReleaseID uuid.UUID `json:"release_id"`
	AccountID uuid.UUID `json:"account_id"`
	Name      string    `json:"name"`
}

func (RequestAccountLinked) Kind() string             { return "request_account_linked" }
func (RequestAccountLinked) AggregateType() string    { return "request" }
func (e RequestAccountLinked) AggregateID() uuid.UUID { return e.RequestID }

type TargetProbeSucceeded struct {
	TargetID uuid.UUID `json:"target_id"`
	Status   string    `json:"status"`
}

func (TargetProbeSucceeded) Kind() string             { return "target_probe_succeeded" }
func (TargetProbeSucceeded) AggregateType() string    { return "target" }
func (e TargetProbeSucceeded) AggregateID() uuid.UUID { return e.TargetID }

type TargetProbeFailed struct {
	TargetID uuid.UUID `json:"target_id"`
	Status   string    `json:"status"`
}

func (TargetProbeFailed) Kind() string             { return "target_probe_failed" }
func (TargetProbeFailed) AggregateType() string    { return "target" }
func (e TargetProbeFailed) AggregateID() uuid.UUID { return e.TargetID }

type ReleaseDeployed struct {
	ReleaseID   uuid.UUID `json:"release_id"`
	ChangedByID uuid.UUID `json:"changed_by_id"`
}

func (ReleaseDeployed) Kind() string             { return "release_deployed" }
func (ReleaseDeployed) AggregateType() string    { return "release" }
func (e ReleaseDeployed) AggregateID() uuid.UUID { return e.ReleaseID }

type ReleaseFailed struct {
	ReleaseID   uuid.UUID `json:"release_id"`
	ChangedByID uuid.UUID `json:"changed_by_id"`
}

func (ReleaseFailed) Kind() string             { return "release_failed" }
func (ReleaseFailed) AggregateType() string    { return "release" }
func (e ReleaseFailed) AggregateID() uuid.UUID { return e.ReleaseID }

type ReleaseHealthTimeout struct {
	ReleaseID uuid.UUID `json:"release_id"`
}

func (ReleaseHealthTimeout) Kind() string             { return "release_health_timeout" }
func (ReleaseHealthTimeout) AggregateType() string    { return "release" }
func (e ReleaseHealthTimeout) AggregateID() uuid.UUID { return e.ReleaseID }

type ReleaseAlive struct {
	ReleaseID uuid.UUID       `json:"release_id"`
	Body      json.RawMessage `json:"body"`
}

func (ReleaseAlive) Kind() string             { return "release_alive" }
func (ReleaseAlive) AggregateType() string    { return "release" }
func (e ReleaseAlive) AggregateID() uuid.UUID { return e.ReleaseID }

type TargetCreated struct {
	TargetID uuid.UUID `json:"target_id"`
	Address  string    `json:"address"`
	User     string    `json:"user"`
}

func (TargetCreated) Kind() string             { return "target_created" }
func (TargetCreated) AggregateType() string    { return "target" }
func (e TargetCreated) AggregateID() uuid.UUID { return e.TargetID }

type TargetUpdated struct {
	TargetID uuid.UUID `json:"target_id"`
	Address  string    `json:"address"`
	User     string    `json:"user"`
}

func (TargetUpdated) Kind() string             { return "target_updated" }
func (TargetUpdated) AggregateType() string    { return "target" }
func (e TargetUpdated) AggregateID() uuid.UUID { return e.TargetID }

type TargetDeleted struct {
	TargetID uuid.UUID `json:"target_id"`
}

func (TargetDeleted) Kind() string             { return "target_deleted" }
func (TargetDeleted) AggregateType() string    { return "target" }
func (e TargetDeleted) AggregateID() uuid.UUID { return e.TargetID }

type ReleaseCreated struct {
	ReleaseID   uuid.UUID `json:"release_id"`
	OwnerID     uuid.UUID `json:"owner_id"`
	TargetID    uuid.UUID `json:"target_id"`
	PayloadID   uuid.UUID `json:"payload_id"`
	ChangedByID uuid.UUID `json:"changed_by_id"`
	Status      string    `json:"status"`
}

func (ReleaseCreated) Kind() string             { return "release_created" }
func (ReleaseCreated) AggregateType() string    { return "release" }
func (e ReleaseCreated) AggregateID() uuid.UUID { return e.ReleaseID }

type ReleaseDeployRequested struct {
	ReleaseID uuid.UUID `json:"release_id"`
	UserID    uuid.UUID `json:"user_id"`
}

func (ReleaseDeployRequested) Kind() string             { return "release_deploy_requested" }
func (ReleaseDeployRequested) AggregateType() string    { return "release" }
func (e ReleaseDeployRequested) AggregateID() uuid.UUID { return e.ReleaseID }

type AccountDisabled struct {
	AccountID      uuid.UUID `json:"account_id"`
	PreviousStatus string    `json:"previous_status"`
}

func (AccountDisabled) Kind() string             { return "account_disabled" }
func (AccountDisabled) AggregateType() string    { return "account" }
func (e AccountDisabled) AggregateID() uuid.UUID { return e.AccountID }

type SessionCreated struct {
	SessionID uuid.UUID `json:"session_id"`
	AccountID uuid.UUID `json:"account_id"`
	Status    string    `json:"status"`
}

func (SessionCreated) Kind() string             { return "session_created" }
func (SessionCreated) AggregateType() string    { return "session" }
func (e SessionCreated) AggregateID() uuid.UUID { return e.SessionID }

type AccountActivated struct {
	AccountID uuid.UUID `json:"account_id"`
	SessionID uuid.UUID `json:"session_id"`
}

func (AccountActivated) Kind() string             { return "account_activated" }
func (AccountActivated) AggregateType() string    { return "account" }
func (e AccountActivated) AggregateID() uuid.UUID { return e.AccountID }

type SessionUpdated struct {
	SessionID uuid.UUID `json:"session_id"`
	AccountID uuid.UUID `json:"account_id"`
	Status    string    `json:"status"`
}

func (SessionUpdated) Kind() string             { return "session_updated" }
func (SessionUpdated) AggregateType() string    { return "session" }
func (e SessionUpdated) AggregateID() uuid.UUID { return e.SessionID }

type SessionDeleted struct {
	SessionID uuid.UUID `json:"session_id"`
	AccountID uuid.UUID `json:"account_id"`
	Status    string    `json:"status"`
}

func (SessionDeleted) Kind() string             { return "session_deleted" }
func (SessionDeleted) AggregateType() string    { return "session" }
func (e SessionDeleted) AggregateID() uuid.UUID { return e.SessionID }

type WebhookReceived struct {
	ReleaseID uuid.UUID       `json:"release_id"`
	RequestID uuid.UUID       `json:"request_id"`
	Body      json.RawMessage `json:"body"`
}

func (WebhookReceived) Kind() string             { return "webhook_received" }
func (WebhookReceived) AggregateType() string    { return "release" }
func (e WebhookReceived) AggregateID() uuid.UUID { return e.ReleaseID }
