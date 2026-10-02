package confluence

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestPagePropertyValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inputs map[string]interface{}
		valid  bool
	}{
		{"string", map[string]interface{}{"value": "full-width"}, true},
		{"empty string", map[string]interface{}{"value": ""}, true},
		{"JSON null", map[string]interface{}{"value_json": "null"}, true},
		{"JSON array", map[string]interface{}{"value_json": "[1,true]"}, true},
		{"missing value", nil, false},
		{"both values", map[string]interface{}{"value": "full-width", "value_json": "true"}, false},
		{"both empty and JSON", map[string]interface{}{"value": "", "value_json": "true"}, false},
		{"invalid JSON", map[string]interface{}{"value_json": "full-width"}, false},
		{"empty JSON", map[string]interface{}{"value_json": ""}, false},
		{"blank key", map[string]interface{}{"value": "x", "key": "  "}, false},
		{"invalid page ID", map[string]interface{}{"value": "x", "page_id": "../42"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := map[string]interface{}{"page_id": "42", "key": "layout"}
			for key, value := range tc.inputs {
				config[key] = value
			}
			diags := Provider().ValidateResource("confluence_page_property", terraform.NewResourceConfigRaw(config))
			if diags.HasError() == tc.valid {
				t.Fatalf("unexpected validation diagnostics: %#v", diags)
			}
		})
	}
}

func TestPagePropertyJSON(t *testing.T) {
	for _, tc := range []struct {
		old, new string
		equal    bool
	}{
		{`{"a":1,"b":[true,null]}`, "{\n\"b\": [true, null], \"a\": 1}", true},
		{`{"n":9007199254740993}`, `{"n":9007199254740992}`, false},
		{`"true"`, `true`, false},
		{`"123"`, `123`, false},
		{`null`, `""`, false},
		{`"\u0061"`, `"a"`, true},
		{`[1,2]`, `[2,1]`, false},
		{`{}`, `{} {}`, false},
		{`invalid`, `invalid`, false},
	} {
		if got := pagePropertyDiffJSON("", tc.old, tc.new, nil); got != tc.equal {
			t.Errorf("compare %s and %s: got %t, want %t", tc.old, tc.new, got, tc.equal)
		}
	}
	for _, value := range []string{"", "true", "123", "full-width", "a\n\"b"} {
		d := schema.TestResourceDataRaw(t, resourcePageProperty().Schema, map[string]interface{}{"value": value})
		var decoded string
		if err := json.Unmarshal(pagePropertyValue(d), &decoded); err != nil || decoded != value {
			t.Fatalf("string %q was not encoded correctly: %q (%v)", value, decoded, err)
		}
	}
}

func TestFindPagePropertyPagination(t *testing.T) {
	requests := 0
	key := "layout & / ü?"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/ex/confluence/cloud-123/wiki/api/v2/pages/42/properties" || r.URL.Query().Get("key") != key {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Error("missing bearer token")
		}
		switch requests {
		case 1:
			_, _ = w.Write([]byte(`{"results":[],"_links":{"next":"https://untrusted.invalid/wiki/api/v2/pages/42/properties?cursor=a%2Bb"}}`))
		case 2:
			if r.URL.Query().Get("cursor") != "a+b" {
				t.Errorf("incorrect cursor: %s", r.URL)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": []PageProperty{{ID: "99", Key: key, Value: json.RawMessage(`"full-width"`)}}})
		default:
			t.Error("unexpected extra request")
		}
	}))
	defer server.Close()
	property, err := testClient(t, server).FindPageProperty("42", key)
	if err != nil || property == nil || property.ID != "99" || requests != 2 {
		t.Fatalf("unexpected lookup: %#v, %v, %d requests", property, err, requests)
	}
}

