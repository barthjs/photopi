package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/barthjs/photopi/internal/apperr"
	"github.com/barthjs/photopi/internal/crypto"
	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/mailer/templates"
	"github.com/barthjs/photopi/internal/share"
	"github.com/barthjs/photopi/internal/store"
)

const (
	// selfShareTokenPurpose domain-separates the tokens of self share links from other signatures of the same key.
	selfShareTokenPurpose = "photopi-share-run"
	// selfShareLinkTTL is how long a self share link stays valid after the run was shared.
	selfShareLinkTTL = 7 * 24 * time.Hour
	// selfShareFileURLTTL is how long the file URLs of a self share page stay valid, a page view signs them anew.
	selfShareFileURLTTL = time.Hour

	// cloudCheckInterval is how often the reachability of the cloud share is checked.
	cloudCheckInterval = 30 * time.Second
	// cloudCheckTimeout bounds one reachability check, so a booth without internet notices it quickly.
	cloudCheckTimeout = 5 * time.Second
)

// captureShareEmailData is the data the share email template renders.
type captureShareEmailData struct {
	AppName   string
	Locale    string
	EventName string
	Subject   string
}

// ShareOptions holds the settings of a ShareService.
type ShareOptions struct {
	// AppName and Language select the sender name and the language of share emails.
	AppName  string
	Language string
	// EventName names the event in share emails and is slugified into the cloud folder of the runs.
	EventName string
	// EmailEnabled allows guests to email themselves their captures, with EmailSubject as subject.
	EmailEnabled bool
	EmailSubject string
	// PrintEnabled attaches the printed collage to every share.
	PrintEnabled bool
	// SelfBaseURL enables the self share: instead of uploading, a run is shared as a signed link
	// to the public share page of this server at SelfBaseURL. Empty uses the cloud share provider.
	SelfBaseURL string
	// SelfSigningKey signs the self share links.
	SelfSigningKey []byte
}

// ShareService sends the captures of a completed run to the guest by email or as a link.
type ShareService struct {
	logger         zerolog.Logger
	options        ShareOptions
	mailer         *mailer.Mailer
	provider       share.Provider
	runs           *store.RunStore
	captureService *CaptureService
	printService   *PrintService
	eventSlug      string

	emails  *runJobs
	uploads *runJobs
	// cloudUnreachable is set while the last reachability check of the cloud share failed.
	cloudUnreachable atomic.Bool
}

// NewShareService creates a ShareService.
func NewShareService(
	logger zerolog.Logger,
	options ShareOptions,
	mailService *mailer.Mailer,
	provider share.Provider,
	runs *store.RunStore,
	captureService *CaptureService,
	printService *PrintService,
) *ShareService {
	return &ShareService{
		logger:         logger,
		options:        options,
		mailer:         mailService,
		provider:       provider,
		runs:           runs,
		captureService: captureService,
		printService:   printService,
		eventSlug:      slugify(options.EventName),

		emails:  newRunJobs(),
		uploads: newRunJobs(),
	}
}

// ListSharedFiles returns signed URLs of every file of a shared run, its captures followed by its collage pages,
// so the public share page loads them straight from the storage.
func (s *ShareService) ListSharedFiles(ctx context.Context, run int) ([]string, error) {
	paths, err := s.runFilePaths(ctx, run)
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(paths))
	for _, p := range paths {
		url, err := s.runs.SignFile(ctx, p, selfShareFileURLTTL)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}

	return urls, nil
}

// ListSharedFilePaths returns the storage paths of every file of a shared run, its captures followed by its collage pages,
// so a caller can answer an unknown run before it starts streaming them with ZipSharedFiles.
func (s *ShareService) ListSharedFilePaths(ctx context.Context, run int) ([]string, error) {
	return s.runFilePaths(ctx, run)
}

// GetEmail reports the state of the latest email of a run, for polling.
// found is false if no email was ever started for run.
func (s *ShareService) GetEmail(run int) (state JobState, found bool) {
	return s.emails.get(run)
}

// GetCloudUpload reports the state of a run's latest cloud upload, for polling.
// found is false if no upload was ever started for run.
func (s *ShareService) GetCloudUpload(run int) (state JobState, found bool) {
	return s.uploads.get(run)
}

