package confluence

import (
	"bytes"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
)

// CreateFileAttachment uploads bytes without converting them to a string.
func (c *Client) CreateFileAttachment(attachment *Attachment, data []byte, pageId string) (*Attachment, error) {
	var response AttachmentResults
	path := fmt.Sprintf("/rest/api/content/%s/child/attachment", pageId)
	if err := c.postFileForm(path, attachment.Title, attachment.Metadata.MediaType, data, &response); err != nil {
		return nil, err
	}
	if len(response.Results) != 1 {
		return nil, errors.New("unexpected number of results returned when creating file attachment")
	}
	return &response.Results[0], nil
}

// UpdateFileAttachment uses Confluence's dedicated attachment-data endpoint.
func (c *Client) UpdateFileAttachment(attachment *Attachment, data []byte, pageId string) (*Attachment, error) {
	var response Attachment
	path := fmt.Sprintf("/rest/api/content/%s/child/attachment/%s/data", pageId, attachment.Id)
	if err := c.postFileForm(path, attachment.Title, attachment.Metadata.MediaType, data, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) postFileForm(path, filename, mediaType string, data []byte, result interface{}) error {
	body, contentType, err := fileFormBytesBuffer(filename, mediaType, data)
	if err != nil {
		return err
	}
	return c.do("POST", path, contentType, body, result)
}

func fileFormBytesBuffer(filename, mediaType string, data []byte) (*bytes.Buffer, string, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": filename}))
	header.Set("Content-Type", mediaType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(data); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return body, writer.FormDataContentType(), nil
}
