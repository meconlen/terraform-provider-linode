package linode

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/linode/linodego"
)

func resourceLinodeObjectStorageBucket() *schema.Resource {
	return &schema.Resource{
		Create: resourceLinodeObjectStorageBucketCreate,
		Read:   resourceLinodeObjectStorageBucketRead,
		Update: resourceLinodeObjectStorageBucketUpdate,
		Delete: resourceLinodeObjectStorageBucketDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"cluster": {
				Type:        schema.TypeString,
				Description: "The cluster of the Linode Object Storage Bucket.",
				Required:    true,
				ForceNew:    true,
			},
			"label": {
				Type:        schema.TypeString,
				Description: "The label of the Linode Object Storage Bucket.",
				Required:    true,
				ForceNew:    true,
			},
			"acl": {
				Type:        schema.TypeString,
				Description: "The Access Control Level of the bucket using a canned ACL string.",
				Optional:    true,
				Default:     "private",
			},
			"cors_enabled": {
				Type:        schema.TypeBool,
				Description: "If true, the bucket will be created with CORS enabled for all origins.",
				Optional:    true,
				Default:     true,
			},
			"cert": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"certificate": {
							Type:        schema.TypeString,
							Description: "The Base64 encoded and PEM formatted SSL certificate.",
							Sensitive:   true,
							Required:    true,
						},
						"private_key": {
							Type:        schema.TypeString,
							Description: "The private key associated with the TLS/SSL certificate.",
							Sensitive:   true,
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func resourceLinodeObjectStorageBucketRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderMeta).Client
	cluster, label, err := decodeLinodeObjectStorageBucketID(d.Id())
	if err != nil {
		return fmt.Errorf("failed to parse Linode ObjectStorageBucket id %s", d.Id())
	}

	bucket, err := client.GetObjectStorageBucket(context.Background(), cluster, label)
	if err != nil {
		return fmt.Errorf("failed to find the specified Linode ObjectStorageBucket: %s", err)
	}

	access, err := client.GetObjectStorageBucketAccess(context.Background(), cluster, label)
	if err != nil {
		return fmt.Errorf("failed to find the access config for the specified Linode ObjectStorageBucket: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", bucket.Cluster, bucket.Label))
	d.Set("cluster", bucket.Cluster)
	d.Set("label", bucket.Label)
	d.Set("acl", access.ACL)
	d.Set("cors_enabled", access.CorsEnabled)

	return nil
}

func resourceLinodeObjectStorageBucketCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderMeta).Client

	cluster := d.Get("cluster").(string)
	label := d.Get("label").(string)
	acl := d.Get("acl").(string)
	corsEnabled := d.Get("cors_enabled").(bool)
	cert := d.Get("cert").([]interface{})

	createOpts := linodego.ObjectStorageBucketCreateOptions{
		Cluster:     cluster,
		Label:       label,
		ACL:         linodego.ObjectStorageACL(acl),
		CorsEnabled: &corsEnabled,
	}

	bucket, err := client.CreateObjectStorageBucket(context.Background(), createOpts)
	if err != nil {
		return fmt.Errorf("failed to create a Linode ObjectStorageBucket: %s", err)
	}

	if len(cert) != 0 {
		uploadOpts := expandLinodeObjectStorageBucketCert(cert[0])
		if _, err := client.UploadObjectStorageBucketCert(context.Background(), cluster, label, uploadOpts); err != nil {
			return fmt.Errorf("failed to upload bucket cert: %s", err)
		}
	}

	d.SetId(fmt.Sprintf("%s:%s", bucket.Cluster, bucket.Label))
	d.Set("cluster", bucket.Cluster)
	d.Set("label", bucket.Label)
	d.Set("acl", acl)
	d.Set("cors_enabled", corsEnabled)

	return resourceLinodeObjectStorageBucketRead(d, meta)
}

func resourceLinodeObjectStorageBucketUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderMeta).Client

	if d.HasChanges("acl", "cors_enabled") {
		if err := updateLinodeObjectStorageBucketAccess(d, client); err != nil {
			return err
		}
	}

	if d.HasChange("cert") {
		if err := updateLinodeObjectStorageBucketCert(d, client); err != nil {
			return err
		}
	}

	return resourceLinodeObjectStorageBucketRead(d, meta)
}

func resourceLinodeObjectStorageBucketDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderMeta).Client
	cluster, label, err := decodeLinodeObjectStorageBucketID(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing Linode ObjectStorageBucket id %s", d.Id())
	}
	err = client.DeleteObjectStorageBucket(context.Background(), cluster, label)
	if err != nil {
		return fmt.Errorf("Error deleting Linode ObjectStorageBucket %s: %s", d.Id(), err)
	}
	return nil
}

func updateLinodeObjectStorageBucketAccess(d *schema.ResourceData, client linodego.Client) error {
	cluster := d.Get("cluster").(string)
	label := d.Get("label").(string)

	updateOpts := linodego.ObjectStorageBucketUpdateAccessOptions{}
	if d.HasChange("acl") {
		updateOpts.ACL = linodego.ObjectStorageACL(d.Get("acl").(string))
	}

	if d.HasChange("cors_enabled") {
		newCorsBool := d.Get("cors_enabled").(bool)
		updateOpts.CorsEnabled = &newCorsBool
	}

	if err := client.UpdateObjectStorageBucketAccess(context.Background(), cluster, label, updateOpts); err != nil {
		return fmt.Errorf("failed to update bucket access: %s", err)
	}

	return nil
}

func updateLinodeObjectStorageBucketCert(d *schema.ResourceData, client linodego.Client) error {
	cluster := d.Get("cluster").(string)
	label := d.Get("label").(string)
	oldCert, newCert := d.GetChange("cert")
	hasOldCert := len(oldCert.([]interface{})) != 0

	if hasOldCert {
		if err := client.DeleteObjectStorageBucketCert(context.Background(), cluster, label); err != nil {
			return fmt.Errorf("failed to delete old bucket cert: %s", err)
		}
	}

	certSpec := newCert.([]interface{})
	if len(certSpec) == 0 {
		return nil
	}

	uploadOptions := expandLinodeObjectStorageBucketCert(certSpec[0])
	if _, err := client.UploadObjectStorageBucketCert(context.Background(), cluster, label, uploadOptions); err != nil {
		return fmt.Errorf("failed to upload new bucket cert: %s", err)
	}
	return nil
}

func expandLinodeObjectStorageBucketCert(v interface{}) linodego.ObjectStorageBucketCertUploadOptions {
	certSpec := v.(map[string]interface{})
	return linodego.ObjectStorageBucketCertUploadOptions{
		Certificate: certSpec["certificate"].(string),
		PrivateKey:  certSpec["private_key"].(string),
	}
}

func decodeLinodeObjectStorageBucketID(id string) (cluster, label string, err error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		err = fmt.Errorf("Linode Object Storage Bucket ID must be of the form <Cluster>:<Label>, was provided: %s", id)
		return
	}
	cluster = parts[0]
	label = parts[1]
	return
}
