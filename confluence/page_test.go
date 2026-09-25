package confluence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func testClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return &Client{
		client:   server.Client(),
		baseURL:  baseURL,
		basePath: "/ex/confluence/cloud-123/wiki",
		apiToken: "secret-token",
	}
}

func TestGetSpaceByKeyUsesV2AndBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.RequestURI() != "/ex/confluence/cloud-123/wiki/api/v2/spaces?keys=DOC" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":"103415828","key":"DOC","name":"Docs","type":"global","status":"current","homepageId":"103416068"}]}`))
	}))
	defer server.Close()

	space, err := testClient(t, server).GetSpaceByKey("DOC")
	if err != nil {
		t.Fatal(err)
	}
	if space.ID != "103415828" || space.HomepageID != "103416068" {
		t.Fatalf("unexpected space: %#v", space)
	}
}

func TestCreatePageUsesV2Payload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/ex/confluence/cloud-123/wiki/api/v2/pages" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var page Page
		if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
			t.Fatal(err)
		}
		if page.SpaceID != "10" || page.Status != "current" || page.Title != "Page" {
			t.Fatalf("unexpected page payload: %#v", page)
		}
		if page.Body == nil || page.Body.Representation != "storage" || page.Body.Value != "<p>body</p>" {
			t.Fatalf("unexpected body payload: %#v", page.Body)
		}
		_, _ = w.Write([]byte(`{"id":"42","spaceId":"10","status":"current","title":"Page","version":{"number":1}}`))
	}))
	defer server.Close()

	created, err := testClient(t, server).CreatePage(&Page{
		SpaceID: "10", Status: "current", Title: "Page",
		Body: &PageBody{Representation: "storage", Value: "<p>body</p>"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "42" || created.Version.Number != 1 {
		t.Fatalf("unexpected created page: %#v", created)
	}
}

func TestUpdatePageReadsCurrentVersionAndIncrementsIt(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch {
		case r.Method == http.MethodGet:
			if r.URL.RequestURI() != "/ex/confluence/cloud-123/wiki/api/v2/pages/42?body-format=storage" {
				t.Fatalf("unexpected GET path: %s", r.URL.RequestURI())
			}
			_, _ = w.Write([]byte(`{"id":"42","version":{"number":7}}`))
		case r.Method == http.MethodPut:
			if r.URL.Path != "/ex/confluence/cloud-123/wiki/api/v2/pages/42" {
				t.Fatalf("unexpected PUT path: %s", r.URL.Path)
			}
			var page Page
			if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
				t.Fatal(err)
			}
			if page.Version == nil || page.Version.Number != 8 {
				t.Fatalf("expected version 8, got %#v", page.Version)
			}
			_, _ = w.Write([]byte(`{"id":"42","version":{"number":8}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	page := &Page{ID: "42", SpaceID: "10", Status: "current", Title: "Updated", Body: &PageBody{Representation: "storage", Value: "<p>updated</p>"}}
	updated, err := testClient(t, server).UpdatePage(page)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || updated.Version.Number != 8 {
		t.Fatalf("unexpected update result: requests=%d page=%#v", requests, updated)
	}
}

func TestResourcePageReadTreatsTrashedAsAbsent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"42","status":"trashed","version":{"number":3}}`))
	}))
	defer server.Close()

	d := schema.TestResourceDataRaw(t, resourcePage().Schema, map[string]interface{}{
		"space_id": "10", "title": "Page", "body": "<p>body</p>",
	})
	d.SetId("42")
	if err := resourcePageRead(d, testClient(t, server)); err != nil {
		t.Fatal(err)
	}
	if d.Id() != "" {
		t.Fatalf("trashed page remained in state with ID %q", d.Id())
	}
}

func TestResourcePageReadTreatsNotFoundAsAbsent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	d := schema.TestResourceDataRaw(t, resourcePage().Schema, map[string]interface{}{
		"space_id": "10", "title": "Page", "body": "<p>body</p>",
	})
	d.SetId("42")
	if err := resourcePageRead(d, testClient(t, server)); err != nil {
		t.Fatal(err)
	}
	if d.Id() != "" {
		t.Fatalf("missing page remained in state with ID %q", d.Id())
	}
}

func TestResourcePageDeleteUsesV2Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/ex/confluence/cloud-123/wiki/api/v2/pages/42" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	d := schema.TestResourceDataRaw(t, resourcePage().Schema, map[string]interface{}{
		"space_id": "10", "title": "Page", "body": "<p>body</p>",
	})
	d.SetId("42")
	if err := resourcePageDelete(d, testClient(t, server)); err != nil {
		t.Fatal(err)
	}
}

func TestResourcePageParentIDCanBePopulatedWhenOmitted(t *testing.T) {
	parentID := resourcePage().Schema["parent_id"]
	if !parentID.Optional || !parentID.Computed {
		t.Fatalf("parent_id must be optional and computed, got optional=%t computed=%t", parentID.Optional, parentID.Computed)
	}
}
