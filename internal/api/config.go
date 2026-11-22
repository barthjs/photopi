package api

import (
	"net/http"
	"slices"

	"github.com/go-playground/validator/v10"

	"github.com/barthjs/photopi/internal/camera"
	"github.com/barthjs/photopi/internal/config"
	"github.com/barthjs/photopi/internal/mailer"
	"github.com/barthjs/photopi/internal/media"
	"github.com/barthjs/photopi/internal/service"
	"github.com/barthjs/photopi/internal/share"
)

// configGetHandler returns the effective application configuration without its secrets.
//
//	@Summary		Get config
//	@Description	Returns the effective application configuration with the default and the source of each setting. Secrets are returned empty (Admin only).
//	@Tags			config
//	@Produce		json
//	@Success		200	{object}	ConfigResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/config [get]
func (a *API) configGetHandler(w http.ResponseWriter, r *http.Request) {
	a.writeConfig(w, r)
}

type ConfigResponse struct {
	ConfigSettings
	// Secrets reports by path whether each secret is set, since their values never leave the backend.
	Secrets  map[string]bool `json:"secrets"`
	Defaults ConfigSettings  `json:"defaults"`
	// Sources names by path where each setting comes from.
	Sources map[string]ConfigSource `json:"sources"`
	File    ConfigFile              `json:"file"`
}

type ConfigSettings struct {
	App     AppSettings     `json:"app"`
	Log     LogSettings     `json:"log"`
	Camera  CameraSettings  `json:"camera"`
	Capture CaptureSettings `json:"capture"`
	Mail    MailSettings    `json:"mail"`
	Print   PrintSettings   `json:"print"`
	Server  ServerSettings  `json:"server"`
	Share   ShareSettings   `json:"share"`
	Storage StorageSettings `json:"storage"`
	System  SystemSettings  `json:"system"`
	Theme   ThemeSettings   `json:"theme"`
}

type ConfigSource struct {
	// Kind is the source of the value, where a flag or an environment variable overrides the config file,
	// so a change saved to the file has no effect on such a setting.
	Kind string `json:"kind" enums:"default,file,env,flag"`
	// Name is the flag, the environment variable, or the file path, empty for the default.
	Name string `json:"name"`
}

type ConfigFile struct {
	// Path is empty when a save creates a new config file.
	Path string `json:"path"`
	// Exists is false until the file at Path is created.
	Exists bool `json:"exists"`
}

type AppSettings struct {
	Name     string `json:"name" validate:"max=100"`
	Env      string `json:"env" enums:"development,staging,production"`
	Language string `json:"language" validate:"oneof=en de"`
	Secret   string `json:"secret"`
}

type LogSettings struct {
	Level      string `json:"level"`
	Channel    string `json:"channel" validate:"oneof=stdout file both"`
	Path       string `json:"path" validate:"max=1024"`
	Format     string `json:"format" validate:"oneof=json console"`
	MaxSizeMB  int    `json:"maxSizeMb" validate:"min=0"`
	MaxAgeDays int    `json:"maxAgeDays" validate:"min=-1"`
	Compress   bool   `json:"compress"`
}

type CameraSettings struct {
	Source        string `json:"source" validate:"oneof=webcam rpicam gphoto2"`
	CaptureWidth  int    `json:"captureWidth" validate:"required_if=Source rpicam,max=10000"`
	CaptureHeight int    `json:"captureHeight" validate:"required_if=Source rpicam,max=10000"`
	PreviewWidth  int    `json:"previewWidth" validate:"required_if=Source rpicam,max=4000"`
	PreviewHeight int    `json:"previewHeight" validate:"required_if=Source rpicam,max=4000"`
	PreviewFPS    int    `json:"previewFps" validate:"required_if=Source rpicam,max=120"`
	OverlayPath   string `json:"overlayPath" validate:"max=1024"`

	ExposureTime         string  `json:"exposureTime" validate:"oneof=auto 1/15 1/20 1/30 1/40 1/50 1/60 1/80 1/100 1/125 1/160 1/200 1/250 1/320 1/400 1/500"`
	ExposureMode         string  `json:"exposureMode" validate:"oneof=normal short"`
	AnalogueGain         float64 `json:"analogueGain" validate:"min=0,max=32"`
	ExposureCompensation float64 `json:"exposureCompensation" validate:"min=-10,max=10"`
	WhiteBalance         string  `json:"whiteBalance" validate:"oneof=auto incandescent tungsten fluorescent indoor daylight cloudy"`
	HorizontalFlip       bool    `json:"horizontalFlip"`
	VerticalFlip         bool    `json:"verticalFlip"`
	Rotation             int     `json:"rotation" validate:"oneof=0 180"`
	JPEGQuality          int     `json:"jpegQuality" validate:"required_if=Source rpicam,max=100"`
	AutofocusMode        string  `json:"autofocusMode" validate:"oneof=default manual auto continuous"`
	Denoise              string  `json:"denoise" validate:"oneof=auto off cdn_off cdn_fast cdn_hq"`
}

