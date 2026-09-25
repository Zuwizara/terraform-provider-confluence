package confluence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAttachment() *schema.Resource {
	return &schema.Resource{
		Create:        resourceAttachmentCreate,
		Read:          resourceAttachmentRead,
		Update:        resourceAttachmentUpdate,
		Delete:        resourceAttachmentDelete,
		CustomizeDiff: resourceAttachmentCustomizeDiff,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"data": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"data", "file_path"},
			},
			"file_path": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"data", "file_path"},
			},
			"file_sha256": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"title": {
				Type:     schema.TypeString,
				Required: true,
			},
			"media_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "text/plain",
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"page_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAttachmentCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if !d.NewValueKnown("data") || !d.NewValueKnown("file_path") {
		return d.SetNewComputed("file_sha256")
	}
	data := []byte(d.Get("data").(string))
	if path := d.Get("file_path").(string); path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read attachment file %q: %w", path, err)
		}
	}
	return d.SetNew("file_sha256", attachmentSHA256(data))
}

func resourceAttachmentCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	attachmentRequest := attachmentFromResourceData(d)
	pageID := d.Get("page_id").(string)
	var attachmentResponse *Attachment
	var err error
	if d.Get("file_path").(string) != "" {
		var data []byte
		data, err = readAttachmentFile(d)
		if err == nil {
			attachmentResponse, err = client.CreateFileAttachment(attachmentRequest, data, pageID)
		}
	} else {
		attachmentResponse, err = client.CreateAttachment(attachmentRequest, d.Get("data").(string), pageID)
	}
	if err != nil {
		return err
	}
	d.SetId(attachmentResponse.Id)
	return resourceAttachmentRead(d, m)
}

func resourceAttachmentRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	attachmentResponse, err := client.GetAttachment(d.Id())
	if isNotFound(err) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}
	if d.Get("file_path").(string) == "" {
		attachmentData, err := client.GetAttachmentBody(attachmentResponse)
		if err != nil {
			return err
		}
		if err := d.Set("data", attachmentData); err != nil {
			return err
		}
		if err := d.Set("file_sha256", attachmentSHA256([]byte(attachmentData))); err != nil {
			return err
		}
	}
	return updateResourceDataFromAttachment(d, attachmentResponse)
}

func resourceAttachmentUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	attachmentRequest := attachmentFromResourceData(d)
	pageID := d.Get("page_id").(string)
	var err error
	if d.Get("file_path").(string) != "" {
		var data []byte
		data, err = readAttachmentFile(d)
		if err == nil {
			_, err = client.UpdateFileAttachment(attachmentRequest, data, pageID)
		}
	} else {
		_, err = client.UpdateAttachment(attachmentRequest, d.Get("data").(string), pageID)
	}
	if err != nil {
		return err
	}
	return resourceAttachmentRead(d, m)
}

func resourceAttachmentDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	err := client.DeleteAttachment(d.Id())
	if err != nil {
		return err
	}
	// d.SetId("") is automatically called assuming delete returns no errors
	return nil
}

func attachmentFromResourceData(d *schema.ResourceData) *Attachment {
	result := &Attachment{
		Id:   d.Id(),
		Type: "attachment",
		Metadata: &Metadata{
			MediaType: d.Get("media_type").(string),
		},
		Title: d.Get("title").(string),
	}
	version := d.Get("version").(int) // Get returns 0 if unset
	if version > 0 {
		result.Version = &Version{Number: version}
	}
	return result
}

func updateResourceDataFromAttachment(d *schema.ResourceData, attachment *Attachment) error {
	d.SetId(attachment.Id)
	mediaType := attachment.MediaType
	if mediaType == "" && attachment.Metadata != nil {
		mediaType = attachment.Metadata.MediaType
	}
	version := 0
	if attachment.Version != nil {
		version = attachment.Version.Number
	}
	m := map[string]interface{}{
		"title":      attachment.Title,
		"version":    version,
		"media_type": mediaType,
	}
	if attachment.PageID != "" {
		m["page_id"] = attachment.PageID
	}
	for k, v := range m {
		err := d.Set(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func readAttachmentFile(d *schema.ResourceData) ([]byte, error) {
	path := d.Get("file_path").(string)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read attachment file %q: %w", path, err)
	}
	if actual := attachmentSHA256(data); actual != d.Get("file_sha256").(string) {
		return nil, fmt.Errorf("attachment file %q changed since its SHA-256 was calculated", path)
	}
	return data, nil
}

func attachmentSHA256(data []byte) string {
	checksum := sha256.Sum256(data)
	return hex.EncodeToString(checksum[:])
}
