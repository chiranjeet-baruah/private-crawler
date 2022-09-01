package types

// Sources defines the type for all the data sources
type Sources interface {

	// Return Type of source
	GetName() string

	// Return error code after processing
	GetErrorCode() string

	// Make http requests to download html
	Request(url string, workflow *CrawlWorkflow, pipeline Pipeline, appC *Config) (canExtract bool, code string, err error)

	// Extract data
	Extract(url string, workflow *CrawlWorkflow, pipeline Pipeline, appC *Config) (code string, err error)

	// Normalize the data to standard schema
	Normalize(workflow *CrawlWorkflow, appC *Config)
}