type CaptureSettings struct {
	ImagesPerRun   int    `json:"imagesPerRun" validate:"min=1,max=20"`
	FilePrefix     string `json:"filePrefix" validate:"required,max=100"`
	EventName      string `json:"eventName" validate:"required,max=100"`
	WelcomeMessage string `json:"welcomeMessage" validate:"max=70"`
	AdminEmail     string `json:"adminEmail" validate:"omitempty,email"`
	SaveOriginals  bool   `json:"saveOriginals"`
}

type PrintSettings struct {
	Enabled        bool   `json:"enabled"`
	PrinterName    string `json:"printerName" validate:"required_if=Enabled true"`
	Layout         string `json:"layout" validate:"required_if=Enabled true,max=60"`
	SubmissionMode string `json:"submissionMode" validate:"oneof=raw managed"`
}

type MailSettings struct {
	Driver      string `json:"driver" validate:"oneof=log smtp"`
	Host        string `json:"host" validate:"required_if=Driver smtp"`
	Port        int    `json:"port" validate:"min=1,max=65535"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromAddress string `json:"fromAddress" validate:"omitempty,email"`
	FromName    string `json:"fromName"`
	ReplyTo     string `json:"replyTo" validate:"omitempty,email"`
	// MessageIDDomain mirrors the rule of the mail section of the config package: a bare domain name.
	MessageIDDomain string `json:"messageIdDomain" validate:"omitempty,excludesall=@<> "`
}

// ServerSettings checks the trusted proxies and the CORS origins as the config package reads them,
// so a value it would reject is a 422 at its field.
type ServerSettings struct {
	Host                 string   `json:"host"`
	Port                 int      `json:"port" validate:"min=1,max=65535"`
	TrustedProxies       string   `json:"trustedProxies" validate:"max=1024,networks"`
	CORSAllowedOrigins   []string `json:"corsAllowedOrigins" validate:"max=50,dive,max=255,origin"`
	CORSAllowCredentials bool     `json:"corsAllowCredentials"`
	SSLEnabled           bool     `json:"sslEnabled"`
	SSLCertFile          string   `json:"sslCertFile" validate:"required_with=SSLKeyFile,max=1024"`
	SSLKeyFile           string   `json:"sslKeyFile" validate:"required_with=SSLCertFile,max=1024"`
}

// init registers the rules of the configuration DTOs with the validator.
func init() {
	// A required_if tag only sees sibling fields, but the fields of StorageSettings.S3
	// and CloudShareSettings.Nextcloud/Self depend on the driver one level up.
	// A struct-level validation checks across that boundary,
	// so a missing field is reported as a field error of the 422 like any other.
	v.RegisterStructValidation(validateStorageSettings, StorageSettings{})
	v.RegisterStructValidation(validateCloudShareSettings, CloudShareSettings{})

	// The validator has no rule for a comma-separated list of networks, which the network settings are written as
	if err := v.RegisterValidation("networks", validateNetworks); err != nil {
		panic(err)
	}

	// The origins and the credentials of CORS follow the rules of the config package, see ServerSettings
	if err := v.RegisterValidation("origin", validateOrigin); err != nil {
		panic(err)
	}
	v.RegisterStructValidation(validateServerSettings, ServerSettings{})
}

// validateNetworks reports whether a text lists networks as the config package reads them. It is the "networks" tag,
// so a list the config package would reject is a 422 at its field instead of a 400 for the whole configuration.
func validateNetworks(fl validator.FieldLevel) bool {
	_, err := config.ParseNetworks(fl.Field().String())
	return err == nil
}

// validateOrigin reports whether a text lists allowed CORS origins as the config package reads them.
// It is the "origin" tag.
func validateOrigin(fl validator.FieldLevel) bool {
	_, err := config.ParseOriginList(fl.Field().String())
	return err == nil
}

// validateServerSettings rejects cookies for every origin, since a browser refuses credentials for the wildcard.
// The rule spans two fields, so a struct-level validation reports it at the switch of the credentials.
func validateServerSettings(sl validator.StructLevel) {
	req := sl.Current().Interface().(ServerSettings) //nolint:errcheck // The validation is registered for this type only.
	if !req.CORSAllowCredentials {
		return
	}
	for _, value := range req.CORSAllowedOrigins {
		// An invalid entry is reported by the origin tag, and the wildcard is checked as the config package reads it
		origins, _ := config.ParseOriginList(value)
		if slices.Contains(origins, "*") {
			sl.ReportError(req.CORSAllowCredentials, "corsAllowCredentials", "corsAllowCredentials", "explicit_origins", "")
			return
		}
	}
}

type EmailShareSettings struct {
	Enabled bool   `json:"enabled"`
	Subject string `json:"subject" validate:"max=255"`
}

type NextcloudSettings struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Folder   string `json:"folder" validate:"max=1024"`
}

type SelfShareSettings struct {
	BaseURL string `json:"baseUrl"`
}

type CloudShareSettings struct {
	Driver    string            `json:"driver" validate:"omitempty,oneof=nextcloud self"`
	Nextcloud NextcloudSettings `json:"nextcloud"`
	Self      SelfShareSettings `json:"self"`
}

// validateCloudShareSettings requires the fields the selected cloud driver needs,
// mirroring share.New's own requirements and the share section of the config package.
func validateCloudShareSettings(sl validator.StructLevel) {
	req := sl.Current().Interface().(CloudShareSettings) //nolint:errcheck // The validation is registered for this type only.

	switch share.Driver(req.Driver) {
	case share.DriverNextcloud:
		if req.Nextcloud.URL == "" {
			sl.ReportError(req.Nextcloud.URL, "nextcloud.url", "nextcloud.url", "required_if", "")
		}
		if req.Nextcloud.Username == "" {
			sl.ReportError(req.Nextcloud.Username, "nextcloud.username", "nextcloud.username", "required_if", "")
		}
	case share.DriverSelf:
		if req.Self.BaseURL == "" {
			sl.ReportError(req.Self.BaseURL, "self.baseUrl", "self.baseUrl", "required_if", "")
		}
	case share.DriverNone:
		// Without a driver, no settings are required
	}
}

type ShareSettings struct {
	Email EmailShareSettings `json:"email"`
	Cloud CloudShareSettings `json:"cloud"`
}

type S3Settings struct {
	Endpoint     string `json:"endpoint"`
	Region       string `json:"region"`
	Bucket       string `json:"bucket"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	UsePathStyle bool   `json:"usePathStyle"`
}

