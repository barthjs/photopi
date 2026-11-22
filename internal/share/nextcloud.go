package share

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// nextcloudDialTimeout bounds connecting to the Nextcloud instance, so a booth without internet notices it
	// within seconds instead of waiting for the operating system to give up.
	nextcloudDialTimeout = 5 * time.Second
	// nextcloudRequestTimeout bounds one WebDAV or OCS request. An upload consists of many requests,
	// so its overall duration grows with the number of files instead of sharing one fixed budget.
	nextcloudRequestTimeout = 60 * time.Second
)

// nextcloudProvider uploads files to a Nextcloud instance over WebDAV
// and shares the resulting folder through the OCS share API.
type nextcloudProvider struct {
	baseURL  string
	username string
	password string
	folder   string
	client   *http.Client
}

// newNextcloudProvider creates a new nextcloudProvider instance.
func newNextcloudProvider(options Options) *nextcloudProvider {
	return &nextcloudProvider{
		baseURL:  options.NextcloudURL,
		username: options.NextcloudUsername,
		password: options.NextcloudPassword,
		folder:   options.NextcloudFolder,
		client:   newNextcloudClient(),
	}
}

// newNextcloudClient returns an HTTP client with bounded connection setup,
// since the default client waits as long as the operating system lets a connection attempt run.
func newNextcloudClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: nextcloudDialTimeout,
	}

	transport := http.DefaultTransport.(*http.Transport).Clone() //nolint:errcheck // The default transport is always an *http.Transport.
	transport.DialContext = dialer.DialContext
	transport.TLSHandshakeTimeout = nextcloudDialTimeout
	transport.ResponseHeaderTimeout = nextcloudRequestTimeout

	return &http.Client{
		Transport: transport,
		Timeout:   nextcloudRequestTimeout,
	}
}

// TestConnection verifies that the configured Nextcloud instance is reachable
// and the credentials are accepted, using a WebDAV PROPFIND on the user's root folder.
func (p *nextcloudProvider) TestConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	webdavBase := fmt.Sprintf("%s/remote.php/dav/files/%s", p.baseURL, p.username)

	req, err := http.NewRequestWithContext(ctx, "PROPFIND", webdavBase, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.username, p.password)
	req.Header.Set("Depth", "0")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("share: nextcloud connection failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusMultiStatus, http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("share: nextcloud rejected the credentials (status %d)", resp.StatusCode)
	default:
		return fmt.Errorf("share: nextcloud returned unexpected status %d", resp.StatusCode)
	}
}

// Upload creates a remoteFolder (nested under the configured root folder) via WebDAV,
// PUTs every file into it, and creates a public read-only share link for the folder.
func (p *nextcloudProvider) Upload(ctx context.Context, remoteFolder string, files []File) (string, error) {
	remotePath := strings.Trim(p.folder, "/")
	if remotePath == "" {
		remotePath = strings.Trim(remoteFolder, "/")
	} else {
		remotePath = remotePath + "/" + strings.Trim(remoteFolder, "/")
	}

	webdavBase := fmt.Sprintf("%s/remote.php/dav/files/%s", p.baseURL, p.username)

	// Create every intermediate folder from the root down
	segments := strings.Split(remotePath, "/")
	var built strings.Builder
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		_, _ = built.WriteString("/" + segment)
		if err := p.mkcol(ctx, webdavBase+built.String()); err != nil {
			return "", fmt.Errorf("share: nextcloud could not create folder %q: %w", strings.TrimPrefix(built.String(), "/"), err)
		}
	}

	for _, file := range files {
		if err := p.put(ctx, fmt.Sprintf("%s/%s/%s", webdavBase, remotePath, file.Name), file.Data); err != nil {
			return "", fmt.Errorf("share: nextcloud upload of %q failed: %w", file.Name, err)
		}
	}

	existing, err := p.findShare(ctx, remotePath)
	if err != nil {
		return "", err
	}
	if existing != "" {
		return existing, nil
	}

	return p.createShare(ctx, remotePath)
}

// mkcol ensures a folder exists with a WebDAV MKCOL request, which answers 405 for an existing one.
func (p *nextcloudProvider) mkcol(ctx context.Context, url string) error {
	// Any other failure is reported, since the uploads would otherwise fail with a status that does not name the folder
	req, err := http.NewRequestWithContext(ctx, "MKCOL", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.username, p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed || (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil
	}
	return fmt.Errorf("unexpected status %d", resp.StatusCode)
}

// put uploads data to a single WebDAV path.
func (p *nextcloudProvider) put(ctx context.Context, url string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.username, p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ocsShareResponse is the subset of the Nextcloud OCS share API response used here.
// https://docs.nextcloud.com/server/stable/developer_manual/client_apis/OCS/ocs-share-api.html#create-a-new-share.
type ocsShareResponse struct {
	OCS struct {
		Meta struct {
			Status     string `json:"status"`
			StatusCode int    `json:"statuscode"`
			Message    string `json:"message"`
		} `json:"meta"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"ocs"`
}

// publicLinkShareType is the OCS share type of public link.
const publicLinkShareType = 3

// ocsShareListResponse is the subset of the Nextcloud OCS response listing the shares of a path.
type ocsShareListResponse struct {
	OCS struct {
		Meta struct {
			Status     string `json:"status"`
			StatusCode int    `json:"statuscode"`
			Message    string `json:"message"`
		} `json:"meta"`
		Data []struct {
			ShareType int    `json:"share_type"`
			URL       string `json:"url"`
		} `json:"data"`
	} `json:"ocs"`
}

// findShare returns the URL of an existing public link share of remotePath, empty if there is none,
// so a repeated upload of a run hands out the same link instead of creating one more.
func (p *nextcloudProvider) findShare(ctx context.Context, remotePath string) (string, error) {
	query := url.Values{
		"path":     {"/" + strings.TrimLeft(remotePath, "/")},
		"reshares": {"false"},
		"format":   {"json"},
	}
	listURL := p.baseURL + "/ocs/v2.php/apps/files_sharing/api/v1/shares?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(p.username, p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("listing shares: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var parsed ocsShareListResponse
	if err := json.UnmarshalRead(resp.Body, &parsed); err != nil {
		return "", fmt.Errorf("decoding share list response: %w", err)
	}
	if parsed.OCS.Meta.Status != "ok" {
		return "", fmt.Errorf("nextcloud OCS API error: %d %s", parsed.OCS.Meta.StatusCode, parsed.OCS.Meta.Message)
	}

	for _, existing := range parsed.OCS.Data {
		if existing.ShareType == publicLinkShareType && existing.URL != "" {
			return existing.URL, nil
		}
	}

	return "", nil
}

// createShare creates a public read-only link share for remotePath and returns its URL.
func (p *nextcloudProvider) createShare(ctx context.Context, remotePath string) (string, error) {
	shareURL := p.baseURL + "/ocs/v2.php/apps/files_sharing/api/v1/shares?format=json"

	form := url.Values{
		"path":        {"/" + strings.TrimLeft(remotePath, "/")},
		"shareType":   {strconv.Itoa(publicLinkShareType)},
		"permissions": {"1"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shareURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.username, p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("creating share: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var parsed ocsShareResponse
	if err := json.UnmarshalRead(resp.Body, &parsed); err != nil {
		return "", fmt.Errorf("decoding share response: %w", err)
	}

	if parsed.OCS.Meta.Status != "ok" {
		return "", fmt.Errorf("nextcloud OCS API error: %d %s", parsed.OCS.Meta.StatusCode, parsed.OCS.Meta.Message)
	}

	return parsed.OCS.Data.URL, nil
}
