package authz

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/open-policy-agent/opa/v1/rego"
)

//go:embed policy.rego
var policySource string

type Subject struct {
	ID     string `json:"id"`
	IsRoot bool   `json:"is_root,omitempty"`
	System bool   `json:"system,omitempty"`
}

type Tenant struct {
	ID      string `json:"id,omitempty"`
	OwnerID string `json:"owner_id,omitempty"`
}

type Resource struct {
	Type       string `json:"type,omitempty"`
	OwnerID    string `json:"owner_id,omitempty"`
	Status     string `json:"status,omitempty"`
	NextStatus string `json:"next_status,omitempty"`
}

type Related struct {
	TargetOwnerID  string `json:"target_owner_id,omitempty"`
	PayloadOwnerID string `json:"payload_owner_id,omitempty"`
}

type Input struct {
	Subject  Subject  `json:"subject"`
	Action   string   `json:"action"`
	Tenant   Tenant   `json:"tenant"`
	Resource Resource `json:"resource"`
	Related  Related  `json:"related"`
}

type Decision struct {
	Allow        bool   `json:"allow"`
	RevealSecret bool   `json:"reveal_secret"`
	Reason       string `json:"reason"`
}

type Engine struct {
	query rego.PreparedEvalQuery
}

var (
	defaultOnce sync.Once
	defaultAuth *Engine
	defaultErr  error
)

func Default() (*Engine, error) {
	defaultOnce.Do(func() {
		defaultAuth, defaultErr = NewEmbedded()
	})
	return defaultAuth, defaultErr
}

func EvaluateDefault(ctx context.Context, input Input) (Decision, error) {
	engine, err := Default()
	if err != nil {
		return Decision{}, err
	}
	return engine.Evaluate(ctx, input)
}

func NewEmbedded() (*Engine, error) {
	query, err := rego.New(
		rego.Query("data.zxc.authz.decision"),
		rego.Module("policy.rego", policySource),
	).PrepareForEval(context.Background())
	if err != nil {
		return nil, fmt.Errorf("prepare authz policy: %w", err)
	}
	return &Engine{query: query}, nil
}

func (e *Engine) Evaluate(ctx context.Context, input Input) (Decision, error) {
	results, err := e.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return Decision{}, fmt.Errorf("eval authz policy: %w", err)
	}
	if len(results) == 0 || len(results[0].Expressions) == 0 {
		return Decision{}, fmt.Errorf("authz policy returned no decision")
	}
	raw, err := json.Marshal(results[0].Expressions[0].Value)
	if err != nil {
		return Decision{}, fmt.Errorf("marshal authz decision: %w", err)
	}
	var decision Decision
	if err := json.Unmarshal(raw, &decision); err != nil {
		return Decision{}, fmt.Errorf("decode authz decision: %w", err)
	}
	return decision, nil
}
