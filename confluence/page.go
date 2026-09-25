package confluence

import (
	"fmt"
	"net/url"
)

type SpaceResults struct {
	Results []SpaceV2 `json:"results"`
}

type SpaceV2 struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	HomepageID string `json:"homepageId"`
}

type Page struct {
	ID       string     `json:"id,omitempty"`
	SpaceID  string     `json:"spaceId,omitempty"`
	ParentID string     `json:"parentId,omitempty"`
	Status   string     `json:"status,omitempty"`
	Title    string     `json:"title,omitempty"`
	Body     *PageBody  `json:"body,omitempty"`
	Version  *Version   `json:"version,omitempty"`
	Links    *PageLinks `json:"_links,omitempty"`
}

type PageBody struct {
	Representation string   `json:"representation,omitempty"`
	Value          string   `json:"value,omitempty"`
	Storage        *Storage `json:"storage,omitempty"`
}

type PageLinks struct {
	Base  string `json:"base,omitempty"`
	WebUI string `json:"webui,omitempty"`
}

type Storage struct {
	Value          string `json:"value,omitempty"`
	Representation string `json:"representation,omitempty"`
}

type Version struct {
	Number int `json:"number,omitempty"`
}

func (c *Client) GetSpaceByKey(key string) (*SpaceV2, error) {
	var response SpaceResults
	path := "/api/v2/spaces?keys=" + url.QueryEscape(key)
	if err := c.Get(path, &response); err != nil {
		return nil, err
	}
	if len(response.Results) == 0 {
		return nil, fmt.Errorf("Confluence space with key %q was not found", key)
	}
	if len(response.Results) > 1 {
		return nil, fmt.Errorf("Confluence returned multiple spaces for key %q", key)
	}
	return &response.Results[0], nil
}

func (c *Client) CreatePage(page *Page) (*Page, error) {
	var response Page
	if err := c.Post("/api/v2/pages", page, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetPage(id string) (*Page, error) {
	var response Page
	path := fmt.Sprintf("/api/v2/pages/%s?body-format=storage", url.PathEscape(id))
	if err := c.Get(path, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UpdatePage(page *Page) (*Page, error) {
	current, err := c.GetPage(page.ID)
	if err != nil {
		return nil, err
	}
	if current.Version == nil {
		return nil, fmt.Errorf("Confluence page %s has no version", page.ID)
	}
	page.Version = &Version{Number: current.Version.Number + 1}
	var response Page
	path := fmt.Sprintf("/api/v2/pages/%s", url.PathEscape(page.ID))
	if err := c.Put(path, page, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeletePage(id string) error {
	return c.Delete(fmt.Sprintf("/api/v2/pages/%s", url.PathEscape(id)))
}