type StorageSettings struct {
	Driver string     `json:"driver" validate:"oneof=local s3"`
	Path   string     `json:"path" validate:"max=1024"`
	S3     S3Settings `json:"s3"`
}

// validateStorageSettings requires the S3 fields media.New needs once the s3 driver is selected,
// mirroring the storage section of the config package. The secret key may be empty,
// since the admin interface never receives it, the config service checks it after keeping the current one.
func validateStorageSettings(sl validator.StructLevel) {
	req := sl.Current().Interface().(StorageSettings) //nolint:errcheck // The validation is registered for this type only.
	if req.Driver != string(media.DriverS3) {
		return
	}

	if req.S3.Endpoint == "" {
		sl.ReportError(req.S3.Endpoint, "s3.endpoint", "s3.endpoint", "required_if", "")
	}
	if req.S3.Bucket == "" {
		sl.ReportError(req.S3.Bucket, "s3.bucket", "s3.bucket", "required_if", "")
	}
	if req.S3.AccessKey == "" {
		sl.ReportError(req.S3.AccessKey, "s3.accessKey", "s3.accessKey", "required_if", "")
	}
}

type SystemSettings struct {
	ServerRestartCommand string `json:"serverRestartCommand" validate:"max=1024"`
	ServerStopCommand    string `json:"serverStopCommand" validate:"max=1024"`
	KioskRestartCommand  string `json:"kioskRestartCommand" validate:"max=1024"`
	KioskStopCommand     string `json:"kioskStopCommand" validate:"max=1024"`
	AllRestartCommand    string `json:"allRestartCommand" validate:"max=1024"`
	AllStopCommand       string `json:"allStopCommand" validate:"max=1024"`
}

