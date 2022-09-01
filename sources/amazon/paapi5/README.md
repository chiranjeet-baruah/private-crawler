# Amazon Product Advertising API 5.0 SDK
Go SDK for [Amazon Product Advertising API 5.0](https://webservices.amazon.com/paapi5/documentation/)

## Usage
### Using URL
```go
accessKey := "<PAAPI_ACCESS_KEY>"
secretKey := "<PAAPI_SECRET_KEY>"
associateTag := "<PAAPI_ASSOCIATE_TAG>"

client, code, err := paapi5.NewClient()
if err != nil {
    // handle error
}
// return type (*types.GetItemsResponse, error)
res, err := client.GetItemsFromURL(context.TODO(), url)
if err != nil {
    // handle error
}
```

## Example response from Amazon
Can be captured in type GetItemsResponse 
```json
{
    "ItemsResult": {
      "Items": [
        {
          "ASIN": "B07JP1QK9T",
          "BrowseNodeInfo": {
            "BrowseNodes": [
              {
                "Ancestor": {
                  "Ancestor": {
                    "Ancestor": {
                      "ContextFreeName": "Electronics",
                      "DisplayName": "Electronics",
                      "Id": "172282"
                    },
                    "ContextFreeName": "Electronics",
                    "DisplayName": "Categories",
                    "Id": "493964"
                  },
                  "ContextFreeName": "Computers & Accessories",
                  "DisplayName": "Computers & Accessories",
                  "Id": "541966"
                },
                "ContextFreeName": "Computer Monitors",
                "DisplayName": "Monitors",
                "Id": "1292115011",
                "IsRoot": false,
                "SalesRank": 691
              }
            ]
          },
          "DetailPageURL": "https://www.amazon.com/dp/B07JP1QK9T?tag=cosmopolitan-20&linkCode=ogi&th=1&psc=1",
          "Images": {
            "Primary": {
              "Large": {
                "Height": 332,
                "URL": "https://m.media-amazon.com/images/I/41GXkqKAmNL.jpg",
                "Width": 500
              },
              "Medium": {
                "Height": 106,
                "URL": "https://m.media-amazon.com/images/I/41GXkqKAmNL._SL160_.jpg",
                "Width": 160
              },
              "Small": {
                "Height": 50,
                "URL": "https://m.media-amazon.com/images/I/41GXkqKAmNL._SL75_.jpg",
                "Width": 75
              }
            },
            "Variants": [
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/41v+oLQ65zL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/41v+oLQ65zL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/41v+oLQ65zL._SL75_.jpg",
                  "Width": 75
                }
              },
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/21kseXpeszL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/21kseXpeszL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/21kseXpeszL._SL75_.jpg",
                  "Width": 75
                }
              },
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/21q02+gaVJL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/21q02+gaVJL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/21q02+gaVJL._SL75_.jpg",
                  "Width": 75
                }
              },
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/211RiGZrWbL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/211RiGZrWbL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/211RiGZrWbL._SL75_.jpg",
                  "Width": 75
                }
              },
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/21rNEpV-IOL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/21rNEpV-IOL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/21rNEpV-IOL._SL75_.jpg",
                  "Width": 75
                }
              },
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/21iPWIVyVWL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/21iPWIVyVWL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/21iPWIVyVWL._SL75_.jpg",
                  "Width": 75
                }
              },
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/311heXaNmeL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/311heXaNmeL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/311heXaNmeL._SL75_.jpg",
                  "Width": 75
                }
              },
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/21dU97ed8mL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/21dU97ed8mL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/21dU97ed8mL._SL75_.jpg",
                  "Width": 75
                }
              },
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/21zG-3A5vHL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/21zG-3A5vHL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/21zG-3A5vHL._SL75_.jpg",
                  "Width": 75
                }
              },
              {
                "Large": {
                  "Height": 332,
                  "URL": "https://m.media-amazon.com/images/I/21TQE92JEPL.jpg",
                  "Width": 500
                },
                "Medium": {
                  "Height": 106,
                  "URL": "https://m.media-amazon.com/images/I/21TQE92JEPL._SL160_.jpg",
                  "Width": 160
                },
                "Small": {
                  "Height": 50,
                  "URL": "https://m.media-amazon.com/images/I/21TQE92JEPL._SL75_.jpg",
                  "Width": 75
                }
              }
            ]
          },
          "ItemInfo": {
            "ByLineInfo": {
              "Brand": {
                "DisplayValue": "LG",
                "Label": "Brand",
                "Locale": "en_US"
              },
              "Manufacturer": {
                "DisplayValue": "LG",
                "Label": "Manufacturer",
                "Locale": "en_US"
              }
            },
            "Classifications": {
              "Binding": {
                "DisplayValue": "Personal Computers",
                "Label": "Binding",
                "Locale": "en_US"
              },
              "ProductGroup": {
                "DisplayValue": "Personal Computer",
                "Label": "ProductGroup",
                "Locale": "en_US"
              }
            },
            "ExternalIds": {
              "EANs": {
                "DisplayValues": [
                  "0719192619975"
                ],
                "Label": "EAN",
                "Locale": "en_US"
              },
              "UPCs": {
                "DisplayValues": [
                  "719192619975"
                ],
                "Label": "UPC",
                "Locale": "en_US"
              }
            },
            "Features": {
              "DisplayValues": [
                "5120 x 2160 Resolution, 60 Hz refresh rate, 5 ms (GtG) Response Time, Thunderbolt 3 / HDMI / DisplayPort 1.4 / USB Type C Inputs, Built-In Speakers, Ultra-thin bezel for slim and sleek design",
                "1200:1 (Typ) Contrast Ratio, 450 cd/m2 Brightness, 178 degree/178 degree Viewing Angles (CR≥10), 10-Bit (8bit+A-FRC), DCI-P3 98% Color Gamut (CIE1931), 0.0518 (H) x 0.1554 (V) mm Pixel Pitch",
                "Windows: Plug and play for PCs with compatible graphics cards supporting 5K2K such as the 2080ti for gamers. Use DisplayPort 1.4 or Thunderbolt 3 USB-C for full 5120 x 2160 resolution",
                "Apple: Plug and play with thunderbolt 3 with 2016 and 2017 models. 2018 MacBooks may require an update to the recent Mac OS X 10.14.2 Beta for thunderbolt to work",
                "3 Years limited Parts and Labor from LG. A DisplayPort cable, USB-B to USB-A cable, and a 2-meter-long Thunderbolt 3 / USB-C cable are included",
                "60 hertz"
              ],
              "Label": "Features",
              "Locale": "en_US"
            },
            "ManufactureInfo": {
              "ItemPartNumber": {
                "DisplayValue": "34BK95U-W",
                "Label": "PartNumber",
                "Locale": "en_US"
              },
              "Model": {
                "DisplayValue": "34BK95U-W",
                "Label": "Model",
                "Locale": "en_US"
              },
              "Warranty": {
                "DisplayValue": "3 Years",
                "Label": "Warranty",
                "Locale": "en_US"
              }
            },
            "ProductInfo": {
              "Color": {
                "DisplayValue": "Black",
                "Label": "Color",
                "Locale": "en_US"
              },
              "ItemDimensions": {
                "Height": {
                  "DisplayValue": 7.5,
                  "Label": "Height",
                  "Locale": "en_US",
                  "Unit": "Inches"
                },
                "Length": {
                  "DisplayValue": 38.7,
                  "Label": "Length",
                  "Locale": "en_US",
                  "Unit": "Inches"
                },
                "Weight": {
                  "DisplayValue": 16.5,
                  "Label": "Weight",
                  "Locale": "en_US",
                  "Unit": "Pounds"
                },
                "Width": {
                  "DisplayValue": 20.7,
                  "Label": "Width",
                  "Locale": "en_US",
                  "Unit": "Inches"
                }
              },
              "ReleaseDate": {
                "DisplayValue": "2019-02-15T00:00:01Z",
                "Label": "ReleaseDate",
                "Locale": "en_US"
              },
              "UnitCount": {
                "DisplayValue": 1,
                "Label": "NumberOfItems",
                "Locale": "en_US"
              }
            },
            "Title": {
              "DisplayValue": "LG 34BK95U-W UltraFine 34\" 21:9 5K 2K (5120 x 2160) Nano IPS LED UltraWide Monitor, 600 cd/m² HDR, Thunderbolt 3 / USB Type-C Inputs Black",
              "Label": "Title",
              "Locale": "en_US"
            }
          },
          "Offers": {
            "Listings": [
              {
                "Availability": {
                  "MaxOrderQuantity": 5,
                  "Message": "In Stock.",
                  "MinOrderQuantity": 1,
                  "Type": "Now"
                },
                "Condition": {
                  "SubCondition": {
                    "Value": "New"
                  },
                  "Value": "New"
                },
                "DeliveryInfo": {
                  "IsAmazonFulfilled": true,
                  "IsFreeShippingEligible": true,
                  "IsPrimeEligible": true
                },
                "Id": "zmveE5dbwHKdXAj8uLSL3EIXvyP9a6VokqClVN0DAeLJFniD6on6qZM31DMpadJE5MgfyXa47dGpERcFX4HtvBlV98Sx4fBIfZKxYHsMHOzW%2Fz4SJJ5EWA%3D%3D",
                "IsBuyBoxWinner": true,
                "MerchantInfo": {
                  "FeedbackCount": 387,
                  "FeedbackRating": 4.68,
                  "Id": "ATVPDKIKX0DER",
                  "Name": "Amazon.com"
                },
                "Price": {
                  "Amount": 1399.0,
                  "Currency": "USD",
                  "DisplayAmount": "$1,399.00",
                  "Savings": {
                    "Amount": 250.99,
                    "Currency": "USD",
                    "DisplayAmount": "$250.99 (15%)",
                    "Percentage": 15
                  }
                },
                "ProgramEligibility": {
                  "IsPrimeExclusive": false,
                  "IsPrimePantry": false
                },
                "SavingBasis": {
                  "Amount": 1649.99,
                  "Currency": "USD",
                  "DisplayAmount": "$1,649.99"
                },
                "ViolatesMAP": false
              }
            ],
            "Summaries": [
              {
                "Condition": {
                  "Value": "New"
                },
                "HighestPrice": {
                  "Amount": 1399.0,
                  "Currency": "USD",
                  "DisplayAmount": "$1,399.00"
                },
                "LowestPrice": {
                  "Amount": 1399.0,
                  "Currency": "USD",
                  "DisplayAmount": "$1,399.00"
                },
                "OfferCount": 2
              },
              {
                "Condition": {
                  "Value": "Used"
                },
                "HighestPrice": {
                  "Amount": 1273.74,
                  "Currency": "USD",
                  "DisplayAmount": "$1,273.74"
                },
                "LowestPrice": {
                  "Amount": 999.99,
                  "Currency": "USD",
                  "DisplayAmount": "$999.99"
                },
                "OfferCount": 3
              }
            ]
          }
        }
      ]
    }
  }
```

