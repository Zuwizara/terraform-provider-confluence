package confluence

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAttachmentUsesV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/ex/confluence/cloud-123/wiki/api/v2/attachments/456" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"456","title":"document.pdf","mediaType":"application/pdf","downloadLink":"/download/attachments/123/document.pdf","version":{"number":4}}`))
	}))
	defer server.Close()

	attachment, err := testClient(t, server).GetAttachment("456")
	if err != nil {
		t.Fatal(err)
	}
	if attachment.Id != "456" || attachment.MediaType != "application/pdf" || attachment.Version.Number != 4 {
		t.Fatalf("unexpected attachment: %#v", attachment)
	}
}

func TestDeleteAttachmentUsesV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/ex/confluence/cloud-123/wiki/api/v2/attachments/456" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := testClient(t, server).DeleteAttachment("456"); err != nil {
		t.Fatal(err)
	}
}