type ThemeSettings struct {
	Primary   string `json:"primary" validate:"omitempty,hexcolor"`
	Secondary string `json:"secondary" validate:"omitempty,hexcolor"`
	Success   string `json:"success" validate:"omitempty,hexcolor"`
	Info      string `json:"info" validate:"omitempty,hexcolor"`
	Warning   string `json:"warning" validate:"omitempty,hexcolor"`
	Error     string `json:"error" validate:"omitempty,hexcolor"`
	Neutral   string `json:"neutral" validate:"omitempty,hexcolor"`
}

type ConfigSaveRequest struct {
	App     AppSettings     `json:"app"`
	Log     LogSettings     `json:"log"`
	Camera  CameraSettings  `json:"camera"`
	Capture CaptureSettings `json:"capture"`
	Mail    MailSettings    `json:"mail"`
	Print   PrintSettings   `json:"print"`
	Server  ServerSettings  `json:"server"`
	Share   ShareSettings   `json:"share"`
	Storage StorageSettings `json:"storage"`
	System  SystemSettings  `json:"system"`
	Theme   ThemeSettings   `json:"theme"`
	// ClearSecrets lists the JSON paths of secrets to remove. A secret sent empty otherwise keeps its value.
	ClearSecrets []string `json:"clearSecrets" validate:"max=50,dive,max=255"`
}

// configSaveHandler validates and writes the application configuration to its config file,
// which takes effect after a restart, since every service reads its configuration once at startup.
//
//	@Summary		Save config
//	@Description	Validates the submitted configuration and writes it to the config file. Does not apply to the running process, a restart is required (Admin only).
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ConfigSaveRequest	true	"Application configuration"
//	@Success		200		{object}	ConfigResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/config [put]
func (a *API) configSaveHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[ConfigSaveRequest](w, r)
	if !ok {
		return
	}

	if err := a.configService.Save(req.toConfig(), req.ClearSecrets); err != nil {
		a.respondError(w, r, err)
		return
	}

	a.writeConfig(w, r)
}

// writeConfig responds with the current configuration without its secrets.
func (a *API) writeConfig(w http.ResponseWriter, r *http.Request) {
	view, err := a.configService.View()
	if err != nil {
		a.respondError(w, r, err)
		return
	}
	a.writeJSON(w, r, http.StatusOK, toConfigResponse(view), nil)
}

// toConfigResponse maps the configuration view to the response.
func toConfigResponse(view service.ConfigView) ConfigResponse {
	sources := make(map[string]ConfigSource, len(view.Sources))
	for path, source := range view.Sources {
		sources[path] = ConfigSource{
			Kind: string(source.Kind),
			Name: source.Name,
		}
	}

	return ConfigResponse{
		ConfigSettings: toConfigSettings(view.Config),
		Secrets:        view.Secrets,
		Defaults:       toConfigSettings(view.Defaults),
		Sources:        sources,
		File: ConfigFile{
			Path:   view.File.Path,
			Exists: view.File.Exists,
		},
	}
}

