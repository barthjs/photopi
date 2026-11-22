package mailer

import (
	"errors"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"mime"
	"net/http"
	"regexp"
	"strings"
)

// imagesDir is the folder of the template file system that holds the images templates embed with cid.
const imagesDir = "templates/images"

// defaultContentIDDomain is the right part of a Content-ID when neither a Message-ID domain nor a sender is set.
const defaultContentIDDomain = "mailer.localhost"

var (
	// ErrImageNotFound is returned when the HTML references an inline image
	// that neither the email nor the template images provide.
	ErrImageNotFound = errors.New("mailer: inline image not found")

	// ErrNotImage is returned when an inline image is not an image.
	ErrNotImage = errors.New("mailer: inline image is not an image")

	// ErrInvalidImageName is returned when an inline image name is not a plain file name.
	ErrInvalidImageName = errors.New("mailer: invalid inline image name")
)

// imageName restricts inline image names to plain file names, so a name never reaches outside the images folder.
var imageName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// InlineImage is an image embedded in the HTML body of one email and referenced in the HTML as {{ cid "image.jpg" }}.
// It takes precedence over a template image of the same name.
type InlineImage struct {
	// Name identifies the image in the HTML. It must be a plain file name.
	Name string
	Data []byte
	// ContentType is the media type, detected from Data when empty.
	ContentType string
}

// EmbeddedImage is an inline image of a Message, ready for transport. The HTML body refers to it as "cid:" + ContentID.
type EmbeddedImage struct {
	Name        string
	ContentID   string
	ContentType string
	Data        []byte
}

// contentIDDomain returns the right part of the Content-IDs: the Message-ID domain,
// else the domain of the sender, so the IDs belong to the sending domain.
func contentIDDomain(messageIDDomain, fromAddress string) string {
	if messageIDDomain != "" {
		return messageIDDomain
	}
	if _, domain, found := strings.CutLast(fromAddress, "@"); found && domain != "" {
		return domain
	}
	return defaultContentIDDomain
}

// cidFunc returns the template function cid, which turns an image name into its "cid:" URL.
func cidFunc(domain string) func(name string) (htmltemplate.URL, error) {
	return func(name string) (htmltemplate.URL, error) {
		if !imageName.MatchString(name) {
			return "", fmt.Errorf("%w: %q", ErrInvalidImageName, name)
		}
		// The URL is typed as trusted, since html/template would otherwise replace the unknown "cid:" scheme
		return htmltemplate.URL("cid:" + name + "@" + domain), nil
	}
}

// imageResolver finds the inline images an HTML body references and loads them.
type imageResolver struct {
	domain     string
	images     fs.FS
	references *regexp.Regexp
}

// newImageResolver creates a resolver for the images of the templates, referenced with the domain in their Content-ID.
func newImageResolver(domain string, images fs.FS) *imageResolver {
	return &imageResolver{
		domain:     domain,
		images:     images,
		references: regexp.MustCompile(`cid:([A-Za-z0-9][A-Za-z0-9._-]*)@` + regexp.QuoteMeta(domain)),
	}
}

// resolve returns the images the HTML references, each once in order of first use, with the images of the email
// ahead of the template images.
func (r *imageResolver) resolve(html string, own []InlineImage) ([]EmbeddedImage, error) {
	byName := make(map[string]InlineImage, len(own))
	for _, image := range own {
		if !imageName.MatchString(image.Name) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidImageName, image.Name)
		}
		byName[image.Name] = image
	}

	var embedded []EmbeddedImage
	seen := make(map[string]bool)
	for _, match := range r.references.FindAllStringSubmatch(html, -1) {
		name := match[1]
		if seen[name] {
			continue
		}
		seen[name] = true

		image, ok := byName[name]
		if !ok {
			data, err := fs.ReadFile(r.images, imagesDir+"/"+name)
			if err != nil {
				return nil, fmt.Errorf("%w: %q", ErrImageNotFound, name)
			}
			image = InlineImage{
				Name: name,
				Data: data,
			}
		}

		contentType, err := imageContentType(image)
		if err != nil {
			return nil, err
		}
		embedded = append(embedded, EmbeddedImage{
			Name:        name,
			ContentID:   name + "@" + r.domain,
			ContentType: contentType,
			Data:        image.Data,
		})
	}
	return embedded, nil
}

// imageContentType returns the media type of image, detected from its content if not given,
// and rejects anything that is not an image.
func imageContentType(image InlineImage) (string, error) {
	contentType := image.ContentType
	if contentType == "" {
		contentType = http.DetectContentType(image.Data)
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "image/") {
		return "", fmt.Errorf("%w: %q is %s", ErrNotImage, image.Name, contentType)
	}
	return mediaType, nil
}