// EmailEnabled reports whether guests are allowed to email themselves their captures.
func (s *ShareService) EmailEnabled() bool {
	return s.options.EmailEnabled
}

// CloudEnabled reports whether runs can be shared as a link, through a cloud share provider or the self share.
func (s *ShareService) CloudEnabled() bool {
	return s.selfShareEnabled() || share.Enabled(s.provider)
}

// RunHasCaptures reports whether any capture is stored for a run,
// so a bad run number fails before a slow job starts in the background.
func (s *ShareService) RunHasCaptures(ctx context.Context, run int) (bool, error) {
	paths, err := s.captureService.ListRunFiles(ctx, run)
	if err != nil {
		return false, fmt.Errorf("service: listing run %d files: %w", run, err)
	}
	return len(paths) > 0, nil
}

// BeginEmail marks the email of a run as pending and reports whether the caller starts RunEmail.
func (s *ShareService) BeginEmail(run int) bool {
	return s.emails.begin(run)
}

// RunEmail emails every capture of a run to the recipient as attachments and records the outcome for GetEmail.
// It is meant to run in the background, with ctx bounding how long it may take.
func (s *ShareService) RunEmail(ctx context.Context, run int, recipient string) error {
	err := s.sendEmail(ctx, run, recipient)
	s.emails.finish(run, jobStateOf(err, ""))
	return err
}

// sendEmail emails every capture of a run to the recipient as attachments.
func (s *ShareService) sendEmail(ctx context.Context, run int, recipient string) error {
	files, err := s.readRunFiles(ctx, run)
	if err != nil {
		return err
	}

	attachments := make([]mailer.Attachment, 0, len(files))
	for _, file := range files {
		attachments = append(attachments, mailer.Attachment{
			Filename: file.Name,
			Data:     file.Data,
		})
	}

	return s.mailer.Send(ctx, recipient, templates.Share, captureShareEmailData{
		AppName:   s.options.AppName,
		Locale:    s.options.Language,
		EventName: s.options.EventName,
		Subject:   s.options.EmailSubject,
	}, mailer.SendOptions{
		Attachments: attachments,
	})
}

// errCloudUnreachable fails an upload right away while the cloud share is known to be unreachable.
var errCloudUnreachable = errors.New("service: cloud share is unreachable")

// BeginCloudUpload marks the cloud upload of a run as pending and reports whether the caller starts RunCloudUpload.
func (s *ShareService) BeginCloudUpload(run int) bool {
	return s.uploads.begin(run)
}

// RunCloudUpload shares a run as a link and records the outcome for GetCloudUpload.
// It is meant to run in the background, with ctx bounding how long it may take.
func (s *ShareService) RunCloudUpload(ctx context.Context, run int) error {
	url, err := s.cloudShare(ctx, run)
	s.uploads.finish(run, jobStateOf(err, url))
	return err
}

// cloudShare returns the share link of a run, uploading its files first unless the self share is used.
func (s *ShareService) cloudShare(ctx context.Context, run int) (string, error) {
	if s.selfShareEnabled() {
		return s.selfShareURL(run, time.Now()), nil
	}

	if s.cloudUnreachable.Load() {
		return "", errCloudUnreachable
	}

	files, err := s.readRunFiles(ctx, run)
	if err != nil {
		return "", err
	}

	remoteFolder := path.Join(s.eventSlug, fmt.Sprintf("%06d", run))
	url, err := s.provider.Upload(ctx, remoteFolder, files)
	if err != nil {
		if isUnreachable(err) {
			s.cloudUnreachable.Store(true)
		}
		return "", err
	}

	return url, nil
}

// selfShareURL returns the link to the public share page of a run, valid for selfShareLinkTTL from now.
func (s *ShareService) selfShareURL(run int, now time.Time) string {
	expires := strconv.FormatInt(now.Add(selfShareLinkTTL).Unix(), 10)
	token := expires + "." + crypto.SignToken(s.options.SelfSigningKey, selfShareTokenPurpose+":"+expires, run)
	return fmt.Sprintf("%s/s/%s", strings.TrimRight(s.options.SelfBaseURL, "/"), token)
}

