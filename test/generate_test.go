package test

import (
	"fmt"
	"testing"
	"time"
)

func TestGenerateJob(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("gentest%d", time.Now().UnixNano())
	log("creating tenant %q for generate job test", name)
	runClient(t, "tenant", "add", "--name", name)

	adb := tenantAccountDB(t, name)
	var profileID string
	if err := adb.QueryRow(
		`INSERT INTO profiles (name, status) VALUES ('testnode', 'unknown') RETURNING id::text`,
	).Scan(&profileID); err != nil {
		t.Fatalf("insert test profile: %v", err)
	}
	log("inserted profile %s", profileID)

	deadline := time.Now().Add(120 * time.Second)
	var postID, talkID, contactID string
	for time.Now().Before(deadline) {
		_ = adb.QueryRow(`
			SELECT id::text, talk_id::text, contact_id::text
			FROM posts
			WHERE profile_id = $1 AND deleted_at IS NULL
			LIMIT 1
		`, profileID).Scan(&postID, &talkID, &contactID)
		if postID != "" {
			break
		}
		time.Sleep(3 * time.Second)
	}

	if postID == "" {
		t.Fatal("generate job did not create any posts within timeout")
	}
	if talkID == "" {
		t.Fatal("post has no talk_id")
	}
	if contactID == "" {
		t.Fatal("post has no contact_id")
	}
	log("generate job created post=%s talk=%s contact=%s", postID, talkID, contactID)

	var talkProfileID string
	if err := adb.QueryRow(`SELECT profile_id::text FROM talks WHERE id = $1`, talkID).Scan(&talkProfileID); err != nil {
		t.Fatalf("talk not found: %v", err)
	}
	if talkProfileID != profileID {
		t.Fatalf("talk belongs to wrong profile: %q", talkProfileID)
	}

	var contactProfileID string
	if err := adb.QueryRow(`SELECT profile_id::text FROM contacts WHERE id = $1`, contactID).Scan(&contactProfileID); err != nil {
		t.Fatalf("contact not found: %v", err)
	}
	if contactProfileID != profileID {
		t.Fatalf("contact belongs to wrong profile: %q", contactProfileID)
	}
}
