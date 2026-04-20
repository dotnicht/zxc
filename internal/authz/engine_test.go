package authz

import (
	"context"
	"testing"
)

func TestEmbeddedPolicy(t *testing.T) {
	engine, err := NewEmbedded()
	if err != nil {
		t.Fatalf("new embedded engine: %v", err)
	}

	testCases := []struct {
		name       string
		input      Input
		wantAllow  bool
		wantReveal bool
	}{
		{
			name: "root can manage tenants",
			input: Input{
				Subject: Subject{ID: "root", IsRoot: true},
				Action:  "tenant.create",
			},
			wantAllow: true,
		},
		{
			name: "tenant owner can create target",
			input: Input{
				Subject: Subject{ID: "owner"},
				Action:  "target.create",
				Tenant:  Tenant{ID: "tenant-1", OwnerID: "owner"},
			},
			wantAllow: true,
		},
		{
			name: "non owner cannot create target",
			input: Input{
				Subject: Subject{ID: "user-1"},
				Action:  "target.create",
				Tenant:  Tenant{ID: "tenant-1", OwnerID: "owner"},
			},
			wantAllow: false,
		},
		{
			name: "target owner can reveal secret",
			input: Input{
				Subject:  Subject{ID: "user-1"},
				Action:   "target.get",
				Tenant:   Tenant{ID: "tenant-1", OwnerID: "owner"},
				Resource: Resource{Type: "target", OwnerID: "user-1"},
			},
			wantAllow:  true,
			wantReveal: true,
		},
		{
			name: "tenant user can read target metadata without secret",
			input: Input{
				Subject:  Subject{ID: "user-2"},
				Action:   "target.get",
				Tenant:   Tenant{ID: "tenant-1", OwnerID: "owner"},
				Resource: Resource{Type: "target", OwnerID: "user-1"},
			},
			wantAllow: true,
		},
		{
			name: "release create requires both dependencies for non owner",
			input: Input{
				Subject: Subject{ID: "user-1"},
				Action:  "release.create",
				Tenant:  Tenant{ID: "tenant-1", OwnerID: "owner"},
				Related: Related{TargetOwnerID: "user-1", PayloadOwnerID: "user-1"},
			},
			wantAllow: true,
		},
		{
			name: "release deploy requires owner and unknown status",
			input: Input{
				Subject:  Subject{ID: "user-1"},
				Action:   "release.deploy",
				Tenant:   Tenant{ID: "tenant-1", OwnerID: "owner"},
				Resource: Resource{Type: "release", OwnerID: "user-1", Status: "unknown"},
			},
			wantAllow: true,
		},
		{
			name: "release deploy denied for non unknown status",
			input: Input{
				Subject:  Subject{ID: "user-1"},
				Action:   "release.deploy",
				Tenant:   Tenant{ID: "tenant-1", OwnerID: "owner"},
				Resource: Resource{Type: "release", OwnerID: "user-1", Status: "alive"},
			},
			wantAllow: false,
		},
		{
			name: "system may transition deployed to alive",
			input: Input{
				Subject:  Subject{ID: "system", System: true},
				Action:   "release.transition",
				Tenant:   Tenant{ID: "tenant-1", OwnerID: "owner"},
				Resource: Resource{Type: "release", Status: "deployed", NextStatus: "alive"},
			},
			wantAllow: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := engine.Evaluate(context.Background(), tc.input)
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			if decision.Allow != tc.wantAllow {
				t.Fatalf("allow=%v want %v", decision.Allow, tc.wantAllow)
			}
			if decision.RevealSecret != tc.wantReveal {
				t.Fatalf("reveal_secret=%v want %v", decision.RevealSecret, tc.wantReveal)
			}
		})
	}
}
