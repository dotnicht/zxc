package main

import (
	"context"
	"flag"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

	"zxc/internal/config"
	"zxc/internal/storage"
)

type pageData struct {
	Buckets       []storage.BucketInfo
	CurrentBucket string
	CurrentPrefix string
	ParentPrefix  string
	Directories   []entry
	Files         []entry
}

type entry struct {
	Name     string
	Key      string
	Link     string
	Size     int64
	Modified string
}

var pageTmpl = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Storage Browser</title>
  <style>
    :root {
      --bg: #f7f4ea;
      --panel: #fffdf6;
      --line: #d7ccb3;
      --ink: #201a12;
      --muted: #6f6557;
      --accent: #0f766e;
      --accent-2: #d97706;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      background: linear-gradient(180deg, #efe7d1 0%, var(--bg) 100%);
      color: var(--ink);
    }
    .wrap {
      max-width: 1100px;
      margin: 0 auto;
      padding: 24px;
    }
    .header, .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 16px;
      box-shadow: 0 10px 30px rgba(32, 26, 18, 0.06);
    }
    .header {
      padding: 20px 22px;
      margin-bottom: 18px;
    }
    h1 {
      margin: 0 0 6px;
      font-size: 28px;
    }
    .muted {
      color: var(--muted);
    }
    .grid {
      display: grid;
      grid-template-columns: 280px 1fr;
      gap: 18px;
    }
    .panel {
      padding: 18px;
    }
    .bucket-list, .item-list {
      list-style: none;
      margin: 0;
      padding: 0;
    }
    .bucket-list li + li, .item-list li + li {
      margin-top: 10px;
    }
    a {
      color: var(--accent);
      text-decoration: none;
    }
    a:hover { text-decoration: underline; }
    .active {
      color: var(--accent-2);
      font-weight: 700;
    }
    .crumb {
      margin-bottom: 16px;
      color: var(--muted);
      word-break: break-all;
    }
    .section-title {
      margin: 0 0 12px;
      font-size: 14px;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--muted);
    }
    .row {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 12px 14px;
      background: #fff;
    }
    .name {
      overflow-wrap: anywhere;
    }
    .meta {
      white-space: nowrap;
      color: var(--muted);
      font-size: 12px;
      text-align: right;
    }
    @media (max-width: 800px) {
      .grid { grid-template-columns: 1fr; }
      .row { flex-direction: column; }
      .meta { text-align: left; white-space: normal; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="header">
      <h1>Storage Browser</h1>
      <div class="muted">No MinIO console login. Browse buckets and objects directly.</div>
    </div>

    <div class="grid">
      <div class="panel">
        <div class="section-title">Buckets</div>
        <ul class="bucket-list">
          {{range .Buckets}}
          <li>
            <a href="/?bucket={{.Name}}" class="{{if eq $.CurrentBucket .Name}}active{{end}}">{{.Name}}</a>
          </li>
          {{else}}
          <li class="muted">No buckets found.</li>
          {{end}}
        </ul>
      </div>

      <div class="panel">
        {{if .CurrentBucket}}
        <div class="section-title">Contents</div>
        <div class="crumb">
          bucket=<strong>{{.CurrentBucket}}</strong>{{if .CurrentPrefix}} / prefix=<strong>{{.CurrentPrefix}}</strong>{{end}}
        </div>
        {{if .ParentPrefix}}
        <div style="margin-bottom: 12px;">
          <a href="/?bucket={{.CurrentBucket}}&prefix={{.ParentPrefix}}">.. up one level</a>
        </div>
        {{else if .CurrentPrefix}}
        <div style="margin-bottom: 12px;">
          <a href="/?bucket={{.CurrentBucket}}">.. back to bucket root</a>
        </div>
        {{end}}

        <ul class="item-list">
          {{range .Directories}}
          <li>
            <div class="row">
              <div class="name"><a href="{{.Link}}">{{.Name}}/</a></div>
              <div class="meta">folder</div>
            </div>
          </li>
          {{end}}
          {{range .Files}}
          <li>
            <div class="row">
              <div class="name"><a href="{{.Link}}">{{.Name}}</a></div>
              <div class="meta">{{.Size}} bytes{{if .Modified}}<br>{{.Modified}}{{end}}</div>
            </div>
          </li>
          {{end}}
          {{if and (eq (len .Directories) 0) (eq (len .Files) 0)}}
          <li class="muted">This location is empty.</li>
          {{end}}
        </ul>
        {{else}}
        <div class="muted">Pick a bucket to browse.</div>
        {{end}}
      </div>
    </div>
  </div>
</body>
</html>`))

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	configPath := flag.String("config", "config.toml", "path to configuration file")
	port := flag.String("port", "19001", "http listen port")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	client, _, err := storage.ClientFromConnectionString(cfg.Storage)
	if err != nil {
		slog.Error("failed to create storage client", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderIndex(w, r, client)
	})
	mux.HandleFunc("/raw", func(w http.ResponseWriter, r *http.Request) {
		serveObject(w, r, client)
	})

	addr := ":" + *port
	slog.Info("starting storage ui", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("storage ui stopped", "error", err)
		os.Exit(1)
	}
}

func renderIndex(w http.ResponseWriter, r *http.Request, client *storage.Client) {
	ctx := r.Context()

	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Name < buckets[j].Name })

	bucket := r.URL.Query().Get("bucket")
	prefix := normalizePrefix(r.URL.Query().Get("prefix"))

	data := pageData{
		Buckets:       buckets,
		CurrentBucket: bucket,
		CurrentPrefix: prefix,
		ParentPrefix:  parentPrefix(prefix),
		Directories:   []entry{},
		Files:         []entry{},
	}

	if bucket != "" {
		objects, err := client.ListObjects(ctx, bucket, prefix, false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, obj := range objects {
			name := strings.TrimPrefix(obj.Key, prefix)
			name = strings.TrimSuffix(name, "/")
			if name == "" {
				continue
			}

			if obj.IsDir {
				data.Directories = append(data.Directories, entry{
					Name: name,
					Key:  obj.Key,
					Link: "/?bucket=" + url.QueryEscape(bucket) + "&prefix=" + url.QueryEscape(obj.Key),
				})
				continue
			}

			modified := ""
			if !obj.LastModified.IsZero() {
				modified = obj.LastModified.UTC().Format("2006-01-02 15:04:05 UTC")
			}
			data.Files = append(data.Files, entry{
				Name:     name,
				Key:      obj.Key,
				Link:     "/raw?bucket=" + url.QueryEscape(bucket) + "&key=" + url.QueryEscape(obj.Key),
				Size:     obj.Size,
				Modified: modified,
			})
		}
	}

	if err := pageTmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func serveObject(w http.ResponseWriter, r *http.Request, client *storage.Client) {
	ctx := context.Background()
	bucket := r.URL.Query().Get("bucket")
	key := r.URL.Query().Get("key")
	if bucket == "" || key == "" {
		http.Error(w, "bucket and key are required", http.StatusBadRequest)
		return
	}

	obj, err := client.Download(ctx, bucket, key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer obj.Close()

	w.Header().Set("Content-Disposition", "inline; filename=\""+path.Base(key)+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, obj); err != nil {
		slog.Error("stream object", "bucket", bucket, "key", key, "error", err)
	}
}

func normalizePrefix(prefix string) string {
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix == "" {
		return ""
	}
	if strings.HasSuffix(prefix, "/") {
		return prefix
	}
	return prefix + "/"
}

func parentPrefix(prefix string) string {
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return ""
	}
	idx := strings.LastIndex(prefix, "/")
	if idx == -1 {
		return ""
	}
	return prefix[:idx+1]
}
