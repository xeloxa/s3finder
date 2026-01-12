package scanner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// InspectResult contains detailed information about a discovered bucket.
type InspectResult struct {
	Bucket      string    `json:"bucket"`
	Exists      bool      `json:"exists"`
	IsPublic    bool      `json:"is_public"`
	ACL         string    `json:"acl"`
	Region      string    `json:"region"`
	ObjectCount int       `json:"object_count"`
	SampleKeys  []string  `json:"sample_keys,omitempty"`
	Error       string    `json:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// Inspector performs deep inspection on discovered buckets using AWS SDK.
type Inspector struct {
	timeout time.Duration
}

// NewInspector creates a new Inspector.
func NewInspector(timeout time.Duration) *Inspector {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Inspector{timeout: timeout}
}

// Inspect performs deep analysis on a bucket.
func (i *Inspector) Inspect(ctx context.Context, bucket string) *InspectResult {
	result := &InspectResult{
		Bucket:      bucket,
		Exists:      true,
		ObjectCount: -1,
		Timestamp:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	// Get bucket region first
	region, err := i.getBucketRegion(ctx, bucket)
	if err != nil {
		result.Error = fmt.Sprintf("region lookup failed: %v", err)
		result.Region = "unknown"
	} else {
		result.Region = region
	}

	// Check ACL and attempt object listing
	isPublic, acl, objects, count := i.checkPublicAccess(ctx, bucket, region)
	result.IsPublic = isPublic
	result.ACL = acl
	result.ObjectCount = count
	result.SampleKeys = objects

	return result
}

// getBucketRegion determines which AWS region hosts the bucket.
func (i *Inspector) getBucketRegion(ctx context.Context, bucket string) (string, error) {
	// Use anonymous credentials for region lookup
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return "", err
	}

	client := s3.NewFromConfig(cfg)

	// GetBucketLocation returns the region
	output, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		// Try to extract region from error message or headers
		if strings.Contains(err.Error(), "PermanentRedirect") {
			// Parse region from redirect
			return i.parseRegionFromError(err.Error()), nil
		}
		return "us-east-1", nil // Default to us-east-1
	}

	region := string(output.LocationConstraint)
	if region == "" {
		region = "us-east-1" // Empty means us-east-1
	}

	return region, nil
}

// checkPublicAccess attempts anonymous listing to determine if bucket is public.
func (i *Inspector) checkPublicAccess(ctx context.Context, bucket, region string) (bool, string, []string, int) {
	if region == "" || region == "unknown" {
		region = "us-east-1"
	}

	// Create anonymous client
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return false, "unknown", nil, -1
	}

	client := s3.NewFromConfig(cfg)

	// Try to list objects anonymously
	output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(100),
	})
	if err != nil {
		// Check if it's an access denied error
		if strings.Contains(err.Error(), "AccessDenied") {
			return false, "private", nil, -1
		}
		if strings.Contains(err.Error(), "AllAccessDisabled") {
			return false, "disabled", nil, -1
		}
		return false, "unknown", nil, -1
	}

	// Successfully listed objects - bucket is public!
	var keys []string
	for _, obj := range output.Contents {
		if obj.Key != nil {
			keys = append(keys, *obj.Key)
			if len(keys) >= 10 {
				break
			}
		}
	}

	count := int(*output.KeyCount)
	if output.IsTruncated != nil && *output.IsTruncated {
		count = -2 // Indicates more than returned
	}

	return true, "public-read", keys, count
}

// parseRegionFromError extracts region from AWS error messages.
func (i *Inspector) parseRegionFromError(errMsg string) string {
	regions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
		"ap-south-1", "ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2",
		"sa-east-1", "ca-central-1",
	}

	for _, r := range regions {
		if strings.Contains(errMsg, r) {
			return r
		}
	}

	return "us-east-1"
}
