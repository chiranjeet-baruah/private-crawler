package types

import "fmt"

// Operation constants data for operation
// reference https://webservices.amazon.com/paapi5/documentation/common-request-parameters.html
type Operation string

// Operation types
const (
	GetBrowseNodes Operation = "GetBrowseNodes"
	GetItems       Operation = "GetItems"
	GetVariations  Operation = "GetVariations"
	SearchItems    Operation = "SearchItems"
)

var operationPathMap = map[Operation]string{
	GetBrowseNodes: "/paapi5/getbrowsenodes",
	GetItems:       "/paapi5/getitems",
	GetVariations:  "/paapi5/getvariations",
	SearchItems:    "/paapi5/searchitems",
}

// GetTarget gives target for given operation
// reference: https://webservices.amazon.com/paapi5/documentation/common-request-parameters.html#target
func (o Operation) GetTarget() string {
	return fmt.Sprintf("com.amazon.paapi5.v1.ProductAdvertisingAPIv1.%v", o)
}

// GetPath for Endpoint
func (o Operation) GetPath() string {
	return operationPathMap[o]
}
