package mailer_test

// sampleData fills every field the templates of this application reference, so each renders completely.
func sampleData(locale string) map[string]any {
	return map[string]any{
		"Locale":    locale,
		"AppName":   "Sample App",
		"Subject":   "",
		"EventName": "Sample Event",
	}
}
