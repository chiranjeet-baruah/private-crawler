package merge

import (
	"testing"

	"github.com/Semantics3/go-crawler/types"
	"github.com/stretchr/testify/suite"
)

type MergeSuite struct {
	suite.Suite
	merge Merge
}

// SetupSuite - Called once before all tests
func (suite *MergeSuite) SetupSuite() {
	// set data
	data := make(map[string][]hash)
	data["WRAPPER"] = []hash{map[string]interface{}{
		"sku":       "test_sku_wrapper",
		"listprice": "100.00",
		"offers": []hash{map[string]interface{}{
			"price":        "80.00",
			"availability": "Available",
		}},
	}}
	data["UNSUPERVISED"] = []hash{map[string]interface{}{
		"sku":                "test_sku_unsupervised",
		"listprice":          "101.00",
		"listprice_currency": "",
		"offers":             []hash{},
	}}
	data["M101"] = []hash{map[string]interface{}{
		"listprice":          "102.00",
		"listprice_currency": "USD",
	}}

	// set merge preference
	mergePreference := hash{
		"sku":                []string{"WRAPPER", "UNSUPERVISED", "M101"},
		"listprice":          "M101",
		"offers":             []string{"UNSUPERVISED", "WRAPPER"},
		"listprice_currency": []string{"WRAPPER", "UNSUPERVISED", "M101"},
		"description":        []string{"WRAPPER", "UNSUPERVISED", "M101"},
	}
	suite.merge = Merge{
		DataSources:     []string{"WRAPPER", "UNSUPERVISED", "M101"},
		Data:            data,
		MergePreference: mergePreference,
	}
}

// Test_01_generateDefaultMergePreference - tests generateDefaultMergePreference function
func (suite *MergeSuite) Test_01_generateDefaultMergePreference() {
	mergePreference := generateDefaultMergePreference(nil, suite.merge.DataSources)

	testKeys := []string{"sku", "listprice"}

	for _, key := range testKeys {
		val, ok := mergePreference[key]
		suite.Assert().Equal(true, ok)
		suite.Assert().Equal(suite.merge.DataSources, val)
	}
}

// Test_02_mergeData - tests mergeData function
func (suite *MergeSuite) Test_02_mergeData() {
	dataFromSources := make(map[string]hash)
	for key, val := range suite.merge.Data {
		dataFromSources[key] = val[0]
	}
	merged, fieldSources := mergeData(dataFromSources, suite.merge.MergePreference)
	// case: take first preference
	suite.Assert().Equal("test_sku_wrapper", merged["sku"])
	suite.Assert().Equal("WRAPPER", fieldSources["sku"])

	// case: take last preference
	suite.Assert().Equal("USD", merged["listprice_currency"])
	suite.Assert().Equal("M101", fieldSources["listprice_currency"])

	// case: merge preference value type string instead of array
	suite.Assert().Equal("102.00", merged["listprice"])
	suite.Assert().Equal("M101", fieldSources["listprice"])

	// case: when preferenced value is empty array
	suite.Assert().Equal(1, len(merged["offers"].([]hash)))
	suite.Assert().Equal("WRAPPER", fieldSources["offers"])

	// case: key not exists in data
	suite.Assert().Equal(nil, merged["description"])
	suite.Assert().Equal(nil, fieldSources["description"])
}

// Test_03_TransformProductsAndMerge - tests mergeData function
func (suite *MergeSuite) Test_03_TransformProductsAndMerge() {
	mg := &suite.merge
	workflow := &types.CrawlWorkflow{}
	workflow.Data = types.ExtractionResponse{}
	workflow.Data.Products = []hash{map[string]interface{}{}}
	mg.TransformProductsAndMerge(workflow)

	// case: should return result
	suite.Assert().Equal(1, len(workflow.Data.Products))
	data := workflow.Data.Products[0]

	// case: should have keys with value
	suite.Assert().Equal("test_sku_wrapper", data["sku"])
	suite.Assert().Equal("102.00", data["listprice"])
	suite.Assert().Equal("USD", data["listprice_currency"])
	suite.Assert().Equal(1, len(data["offers"].([]hash)))

	// case: should not have keys
	suite.Assert().Equal(nil, data["description"])
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMergeTestSuite(t *testing.T) {
	suite.Run(t, new(MergeSuite))
}
