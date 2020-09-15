// Package version holds the global version information for the application.
package version

// Most of these variables are set by the Go linker; do not edit by hand
var (
	// internal identifier for logs and such; lowercase, no spaces
	Id = "urn"

	// display name for application; can have case and spaces
	Name = "URN persistent identifier service"

	// commit tag
	Tag = ""

	// commit hash
	Hash = ""

	// commit branch
	Branch = ""

	// source repository URL
	Repo = ""

	// version number; really just Tag without leading 'v'
	Version = func() string {
		if Tag != "" {
			return Tag[1:]
		}
		return "0.0.0"
	}()
)