// readRunFiles lists and fully reads every file of a run, its captures followed by its collage pages.
func (s *ShareService) readRunFiles(ctx context.Context, run int) ([]share.File, error) {
	paths, err := s.runFilePaths(ctx, run)
	if err != nil {
		return nil, err
	}

	files := make([]share.File, 0, len(paths))
	for _, p := range paths {
		data, err := s.runs.GetFile(ctx, p)
		if err != nil {
			return nil, err
		}
		files = append(files, share.File{
			Name: path.Base(p),
			Data: data,
		})
	}

	return files, nil
}

// runFilePaths returns the storage paths of the captures of a run and, with printing enabled, its collage pages.
// A collage that cannot be rendered is logged and left out, so it never blocks the share.
func (s *ShareService) runFilePaths(ctx context.Context, run int) ([]string, error) {
	paths, err := s.captureService.ListRunFiles(ctx, run)
	if err != nil {
		return nil, fmt.Errorf("service: listing run %d files: %w", run, err)
	}
	if len(paths) == 0 {
		return nil, apperr.ErrNotFound
	}

	if !s.options.PrintEnabled {
		return paths, nil
	}

	collagePaths, err := s.printService.RenderCollage(ctx, run)
	if err != nil {
		s.logger.Warn().Err(err).Int("run", run).Msg("sharing run without its collage")
		return paths, nil
	}

	return append(paths, collagePaths...), nil
}

// jobStateOf returns the state a finished job records: done with url, or failed with a reason
// that tells an unreachable destination apart from every other failure.
func jobStateOf(err error, url string) JobState {
	if err == nil {
		return JobState{
			Status: JobDone,
			URL:    url,
		}
	}

	reason := JobReasonFailed
	if isUnreachable(err) {
		reason = JobReasonUnreachable
	}

	return JobState{
		Status: JobFailed,
		Reason: reason,
	}
}

// RunCloudCheck checks the reachability of the cloud share until ctx ends,
// so an upload without internet fails right away instead of after its timeout.
func (s *ShareService) RunCloudCheck(ctx context.Context) {
	if s.selfShareEnabled() || !share.Enabled(s.provider) {
		return
	}

	ticker := time.NewTicker(cloudCheckInterval)
	defer ticker.Stop()

	for {
		s.checkCloud(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// checkCloud runs one reachability check of the cloud share and records whether it is unreachable.
// Rejected credentials or an unexpected answer still mean the cloud is reachable,
// the upload then reports the actual problem.
func (s *ShareService) checkCloud(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, cloudCheckTimeout)
	defer cancel()

	err := s.provider.TestConnection(ctx)
	unreachable := err != nil && isUnreachable(err)
	if unreachable != s.cloudUnreachable.Swap(unreachable) {
		s.logger.Info().Bool("reachable", !unreachable).Msg("cloud share reachability changed")
	}
}

// isUnreachable reports whether err means the destination could not be reached over the network.
func isUnreachable(err error) bool {
	if errors.Is(err, errCloudUnreachable) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	_, ok := errors.AsType[net.Error](err)
	return ok
}

// ResolveSelfShareToken returns the run a self share token was signed for,
// ok is false if the token is invalid or expired, or the self share is disabled.
func (s *ShareService) ResolveSelfShareToken(token string, now time.Time) (run int, ok bool) {
	if !s.selfShareEnabled() {
		return 0, false
	}

	expires, signed, found := strings.Cut(token, ".")
	if !found {
		return 0, false
	}

	expiresUnix, err := strconv.ParseInt(expires, 10, 64)
	if err != nil || now.Unix() > expiresUnix {
		return 0, false
	}

	return crypto.VerifyToken(s.options.SelfSigningKey, selfShareTokenPurpose+":"+expires, signed)
}

// selfShareEnabled reports whether runs are shared through the public share page of this server.
func (s *ShareService) selfShareEnabled() bool {
	return s.options.SelfBaseURL != ""
}

// ZipSharedFiles streams the files of a shared run at paths, as listed by ListSharedFilePaths, as a zip archive to w.
func (s *ShareService) ZipSharedFiles(ctx context.Context, w io.Writer, paths []string) error {
	return s.runs.ZipFiles(ctx, w, paths)
}
