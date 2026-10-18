// Package templates names the email templates. The templates lie next to this file,
// and the mailer embeds the whole folder but only reads its .gohtml files.
package templates

// Share is the email that sends the photos of a capture run.
const Share = "capture_share.gohtml"
