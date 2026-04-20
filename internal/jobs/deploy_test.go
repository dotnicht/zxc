package jobs

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"zxc/internal/consts"
)

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := io.WriteString(f, body); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

func readZip(t *testing.T, content []byte) map[string]string {
	t.Helper()

	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}

	out := make(map[string]string, len(r.File))
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", f.Name, err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zip entry %s: %v", f.Name, err)
		}
		out[f.Name] = string(body)
	}
	return out
}

func TestInjectConfigRewritesConfigAndEmbedsKey(t *testing.T) {
	zipContent := makeZip(t, map[string]string{
		"app.conf": consts.URL + "\n" + consts.AUTH + "\n",
		"run.sh":   "echo ok",
	})

	got, err := injectConfig(zipContent, "app.conf", "rel-123", "https://hook.example", "ssh-private-key")
	if err != nil {
		t.Fatalf("injectConfig returned error: %v", err)
	}

	files := readZip(t, got)
	if files["app.conf"] != "https://hook.example\nrel-123\n" {
		t.Fatalf("unexpected config contents: %q", files["app.conf"])
	}
	if files["run.sh"] != "echo ok" {
		t.Fatalf("unexpected run.sh contents: %q", files["run.sh"])
	}
	if files["key"] != "ssh-private-key" {
		t.Fatalf("unexpected key contents: %q", files["key"])
	}
}

func TestInjectConfigWithoutKey(t *testing.T) {
	zipContent := makeZip(t, map[string]string{
		"app.conf": consts.URL + "\n",
	})

	got, err := injectConfig(zipContent, "app.conf", "rel-1", "http://hook", "")
	if err != nil {
		t.Fatalf("injectConfig returned error: %v", err)
	}

	files := readZip(t, got)
	if _, ok := files["key"]; ok {
		t.Fatalf("key file should not be present when key is empty")
	}
}

func TestInjectConfigRejectsInvalidZip(t *testing.T) {
	if _, err := injectConfig([]byte("not-a-zip"), "app.conf", "rel-1", "http://hook", ""); err == nil {
		t.Fatalf("expected invalid zip error")
	}
}