// toConfigSettings maps a configuration to its settings.
func toConfigSettings(cfg config.Config) ConfigSettings {
	return ConfigSettings{
		App: AppSettings{
			Name:     cfg.App.Name,
			Env:      string(cfg.App.Env),
			Language: string(cfg.App.Language),
			Secret:   cfg.App.Secret,
		},
		Log: LogSettings{
			Level:      cfg.Log.RawLevel,
			Channel:    string(cfg.Log.Channel),
			Path:       cfg.Log.Path,
			Format:     string(cfg.Log.Format),
			MaxSizeMB:  cfg.Log.MaxSizeMB,
			MaxAgeDays: cfg.Log.MaxAgeDays,
			Compress:   cfg.Log.Compress,
		},
		Camera: CameraSettings{
			Source:        string(cfg.Camera.Source),
			CaptureWidth:  cfg.Camera.CaptureWidth,
			CaptureHeight: cfg.Camera.CaptureHeight,
			PreviewWidth:  cfg.Camera.PreviewWidth,
			PreviewHeight: cfg.Camera.PreviewHeight,
			PreviewFPS:    cfg.Camera.PreviewFPS,
			OverlayPath:   cfg.Camera.OverlayPath,

			ExposureTime:         string(cfg.Camera.ExposureTime),
			ExposureMode:         string(cfg.Camera.ExposureMode),
			AnalogueGain:         cfg.Camera.AnalogueGain,
			ExposureCompensation: cfg.Camera.ExposureCompensation,
			WhiteBalance:         string(cfg.Camera.WhiteBalance),
			HorizontalFlip:       cfg.Camera.HorizontalFlip,
			VerticalFlip:         cfg.Camera.VerticalFlip,
			Rotation:             cfg.Camera.Rotation,
			JPEGQuality:          cfg.Camera.JPEGQuality,
			AutofocusMode:        string(cfg.Camera.AutofocusMode),
			Denoise:              string(cfg.Camera.Denoise),
		},
		Capture: CaptureSettings{
			ImagesPerRun:   cfg.Capture.ImagesPerRun,
			FilePrefix:     cfg.Capture.FilePrefix,
			EventName:      cfg.Capture.EventName,
			WelcomeMessage: cfg.Capture.WelcomeMessage,
			AdminEmail:     cfg.Capture.AdminEmail,
			SaveOriginals:  cfg.Capture.SaveOriginals,
		},
		Mail: MailSettings{
			Driver:          string(cfg.Mail.Driver),
			Host:            cfg.Mail.Host,
			Port:            cfg.Mail.Port,
			Username:        cfg.Mail.Username,
			Password:        cfg.Mail.Password,
			FromAddress:     cfg.Mail.FromAddress,
			FromName:        cfg.Mail.FromName,
			ReplyTo:         cfg.Mail.ReplyTo,
			MessageIDDomain: cfg.Mail.MessageIDDomain,
		},
		Print: PrintSettings{
			Enabled:        cfg.Print.Enabled,
			PrinterName:    cfg.Print.PrinterName,
			Layout:         cfg.Print.Layout,
			SubmissionMode: cfg.Print.SubmissionMode,
		},
		Server: ServerSettings{
			Host:                 cfg.Server.Host,
			Port:                 cfg.Server.Port,
			TrustedProxies:       cfg.Server.RawTrustedProxies,
			CORSAllowedOrigins:   cfg.Server.CORSAllowedOrigins,
			CORSAllowCredentials: cfg.Server.CORSAllowCredentials,
			SSLEnabled:           cfg.Server.SSLEnabled,
			SSLCertFile:          cfg.Server.SSLCertFile,
			SSLKeyFile:           cfg.Server.SSLKeyFile,
		},
		Share: ShareSettings{
			Email: EmailShareSettings{
				Enabled: cfg.Share.Email.Enabled,
				Subject: cfg.Share.Email.Subject,
			},
			Cloud: CloudShareSettings{
				Driver: string(cfg.Share.Cloud.Driver),
				Nextcloud: NextcloudSettings{
					URL:      cfg.Share.Cloud.Nextcloud.URL,
					Username: cfg.Share.Cloud.Nextcloud.Username,
					Password: cfg.Share.Cloud.Nextcloud.Password,
					Folder:   cfg.Share.Cloud.Nextcloud.Folder,
				},
				Self: SelfShareSettings{
					BaseURL: cfg.Share.Cloud.Self.BaseURL,
				},
			},
		},
		Storage: StorageSettings{
			Driver: string(cfg.Storage.Driver),
			Path:   cfg.Storage.Path,
			S3: S3Settings{
				Endpoint:     cfg.Storage.S3.Endpoint,
				Region:       cfg.Storage.S3.Region,
				Bucket:       cfg.Storage.S3.Bucket,
				AccessKey:    cfg.Storage.S3.AccessKey,
				SecretKey:    cfg.Storage.S3.SecretKey,
				UsePathStyle: cfg.Storage.S3.UsePathStyle,
			},
		},
		System: SystemSettings{
			ServerRestartCommand: cfg.System.ServerRestartCommand,
			ServerStopCommand:    cfg.System.ServerStopCommand,
			KioskRestartCommand:  cfg.System.KioskRestartCommand,
			KioskStopCommand:     cfg.System.KioskStopCommand,
			AllRestartCommand:    cfg.System.AllRestartCommand,
			AllStopCommand:       cfg.System.AllStopCommand,
		},
		Theme: toThemeSettings(cfg.Theme),
	}
}

