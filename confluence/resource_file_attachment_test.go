package confluence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAttachmentRequiresExactlyOneContentSource(t *testing.T) {
	for _, test := range []struct {
		name      string
		content   map[string]interface{}
		wantError bool
	}{
		{name: "missing", content: map[string]interface{}{}, wantError: true},
		{name: "both", content: map[string]interface{}{"data": "inline", "file_path": "attachment.txt"}, wantError: true},
		{name: "inline", content: map[string]interface{}{"data": "inline"}},
		{name: "file", content: map[string]interface{}{"file_path": "attachment.txt"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := map[string]interface{}{"title": "attachment.txt", "page_id": "123"}
			for key, value := range test.content {
				config[key] = value
			}
			diagnostics := resourceAttachment().Validate(terraform.NewResourceConfigRaw(config))
			if diagnostics.HasError() != test.wantError {
				t.Fatalf("validation error = %t, want %t: %#v", diagnostics.HasError(), test.wantError, diagnostics)
			}
		})
	}
}

func TestFileAttachmentSHA256IsCalculatedDuringPlanning(t *testing.T) {
	contents := []byte("first version")
	path := filepath.Join(t.TempDir(), "attachment.txt")
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
	resource := resourceAttachment()
	if !resource.Schema["file_sha256"].Computed || resource.Schema["file_sha256"].Required {
		t.Fatal("file_sha256 must be computed rather than required")
	}
	config := terraform.NewResourceConfigRaw(map[string]interface{}{
		"file_path": path,
		"title":     "attachment.txt",
		"page_id":   "123",
	})
	diff, err := resource.Diff(context.Background(), nil, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%x", sha256.Sum256(contents))
	if got := diff.Attributes["file_sha256"].New; got != want {
		t.Fatalf("planned file_sha256 = %q, want %q", got, want)
	}

	state := &terraform.InstanceState{ID: "456", Attributes: map[string]string{
		"file_path": path, "file_sha256": want, "title": "attachment.txt",
		"media_type": "text/plain", "page_id": "123", "version": "1",
	}}
	updatedContents := []byte("second version")
	if err := os.WriteFile(path, updatedContents, 0600); err != nil {
		t.Fatal(err)
	}
	diff, err = resource.Diff(context.Background(), state, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantUpdated := fmt.Sprintf("%x", sha256.Sum256(updatedContents))
	hashDiff := diff.Attributes["file_sha256"]
	if hashDiff == nil || hashDiff.Old != want || hashDiff.New != wantUpdated {
		t.Fatalf("file content change produced unexpected diff: %#v", hashDiff)
	}
}

func TestFileAttachmentUploadDoesNotStoreContents(t *testing.T) {
	for _, test := range []struct {
		name      string
		contents  []byte
		mediaType string
	}{
		{"image", []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0xff}, "image/png"},
		{"text", []byte("example text attachment"), "text/plain"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "attachment")
			if err := os.WriteFile(path, test.contents, 0600); err != nil {
				t.Fatal(err)
			}
			uploads := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost:
					if uploads == 0 && r.URL.Path != "/rest/api/content/123/child/attachment" {
						t.Errorf("create path = %q", r.URL.Path)
					}
					if uploads == 1 && r.URL.Path != "/rest/api/content/123/child/attachment/456/data" {
						t.Errorf("update path = %q", r.URL.Path)
					}
					if r.Header.Get("X-Atlassian-Token") != "nocheck" {
						t.Error("missing X-Atlassian-Token header")
					}
					reader, err := r.MultipartReader()
					if err != nil {
						t.Error(err)
						return
					}
					part, err := reader.NextPart()
					if err != nil {
						t.Error(err)
						return
					}
					got, err := io.ReadAll(part)
					if err != nil {
						t.Error(err)
					}
					if !bytes.Equal(got, test.contents) {
						t.Errorf("uploaded bytes = %v, want %v", got, test.contents)
					}
					if part.FileName() != "attachment" || part.FormName() != "file" || part.Header.Get("Content-Type") != test.mediaType {
						t.Errorf("multipart headers = %#v", part.Header)
					}
					w.Header().Set("Content-Type", "application/json")
					if uploads == 0 {
						_, _ = io.WriteString(w, `{"results":[{"id":"456"}]}`)
					} else {
						_, _ = io.WriteString(w, `{"id":"456"}`)
					}
					uploads++
				case r.Method == http.MethodGet && r.URL.Path == "/api/v2/attachments/456":
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, fmt.Sprintf(`{"id":"456","title":"attachment","mediaType":%q,"version":{"number":%d}}`, test.mediaType, uploads))
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusNotFound)
				}
			}))
			defer server.Close()
			baseURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client := &Client{client: server.Client(), baseURL: baseURL}
			resource := resourceAttachment()
			d := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
				"file_path":   path,
				"file_sha256": fmt.Sprintf("%x", sha256.Sum256(test.contents)),
				"title":       "attachment",
				"media_type":  test.mediaType,
				"page_id":     "123",
			})
			if err := resourceAttachmentCreate(d, client); err != nil {
				t.Fatal(err)
			}
			if d.Id() != "456" || d.Get("version").(int) != 1 {
				t.Errorf("state after create: id=%q version=%v", d.Id(), d.Get("version"))
			}
			if err := resourceAttachmentUpdate(d, client); err != nil {
				t.Fatal(err)
			}
			if uploads != 2 || d.Get("version").(int) != 2 {
				t.Errorf("state after update: uploads=%d version=%v", uploads, d.Get("version"))
			}
			state, err := json.Marshal(d.State())
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(state), string(test.contents)) {
				t.Fatal("file contents found in Terraform state")
			}
		})
	}
}

func TestFileAttachmentRejectsChangedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "attachment.txt")
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	d := schema.TestResourceDataRaw(t, resourceAttachment().Schema, map[string]interface{}{
		"file_path":   path,
		"file_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte("original"))),
		"title":       "attachment.txt",
		"page_id":     "123",
	})
	if _, err := readAttachmentFile(d); err == nil {
		t.Fatal("expected an error when the file changed after its hash was calculated")
	}
}

func TestResourceAttachmentFileModeReadTreatsNotFoundAsAbsent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	d := schema.TestResourceDataRaw(t, resourceAttachment().Schema, map[string]interface{}{
		"file_path": "unused", "file_sha256": strings.Repeat("0", 64), "title": "attachment", "page_id": "123",
	})
	d.SetId("456")
	if err := resourceAttachmentRead(d, testClient(t, server)); err != nil {
		t.Fatal(err)
	}
	if d.Id() != "" {
		t.Fatalf("missing file attachment remained in state with ID %q", d.Id())
	}
}
