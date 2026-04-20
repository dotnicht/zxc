package storage

import "testing"

func TestParseConnectionString(t *testing.T) {
	cfg, bucket, prefix, err := ParseConnectionString("https://user:pass@example.com:9000/tenant-bucket/releases/assets")
	if err != nil {
		t.Fatalf("ParseConnectionString returned error: %v", err)
	}

	if cfg.Endpoint != "example.com:9000" {
		t.Fatalf("endpoint=%q want %q", cfg.Endpoint, "example.com:9000")
	}
	if cfg.AccessKey != "user" || cfg.SecretKey != "pass" {
		t.Fatalf("unexpected credentials: %+v", cfg)
	}
	if !cfg.UseSSL {
		t.Fatalf("UseSSL=false want true")
	}
	if bucket != "tenant-bucket" {
		t.Fatalf("bucket=%q want %q", bucket, "tenant-bucket")
	}
	if prefix != "releases/assets/" {
		t.Fatalf("prefix=%q want %q", prefix, "releases/assets/")
	}
}

func TestParseConnectionStringInvalid(t *testing.T) {
	if _, _, _, err := ParseConnectionString("://bad"); err == nil {
		t.Fatalf("expected invalid connection string error")
	}
}

func TestBucketConnectionString(t *testing.T) {
	client := &Client{cfg: Config{
		Endpoint:  "storage.example:9000",
		AccessKey: "minio",
		SecretKey: "secret",
		UseSSL:    true,
	}}

	got := client.BucketConnectionString("tenant-a")
	want := "https://minio:secret@storage.example:9000/tenant-a"
	if got != want {
		t.Fatalf("connection string=%q want %q", got, want)
	}
}

func TestBucketName(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "normalizes symbols", in: "Tenant Name!", want: "tenant-name"},
		{name: "pads short name", in: "a", want: "a--"},
		{name: "trims long name", in: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz", want: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijk"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BucketName(tc.in); got != tc.want {
				t.Fatalf("BucketName(%q)=%q want %q", tc.in, got, tc.want)
			}
		})
	}
}
