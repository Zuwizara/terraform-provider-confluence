package confluence

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// PageProperty is a JSON value attached to a page, with its own version.
type PageProperty struct {
	ID      string          `json:"id,omitempty"`
	Key     string          `json:"key"`
	Value   json.RawMessage `json:"value"`
	Version *Version        `json:"version,omitempty"`
}

func pagePropertyPath(pageID string) string {
	return "/api/v2/pages/" + url.PathEscape(pageID) + "/properties"
}

func (c *Client) GetPageProperty(pageID, propertyID string) (*PageProperty, error) {
	var result PageProperty
	if err := c.Get(pagePropertyPath(pageID)+"/"+url.PathEscape(propertyID), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// FindPageProperty returns nil only when the page exists but the key does not.
func (c *Client) FindPageProperty(pageID, key string) (*PageProperty, error) {
	query := url.Values{"key": {key}}
	seen := map[string]bool{}
	var found *PageProperty
	for {
		var result struct {
			Results []PageProperty `json:"results"`
			Links   struct {
				Next string `json:"next"`
			} `json:"_links"`
		}
		if err := c.Get(pagePropertyPath(pageID)+"?"+query.Encode(), &result); err != nil {
			return nil, err
		}
		for _, property := range result.Results {
			if property.Key != key {
				continue
			}
			if found != nil {
				return nil, fmt.Errorf("multiple properties with key %q on page %s", key, pageID)
			}
			found = &property
		}
		if result.Links.Next == "" {
			return found, nil
		}
		// Rebuild the request locally; never forward credentials to a next-link host.
		next, err := url.Parse(result.Links.Next)
		if err != nil {
			return nil, fmt.Errorf("invalid property pagination link: %w", err)
		}
		cursor := next.Query().Get("cursor")
		if cursor == "" || seen[cursor] {
			return nil, fmt.Errorf("missing or repeated property pagination cursor")
		}
		seen[cursor] = true
		query.Set("cursor", cursor)
	}
}

func (c *Client) CreatePageProperty(pageID, key string, value json.RawMessage) (*PageProperty, error) {
	var result PageProperty
	if err := c.Post(pagePropertyPath(pageID), &PageProperty{Key: key, Value: value}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdatePageProperty(pageID, propertyID, key string, value json.RawMessage) (*PageProperty, error) {
	current, err := c.GetPageProperty(pageID, propertyID)
	if err != nil {
		return nil, err
	}
	if current.Version == nil || current.Version.Number < 1 {
		return nil, fmt.Errorf("Confluence property %s/%s has no valid version", pageID, propertyID)
	}
	request := &PageProperty{Key: key, Value: value, Version: &Version{Number: current.Version.Number + 1}}
	var result PageProperty
	if err := c.Put(pagePropertyPath(pageID)+"/"+url.PathEscape(propertyID), request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeletePageProperty(pageID, propertyID string) error {
	return c.Delete(pagePropertyPath(pageID) + "/" + url.PathEscape(propertyID))
}
