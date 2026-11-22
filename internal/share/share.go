// Package share uploads completed capture runs to a configurable cloud storage provider
// and returns a public share URL for guests.
package share

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrUnsupported is returned by the provider of DriverNone, which has nowhere to upload to.
	ErrUnsupported = errors.New("share: no cloud share provider configured")
)

// Driver selects the cloud share provider.
type Driver string

const (
	DriverNone      Driver = ""
	DriverNextcloud Driver = "nextcloud"
	DriverSelf      Driver = "self"
)

// File is one file of an upload with its name and content.
type File struct {
	Name string
	Data []byte
}

// Provider is an interface for uploading files to a remote storage provider.
type Provider interface {
	// TestConnection verifies that the configured destination of the provider is reachable and accepts the credentials.
	TestConnection(ctx context.Context) error
	// Upload uploads files into a remoteFolder, creating it if necessary, and returns a public share URL for that folder.
	Upload(ctx context.Context, remoteFolder string, files []File) (shareURL string, err error)
}

// Options holds the settings required to construct a Provider instance.
type Options struct {
	Driver            Driver
	NextcloudURL      string
	NextcloudUsername string
	NextcloudPassword string
	NextcloudFolder   string
}

// New creates a Provider for the configured Driver, whose uploads fail with ErrUnsupported for DriverNone
// and DriverSelf, which shares through the share page of the application.
func New(options Options) (Provider, error) {
	switch options.Driver {
	case DriverNone:
		return noneProvider{}, nil
	case DriverNextcloud:
		return newNextcloudProvider(options), nil
	case DriverSelf:
		return noneProvider{}, nil
	default:
		return nil, fmt.Errorf("share: unknown driver %q", options.Driver)
	}
}

// Enabled reports whether p uploads somewhere, which is false for the provider of DriverNone.
func Enabled(p Provider) bool {
	_, none := p.(noneProvider)
	return p != nil && !none
}

// noneProvider is the provider of DriverNone. It has no destination, so there is nothing to test,
// and an upload is a programming error the caller should have prevented by checking Enabled.
type noneProvider struct{}

// TestConnection succeeds, since there is nothing to reach.
func (noneProvider) TestConnection(context.Context) error {
	return nil
}

// Upload fails with ErrUnsupported.
func (noneProvider) Upload(context.Context, string, []File) (string, error) {
	return "", ErrUnsupported
}