// toThemeSettings maps the theme of the configuration to its settings.
func toThemeSettings(theme config.ThemeConfig) ThemeSettings {
	return ThemeSettings{
		Primary:   theme.Primary,
		Secondary: theme.Secondary,
		Success:   theme.Success,
		Info:      theme.Info,
		Warning:   theme.Warning,
		Error:     theme.Error,
		Neutral:   theme.Neutral,
	}
}

// toConfig maps the request onto a configuration.
func (r ConfigSaveRequest) toConfig() config.Config {
	return config.Config{
		App: config.AppConfig{
			Name:     r.App.Name,
			Env:      config.Environment(r.App.Env),
			Language: config.Language(r.App.Language),
			Secret:   r.App.Secret,
		},
		Log: config.LogConfig{
			RawLevel:   r.Log.Level,
			Channel:    config.LogChannel(r.Log.Channel),
			Path:       r.Log.Path,
			Format:     config.LogFormat(r.Log.Format),
			MaxSizeMB:  r.Log.MaxSizeMB,
			MaxAgeDays: r.Log.MaxAgeDays,
			Compress:   r.Log.Compress,
		},
		Camera: config.CameraConfig{
			Source:        camera.Source(r.Camera.Source),
			CaptureWidth:  r.Camera.CaptureWidth,
			CaptureHeight: r.Camera.CaptureHeight,
			PreviewWidth:  r.Camera.PreviewWidth,
			PreviewHeight: r.Camera.PreviewHeight,
			PreviewFPS:    r.Camera.PreviewFPS,
			OverlayPath:   r.Camera.OverlayPath,

			ExposureTime:         camera.ExposureTime(r.Camera.ExposureTime),
			ExposureMode:         camera.ExposureMode(r.Camera.ExposureMode),
			AnalogueGain:         r.Camera.AnalogueGain,
			ExposureCompensation: r.Camera.ExposureCompensation,
			WhiteBalance:         camera.WhiteBalance(r.Camera.WhiteBalance),
			HorizontalFlip:       r.Camera.HorizontalFlip,
			VerticalFlip:         r.Camera.VerticalFlip,
			Rotation:             r.Camera.Rotation,
			JPEGQuality:          r.Camera.JPEGQuality,
			AutofocusMode:        camera.AutofocusMode(r.Camera.AutofocusMode),
			Denoise:              camera.Denoise(r.Camera.Denoise),
		},
		Capture: config.CaptureConfig{
			ImagesPerRun:   r.Capture.ImagesPerRun,
			FilePrefix:     r.Capture.FilePrefix,
			EventName:      r.Capture.EventName,
			WelcomeMessage: r.Capture.WelcomeMessage,
			AdminEmail:     r.Capture.AdminEmail,
			SaveOriginals:  r.Capture.SaveOriginals,
		},
		Print: config.PrintConfig{
			Enabled:        r.Print.Enabled,
			PrinterName:    r.Print.PrinterName,
			Layout:         r.Print.Layout,
			SubmissionMode: r.Print.SubmissionMode,
		},
		Mail: config.MailConfig{
			Driver:          mailer.Driver(r.Mail.Driver),
			Host:            r.Mail.Host,
			Port:            r.Mail.Port,
			Username:        r.Mail.Username,
			Password:        r.Mail.Password,
			FromAddress:     r.Mail.FromAddress,
			FromName:        r.Mail.FromName,
			ReplyTo:         r.Mail.ReplyTo,
			MessageIDDomain: r.Mail.MessageIDDomain,
		},
		Server: config.ServerConfig{
			Host:                 r.Server.Host,
			Port:                 r.Server.Port,
			RawTrustedProxies:    r.Server.TrustedProxies,
			CORSAllowedOrigins:   r.Server.CORSAllowedOrigins,
			CORSAllowCredentials: r.Server.CORSAllowCredentials,
			SSLEnabled:           r.Server.SSLEnabled,
			SSLCertFile:          r.Server.SSLCertFile,
			SSLKeyFile:           r.Server.SSLKeyFile,
		},
		Share: config.ShareConfig{
			Email: config.EmailShareConfig{
				Enabled: r.Share.Email.Enabled,
				Subject: r.Share.Email.Subject,
			},
			Cloud: r.Share.Cloud.toConfig(),
		},
		Storage: r.Storage.toConfig(),
		System: config.SystemConfig{
			ServerRestartCommand: r.System.ServerRestartCommand,
			ServerStopCommand:    r.System.ServerStopCommand,
			KioskRestartCommand:  r.System.KioskRestartCommand,
			KioskStopCommand:     r.System.KioskStopCommand,
			AllRestartCommand:    r.System.AllRestartCommand,
			AllStopCommand:       r.System.AllStopCommand,
		},
		Theme: config.ThemeConfig{
			Primary:   r.Theme.Primary,
			Secondary: r.Theme.Secondary,
			Success:   r.Theme.Success,
			Info:      r.Theme.Info,
			Warning:   r.Theme.Warning,
			Error:     r.Theme.Error,
			Neutral:   r.Theme.Neutral,
		},
	}
}

