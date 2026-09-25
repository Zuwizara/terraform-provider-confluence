package confluence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client provides a connection to the Confluence API
type Client struct {
	client   *http.Client
	baseURL  *url.URL
	basePath string
	apiToken string
}

// NewClientInput provides information to connect to the Confluence API
type NewClientInput struct {
	cloudID  string
	apiToken string
}

// HTTPError is returned for non-successful Confluence API responses.
type HTTPError struct {
	StatusCode int
	Method     string
	Path       string
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("Confluence API returned HTTP %d for %s %s: %s", e.StatusCode, e.Method, e.Path, e.Message)
}

// ErrorResponse describes why a request failed
type ErrorResponse struct {
	StatusCode int `json:"statusCode,omitempty"`
	Data       struct {
		Authorized bool     `json:"authorized,omitempty"`
		Valid      bool     `json:"valid,omitempty"`
		Errors     []string `json:"errors,omitempty"`
		Successful bool     `json:"successful,omitempty"`
	} `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewClient returns an authenticated client ready to use
func NewClient(input *NewClientInput) *Client {
	baseURL := url.URL{
		Scheme: "https",
		Host:   "api.atlassian.com",
	}
	return &Client{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		baseURL:  &baseURL,
		basePath: fmt.Sprintf("/ex/confluence/%s/wiki", url.PathEscape(input.cloudID)),
		apiToken: input.apiToken,
	}
}

// GetString uses the client to send a GET request and returns a string
func (c *Client) GetString(path string) (string, error) {
	body := new(bytes.Buffer)
	responseBody, err := c.doRaw("GET", path, "", body)
	if err != nil {
		return "", err
	}
	result := responseBody.String()
	return result, nil
}

// Get uses the client to send a GET request
func (c *Client) Get(path string, result interface{}) error {
	body := new(bytes.Buffer)
	return c.do("GET", path, "", body, result)
}

// Delete uses the client to send a DELETE request
func (c *Client) Delete(path string) error {
	body := new(bytes.Buffer)
	return c.do("DELETE", path, "", body, nil)
}

// Post uses the client to send a POST request
func (c *Client) Post(path string, body interface{}, result interface{}) error {
	b, err := jsonBytesBuffer(body)
	if err != nil {
		return err
	}
	return c.do("POST", path, "application/json", b, result)
}

// Put uses the client to send a PUT request
func (c *Client) Put(path string, body interface{}, result interface{}) error {
	b, err := jsonBytesBuffer(body)
	if err != nil {
		return err
	}
	return c.do("PUT", path, "application/json", b, result)
}

// PostForm uses the client to send a multi-part form POST request
func (c *Client) PostForm(path, filename, body string, result interface{}) error {
	b, ct, err := formBytesBuffer(filename, body)
	if err != nil {
		return err
	}
	return c.do("POST", path, ct, b, result)
}

// PutForm uses the client to send a multi-part form PUT request
func (c *Client) PutForm(path, filename, body string, result interface{}) error {
	b, ct, err := formBytesBuffer(filename, body)
	if err != nil {
		return err
	}
	return c.do("PUT", path, ct, b, result)
}

func jsonBytesBuffer(body interface{}) (*bytes.Buffer, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(bodyBytes), nil
}

func bytesBufferJSON(bodyBytes *bytes.Buffer, result interface{}) error {
	if result == nil {
		return nil
	}
	reader := bytes.NewReader(bodyBytes.Bytes())
	return json.NewDecoder(reader).Decode(result)
}

// formBytesBuffer returns the body as a multi-part form and the content type
func formBytesBuffer(filename, body string) (*bytes.Buffer, string, error) {
	bodyBytes := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBytes)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", err
	}
	_, err = io.WriteString(part, body)
	if err != nil {
		return nil, "", err
	}
	err = writer.Close()
	if err != nil {
		return nil, "", err
	}
	return bodyBytes, writer.FormDataContentType(), nil
}

func (c *Client) do(method, path, contentType string, body *bytes.Buffer, result interface{}) error {
	responseBody, err := c.doRaw(method, path, contentType, body)
	if err != nil {
		return err
	}
	return bytesBufferJSON(responseBody, result)
}

// do uses the client to send a specified request
func (c *Client) doRaw(method, path, contentType string, body *bytes.Buffer) (*bytes.Buffer, error) {
	fullPath := c.basePath + path
	u, err := c.baseURL.Parse(fullPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Add("X-Atlassian-Token", "nocheck")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, readErr
		}
		message := strings.TrimSpace(string(responseBytes))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return nil, &HTTPError{StatusCode: resp.StatusCode, Method: method, Path: fullPath, Message: message}
	}
	result := new(bytes.Buffer)
	_, err = result.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func isNotFound(err error) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound
}

func (e *ErrorResponse) String() string {
	d := e.Data
	var errorsString string
	if len(d.Errors) > 0 {
		errorsString = fmt.Sprintf("\n  * %s", strings.Join(d.Errors, "\n  * "))
	}
	return fmt.Sprintf("%s\nAuthorized: %t\nValid: %t\nSuccessful: %t%s",
		e.Message, d.Authorized, d.Valid, d.Successful, errorsString)
}
