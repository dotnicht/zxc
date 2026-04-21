package jobs

import "testing"

func TestExtractNodeName(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
		ok   bool
	}{
		{name: "top level node_name", body: `{"node_name":"node-a"}`, want: "node-a", ok: true},
		{name: "camel case", body: `{"nodeName":"node-b"}`, want: "node-b", ok: true},
		{name: "string node", body: `{"node":"node-c"}`, want: "node-c", ok: true},
		{name: "nested node", body: `{"node":{"name":"node-d"}}`, want: "node-d", ok: true},
		{name: "missing", body: `{"event":"alive"}`, want: "", ok: false},
		{name: "blank", body: `{"node_name":"  "}`, want: "", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := extractNodeName([]byte(tc.body))
			if ok != tc.ok {
				t.Fatalf("ok=%v want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