// configStorageTestHandler verifies that the submitted storage configuration is reachable without saving it.
//
//	@Summary		Test storage config
//	@Description	Builds the submitted storage configuration and verifies it is reachable, without saving it (Admin only).
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			request	body		StorageSettings	true	"Storage configuration to test"
//	@Success		200		{object}	ConfigTestResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Security		SessionAuth
//	@Router			/config/storage/test [post]
func (a *API) configStorageTestHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[StorageSettings](w, r)
	if !ok {
		return
	}

	a.writeTestResult(w, r, a.configService.TestStorage(r.Context(), req.toConfig()))
}

// ConfigTestResponse reports a failed check with OK false rather than an HTTP error status,
// since an unreachable destination is an expected outcome of the check.
type ConfigTestResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// toConfig maps the storage settings to the configuration.
func (r StorageSettings) toConfig() config.StorageConfig {
	return config.StorageConfig{
		Driver: media.Driver(r.Driver),
		Path:   r.Path,
		S3: config.S3Config{
			Endpoint:     r.S3.Endpoint,
			Region:       r.S3.Region,
			Bucket:       r.S3.Bucket,
			AccessKey:    r.S3.AccessKey,
			SecretKey:    r.S3.SecretKey,
			UsePathStyle: r.S3.UsePathStyle,
		},
	}
}

// configShareTestHandler verifies that the submitted cloud share configuration is reachable without saving it.
//
//	@Summary		Test cloud share config
//	@Description	Builds the submitted cloud share configuration and verifies it is reachable, without saving it (Admin only).
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CloudShareSettings	true	"Cloud share configuration to test"
//	@Success		200		{object}	ConfigTestResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		422		{object}	ValidationErrorResponse
//	@Security		SessionAuth
//	@Router			/config/share/test [post]
func (a *API) configShareTestHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := a.decodeJSON[CloudShareSettings](w, r)
	if !ok {
		return
	}

	a.writeTestResult(w, r, a.configService.TestShare(r.Context(), req.toConfig()))
}

// toConfig maps the cloud share settings to the configuration.
func (r CloudShareSettings) toConfig() config.CloudShareConfig {
	return config.CloudShareConfig{
		Driver: share.Driver(r.Driver),
		Nextcloud: config.NextcloudConfig{
			URL:      r.Nextcloud.URL,
			Username: r.Nextcloud.Username,
			Password: r.Nextcloud.Password,
			Folder:   r.Nextcloud.Folder,
		},
		Self: config.SelfShareConfig{
			BaseURL: r.Self.BaseURL,
		},
	}
}

// writeTestResult reports the outcome of a connection test. A failed test is a result to show, not a failed request.
func (a *API) writeTestResult(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		a.writeJSON(w, r, http.StatusOK, ConfigTestResponse{
			Message: err.Error(),
		}, nil)
		return
	}
	a.writeJSON(w, r, http.StatusOK, ConfigTestResponse{
		OK: true,
	}, nil)
}
