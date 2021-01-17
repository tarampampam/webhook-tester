package version

// version value will be set during compilation
var version string = "undefined@undefined"

// Version returns version value.
func Version() string {
	return version
}