func TestFindPagePropertyInvalidPaginationAndDuplicates(t *testing.T) {
	for _, body := range []string{
		`{"results":[],"_links":{"next":"?cursor=repeated"}}`,
		`{"results":[],"_links":{"next":"/no-cursor"}}`,
		`{"results":[{"id":"1","key":"layout"},{"id":"2","key":"layout"}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			if _, err := testClient(t, server).FindPageProperty("42", "layout"); err == nil {
				t.Fatal("expected lookup error")
			}
		})
	}
}

func TestPagePropertyCreateAndAdopt(t *testing.T) {
	for _, tc := range []struct {
		name            string
		existing, adopt bool
	}{
		{"create", false, false}, {"reject duplicate", true, false}, {"adopt", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				collection := "/ex/confluence/cloud-123/wiki/api/v2/pages/42/properties"
				if r.Method == http.MethodGet {
					if r.URL.Path == collection {
						if tc.existing {
							_, _ = w.Write([]byte(`{"results":[{"id":"99","key":"layout","value":"old","version":{"number":7}}]}`))
						} else {
							_, _ = w.Write([]byte(`{"results":[]}`))
						}
						return
					}
					if r.URL.Path != collection+"/99" {
						t.Errorf("unexpected path: %s", r.URL.Path)
					}
					_, _ = w.Write([]byte(`{"id":"99","key":"layout","value":"full-width","version":{"number":7}}`))
					return
				}
				writes++
				var request PageProperty
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				if request.Key != "layout" || string(request.Value) != `"full-width"` || request.ID != "" {
					t.Errorf("wrong payload: %#v", request)
				}
				if tc.existing {
					if r.Method != http.MethodPut || r.URL.Path != collection+"/99" || request.Version == nil || request.Version.Number != 8 {
						t.Errorf("incorrect update: %s %s %#v", r.Method, r.URL, request)
					}
				} else if r.Method != http.MethodPost || r.URL.Path != collection || request.Version != nil {
					t.Errorf("incorrect create: %s %s %#v", r.Method, r.URL, request)
				}
				_, _ = w.Write([]byte(`{"id":"99","key":"layout","value":"full-width","version":{"number":8}}`))
			}))
			defer server.Close()
			d := schema.TestResourceDataRaw(t, resourcePageProperty().Schema, map[string]interface{}{
				"page_id": "42", "key": "layout", "value": "full-width", "adopt_existing": tc.adopt,
			})
			err := resourcePagePropertyCreate(d, testClient(t, server))
			if tc.existing && !tc.adopt {
				if err == nil || !strings.Contains(err.Error(), "42/99") || writes != 0 || d.Id() != "" {
					t.Fatalf("expected import hint without mutation: %v, %d writes, id %s", err, writes, d.Id())
				}
			} else if err != nil || d.Id() != "42/99" || d.Get("value") != "full-width" || writes != 1 {
				t.Fatalf("unexpected create/adopt: %v, %d writes, state %#v", err, writes, d.State())
			}
		})
	}
}

func TestPagePropertyReadAndImport(t *testing.T) {
	for _, tc := range []struct {
		name, remote, value, valueJSON string
		config                         map[string]interface{}
	}{
		{"import string", `"true"`, "true", "", nil},
		{"import object", `{"id":9007199254740993}`, "", `{"id":9007199254740993}`, nil},
		{"import null", `null`, "", `null`, nil},
		{"JSON string", `"full-width"`, "", `"full-width"`, map[string]interface{}{"value_json": `"old"`}},
		{"string type drift", `true`, "", `true`, map[string]interface{}{"value": ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprintf(w, `{"id":"99","key":"layout","value":%s,"version":{"number":3}}`, tc.remote)
			}))
			defer server.Close()
			d := schema.TestResourceDataRaw(t, resourcePageProperty().Schema, tc.config)
			d.SetId("42/99")
			if _, err := resourcePagePropertyImport(t.Context(), d, nil); err != nil {
				t.Fatal(err)
			}
			if err := resourcePagePropertyRead(d, testClient(t, server)); err != nil {
				t.Fatal(err)
			}
			if d.Get("value") != tc.value || d.Get("value_json") != tc.valueJSON || d.Get("page_id") != "42" || d.Get("property_id") != "99" {
				t.Fatalf("unexpected state: %#v", d.State())
			}
		})
	}
	for _, id := range []string{"", "42", "42/", "/99", "42/99/1", "a/99", "0/99", "42/../99"} {
		if _, _, err := parsePagePropertyID(id); err == nil {
			t.Errorf("accepted invalid import ID %q", id)
		}
	}
}

func TestPagePropertyErrors(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 409, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "failed", status) }))
			defer server.Close()
			client := testClient(t, server)
			for _, action := range []func(*schema.ResourceData, interface{}) error{resourcePagePropertyRead, resourcePagePropertyDelete} {
				d := schema.TestResourceDataRaw(t, resourcePageProperty().Schema, nil)
				d.SetId("42/99")
				err := action(d, client)
				if status == 404 {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					var apiError *HTTPError
					if !errors.As(err, &apiError) || apiError.StatusCode != status || d.Id() != "42/99" {
						t.Fatalf("lost state or API error: %v, %s", err, d.Id())
					}
				}
			}
			if _, err := client.FindPageProperty("42", "layout"); err == nil {
				t.Fatal("lookup errors must not be treated as missing keys")
			}
		})
	}
}

func TestPagePropertyVersionConflict(t *testing.T) {
	puts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"id":"99","key":"layout","value":false,"version":{"number":12}}`))
			return
		}
		puts++
		http.Error(w, "version conflict", http.StatusConflict)
	}))
	defer server.Close()
	_, err := testClient(t, server).UpdatePageProperty("42", "99", "layout", json.RawMessage(`true`))
	var apiError *HTTPError
	if !errors.As(err, &apiError) || apiError.StatusCode != 409 || puts != 1 {
		t.Fatalf("expected conflict without blind retry: %v, %d writes", err, puts)
	}
}
