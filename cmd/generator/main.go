package main

import (
	"context"
	"math/rand"
	"strings"

	"github.com/hashicorp/go-plugin"
	genplugin "zxc/plugin/generator"
)

var sentences = []string{
	"Hello, how are you doing today?",
	"Just checking in to see how things are going.",
	"Did you get a chance to look at the latest update?",
	"Let me know if you need anything from my end.",
	"I think we are making good progress here.",
	"Can we schedule a call to discuss this further?",
	"Looking forward to hearing your thoughts on this.",
	"Thanks for the quick response, really appreciate it.",
	"I will follow up with more details shortly.",
	"Everything seems to be on track so far.",
}

const prefix = "zxc-gen:"

func randomText() string {
	n := rand.Intn(3) + 1
	picked := make([]string, n)
	for i := range picked {
		picked[i] = sentences[rand.Intn(len(sentences))]
	}
	return prefix + " " + strings.Join(picked, " ")
}

type randomGenerator struct{}

func (g *randomGenerator) Post(_ context.Context, _ string) (string, error) {
	return randomText(), nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: genplugin.Handshake,
		Plugins: map[string]plugin.Plugin{
			"generator": &genplugin.Plugin{Impl: &randomGenerator{}},
		},
	})
}
