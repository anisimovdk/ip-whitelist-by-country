package ipdata

// IPProcessor defines the interface for IP data processing
type IPProcessor interface {
	GetIPListForCountry(countryCode string) ([]string, error)
}

// Ensure Processor implements IPProcessor
var _ IPProcessor = (*Processor)(nil)
