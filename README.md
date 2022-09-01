# go-crawler

Crawl pipeline to coordinate web crawl and content extraction to extract structured product data from any Ecommerce web page

## Dependencies

| DB                     | Name                                                          | Description                                             |
| ---------------------- | ------------------------------------------------------------- | ------------------------------------------------------- |
| Proxy Cloud            | prod_proxy_router_spl-0.semantics3.com:4000                   | Proxy Cloud to fetch pages                              |
| Content Extraction RPC | prod_rd_rabbitmq_spl-0.semantics3.com:5672                    | Content Extraction RPC                                  |
| Translation RPC        | prod_rd_rabbitmq_spl-0.semantics3.com:5672(rd-translate-prod) | Language Translation RPC                                |
| Redis                  | prod_wrapper_spl-0.semantics3.com:6379                        | Stores wrappers/sitedetails for crawling                |
| Redis                  | prod_rdstore_spl-0.semantics3.com:6379                        | Stores site statuses (ACTIVE/PAUSE_BROKEN_WRAPPER)      |
| Postgres               | skus-db.semantics3.com:5672                                   | Stores raw data of products. Used for translation cache |
| Job Server             | prod_jobserver_rd_spl-0.semantics3.com:3130                   | Job server REST service to serve jobs                   |

## Command Line Options

| Option      | Type    | Description                                                                    |
| ----------- | ------- | ------------------------------------------------------------------------------ |
| `env`       | string  | Determines the configuration file to load from `config/` Defaults to `staging` |
| `prof`      | boolean | Whether to run `pprof` on the server for debugging. Defaults to `false`        |
|             |
| `job`       | boolean | Run the crawler as a Job Server worker                                         |
| `job-type`  | string  | Crawl pipeline to use. Defaults to `recrawl`                                   |
|             |
| `rest`      | boolean | Run the crawler as a web service. Listens on port `4310`                       |
|             |
| `test`      | boolean | Crawl a single URL                                                             |
| `url`       | string  | URL to crawl with the `--test` option                                          |
|             |         |
| `test-file` | boolean | Test a batch of URLs                                                           |
| `file`      | string  | File to read URLs to crawl with the `--test-file` option                       |

## Usage

### Run a crawl in batch mode

```shell
$ go-crawler --env staging \
        --job-type recrawl \
        --test-file \
        --file ~/inkstation.com.au.urls
```

### Run a crawl on a single URL

```
$ go-crawler --test \
        --url https://kith.com/collections/y-3-apparel/products/y-3-ft-crewneck-black
```

### Start crawler as a web service and make a request

```bash
$ go-crawler --env production-realtime --rest
```

```bash
$ echo '{
  "job_details": {
    "job_type": "recrawl"
  },
  "job_params": {},
  "tasks": {
    "https://kith.com/collections/y-3-apparel/products/y-3-ft-crewneck-black": {
      "priority": 101,
      "linkType": "content"
     }
  }
}' | http POST http://localhost:4310/crawl/url/simple
```

#### Output schema

```javascript
{
  "https://kith.com/collections/kith-apparel-women//products/kith-women-glitter-logo-sweatpant-heather-grey": {
    "url": "https://kith.com/collections/kith-apparel-women//products/kith-women-glitter-logo-sweatpant-heather-grey",
    "failuremessage": null,
    "failuretype": null,
    "status": 1,
    "validation": {
      "warn": [
        "[images_variations.0] additional field present 'https://cdn.shopify.com/s/files/1/0094/2252/products/KHW3132-100-10.progressive.jpg'",
        "[images_variations.1] additional field present 'https://cdn.shopify.com/s/files/1/0094/2252/products/KHW6093-103B.progressive.jpg'",
        "[images_variations.2] additional field present 'https://cdn.shopify.com/s/files/1/0094/2252/products/KHW6093-103A.progressive.jpg'",
        "[name_firstkeyword] additional field present 'kith'",
        "[variation_fields.size] additional field present 'S'"
      ]
    },
    "webResponse": {
      "timeTaken": 0.210872945,
      "success": true,
      "response_size": 0,
      "redirect": "https://kith.com/products/kith-women-glitter-logo-sweatpant-heather-grey",
      "from_cache": true,
      "url": "https://kith.com/collections/kith-apparel-women//products/kith-women-glitter-logo-sweatpant-heather-grey",
      "content": "...",
      "time": 1549449923,
      "response_headers": {

      },
      "status": 200,
      "attempts": 1,
      "screenshot_path": null,
      "cookie": "_shopify_y=91cc07b8-832e-42be-ae14-15fc6541baca;cart_currency=USD;secure_customer_sig=;_shopify_country=United+States;cart_sig=;_landing_page=%2Fproducts%2Fkith-women-glitter-logo-sweatpant-heather-grey;_orig_referrer=https%3A%2F%2Fkith.com%2Fcollections%2Fkith-apparel-women%2F%2Fproducts%2Fkith-women-glitter-logo-sweatpant-heather-grey;"
    },
    "rdstore_data": {
      "value": {
        "recrawl_frequency": "RF3",
        "crawl_updated": "1549439536",
        "products": [
          "..."
        ]
      }
    },
    "domainInfo": {
      "site_status": "ACTIVE",
      "wrapper_id": "5bf3940992eff3010f46c369",
      "sitedetail": {
        "...": "..."
      },
      "isProductUrl": true,
      "canonicalUrl": "https://kith.com/collections/kith-apparel-women//products/kith-women-glitter-logo-sweatpant-heather-grey",
      "parent_sku": "kith_kith-women-glitter-logo-sweatpant-heather-grey""isCssModelTrained": false,
      "isSearchUrl": false,
      "get_category_info": false,
      "domainName": "kith.com"
    },
    "product_metrics": {
      "job_type": "testwrapper",
      "retry_count": 0,
      "extraction": 0.368071633,
      "site": "kith.com",
      "url_count": 0,
      "customer": "sem3",
      "latency": 0,
      "recrawl_frequency": "RF3",
      "domain_info": 0.55807054,
      "total": 2.133142992,
      "value": 1,
      "success": "true"
    },
    "data": {
      "products": [
        {
          "_reserved_init_url": "https://kith.com/collections/kith-apparel-women//products/kith-women-glitter-logo-sweatpant-heather-grey",
          "variation_id": "kith_kith-women-glitter-logo-sweatpant-heather-grey",
          "features": {
            "Style": "KHW6093-103",
            "blob": [
              "Cotton fleece fabric",
              "..."
            ]
          }
        }
      ],
      "extraction_engine": "WRAPPER",
      "extraction_time": 0.032188,
      "extraction_breakdown": {
        "products.variation_noseparatepage": 3e-06,
        "products.variations[0].availability": 0.000183,
        "products.geo_id": 4e-06,
        "products.description": 0.000531,
        "...": 0.34
      },
      "code": "",
      "overriding_webresponse_status": 200,
      "links": {
        "https://kith.com/collections/latest-kith-products-women/products/kith-cody-cooling-tights-black": {
          "parent": "category",
          "priority": "102",
          "abs_depth": 1,
          "linkType": "content",
          "ancestor": "category",
          "rel_depth": 1
        }
      },
      "__raw_extracted_data": {
        "products": [
          {
            "name": "..."
          }
        ]
      },
      "status": 1
    },
    "job_params": {
      "cache_ttl": 10800,
      "cache": 1,
      "frequency": "RF3"
    },
    "crawl_time": 1549449923,
    "ajax_failed_status_map": {

    },
    "cache_key": "CE/testwrapper/kith_com/7be40dccf156bc5e0dc90ec1ab9a85ee"
  }
}
```

### Start crawler as a Job Server worker

```bash
$ go-crawler --env staging \
        --job \
        --job-type recrawl
```

## Development Environment

```
Programming Language: Golang 1.14
Integration Test tool: Newman@4.6.0 (Command-line tester for Postman collections)
Interface: HTTP, Job, RPC service (via RabbitMQ) [can start in multiple modes]
DB: Redis, Postgres
BuildTool/DepTool: gomod
```

## Development

- Work off a feature branch forked from `master`
- Name your branch `dev-{hyphenated-feature-description}` eg. `dev-add-custom-pipeline`
- Open a PR when you are done

```bash
gvm install go1.14
gvm use go1.14
gvm use go1.14 && export GOPATH=$HOME/gocode # Update this according to your machine

# Use git clone using SSH Keys for private repos
git config --global url."git@github.com:Semantics3/sem3-go-crawl-utils".insteadOf "https://github.com/Semantics3/sem3-go-crawl-utils"
git config --global url."git@github.com:Semantics3/sem3-go-data-consumer".insteadOf "https://github.com/Semantics3/sem3-go-data-consumer"

go mod vendor
CGO_ENABLED=0 go build -mod=vendor
```

### Develop: Build for Production

```
# always cut a release from master
$ git checkout master

# change VERSION in main.go

# CircleCI Build for production (Triggered by tag)
git tag release-v1.8.12
git push --tags origin master

CGO_ENABLED=0 go build -mod=vendor

# Manual build for production (Incase if we ran into any issue with CircleCI, otherwise not needed)
docker build --build-arg USER_ID=1001 -t sem3/go-crawler .
docker rmi 511527984524.dkr.ecr.us-east-1.amazonaws.com/sem3/go-crawler:latest; docker tag sem3/go-crawler 511527984524.dkr.ecr.us-east-1.amazonaws.com/sem3/go-crawler:latest; docker push 511527984524.dkr.ecr.us-east-1.amazonaws.com/sem3/go-crawler:latest;
```

## Testing

### Testing: Setup

In dev4Infra or dev-di instance, run the following commands to get a working version of this service

```bash
cd $GOPATH/src/github.com/Semantics3/go-crawler
gvm use go1.14.9; go version
# Build go-crawler
go install github.com/Semantics3/go-crawler
AWS_REGION=us-east-1 AWS_PROFILE=dev-engineering $GOPATH/bin/go-crawler --job-type discovery_crawl --env staging --rest

# In another shell, run these commands
# Setup sem3-di-content-extraction-service
cd ~/code or regular-work-dir;
git clone git@github.com:Semantics3/sem3-di-content-extraction-service
# Should use perl 5.18.2 with threads support [NOTE!!]
perl -v
cd sem3-di-content-extraction-service
# To speed up setup, you could copy deps from
cp -r /home/srinivas/projects/crawler/sem3-di-content-extraction-service/local ./
cpanm --local-lib local --notest Mojolicious@7.94
PERL_CARTON_MIRROR=http://54.204.28.137:3000 carton install
PERL_CARTON_MIRROR=http://54.204.28.137:3000 carton update --recursedeps --sem3only

carton exec perl script/sem3_di_content_extraction_service.pl --env staging

nvm use v13.3.0
npm install -g newman@4.6.0
```

You need to have your AWS credentials setup

```
$ cat ~/.aws/credentials
[default]
aws_access_key_id=YOUR_AWS_API_KEY
aws_secret_access_key=YOUR_AWS_API_SECRET

$ cat ~/.aws/config
[profile dev-engineering]
role_arn = arn:aws:iam::511527984524:role/dev-engineering
source_profile = default
mfa_serial = arn:aws:iam::511527984524:mfa/dev-FILLUSERNAME

[profile dev-common]
role_arn = arn:aws:iam::511527984524:role/dev-common
source_profile = default
mfa_serial = arn:aws:iam::511527984524:mfa/dev-FILLUSERNAME
[default]
region = us-east-1
```

### Testing: Unit Tests for Data Sub-package

- Test case for Unsupervised site and not a product page [DIFFBOT]  has been removed, because Diffbot was extracting some number as product price even from non-product pages like wikipedia or postgres documentation links. We may have to use Diffbot URL classification API before requesting Diffbot Extraction API. Removing this test for now [5-Jan-2021]
- Usually, upon some error in WRAPPER based extraction, we fallback to other data sources. However, in realtime pipeline, if WRAPPER based extraction returns DOES_NOT_EXIST  or NOT_PRODUCT_PAGE  , then we dont want to fallback to UNSUPERVISED or other data sources, since this error code has to be passed on to customer.

#### Unit Tests: Setup

| service       | type           | address                                                                                                               | reason                                                                                                                    |
| :------------ | :------------- | --------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `RD Rabbitmq` | RabbitMQ       | prod_rd_rabbitmq_spl-0.semantics3.com:5672                                                                            | Makes RPC call to queue rd-translate-prod                                                                                 |
| `RPC Server`  | Python Service | [prod_translation_service](https://console.semantics3.com/#/clusters?q=sem3-ds-translation-service&items_per_page=10) | Makes RPC call to queue rd-translate-prod                                                                                 |
| `Skus RD`     | Postgres       | skus-db.semantics3.com:5672                                                                                           | To test if translation caching is working fine. Critical to make sure no duplicate calls are made to Google Translate API |

```bash
go test -v -count=1 github.com/Semantics3/go-crawler/data
```

#### Unit Tests: Merge
```bash
go build
go test -v merge/*
```

### Testing: End-to-End Integration Tests

#### Integration Tests: Setup

| service              | type             | address                                                  | reason                                                             |
| :------------------- | :--------------- | -------------------------------------------------------- | ------------------------------------------------------------------ |
| `RD Rabbitmq`        | RabbitMQ         | prod_rd_rabbitmq_spl-0.semantics3.com:5672               | For RPC mode                                                       |
| `Skus RD`            | Postgres         | skus-rd-v2.c97ivjugwy4b.us-east-1.rds.amazonaws.com:5672 | For workflow                                                       |
| `Content Extraction` | Perl RPC Service | localhost                                                | To extract data from HTML page crawled by end-end testing workflow |

1. Make sure you have content extraction service running: `carton exec perl script/sem3_di_content_extraction_service.pl --env staging`
2. Make sure you have go-crawler running in REST mode: `go build; AWS_REGION=us-east-1 AWS_PROFILE=dev-engineering ./go-crawler --rest --env staging`
3. Keep in mind that, `go-crawler will ask for MFA code` while not running in prod mode.
4. Keep in mind that this will not update any databases unless go-crawler is running in prod mode.
5. Execute below curl command to **test single URL**

```bash
curl -H "Content-Type: application/json" -X POST -d \
  '{
      "job_details": {
              "job_type": "discovery_crawl"
      },
      "job_params": {
              "cache": 1, "extract_variations": 1, "forcediscover": 1
      },
      "tasks": {
              "https://www.trendyol.com/poncik/kirmizi-kopekli-pati-desenli-bebek-takimi-k2100-p-32456416": {
                      "priority": 101,
                      "linkType": "content"
              }
      }
  }' http://localhost:4310/crawl/url/simple >& ~/seedFiles/trendyol1_recrawl.json
```

6. **For full-fledged end-end testing**

Add new environment files to `test/e2e/environments` as you see fit.

**Note:** The Runkit service has hourly rate limits. So, running the test suite multiple times in an hour may incorrectly cause some tests to fail.

```shell
$ newman run --environment tests/e2e/envs/development.json \
        --globals tests/e2e/globals.json \
        tests/e2e/collection.json \
        --bail
```

_Expected output_

Last few mins

```bash
↳ AJAX returned 503 with no flags enabled
  POST http://localhost:4310/crawl/url/simple [200 OK, 6.93KB, 4s]
  ✓  response is ok
  ✓  workflow must not fail

↳ AJAX returned 503 with ajax_important enabled
  POST http://localhost:4310/crawl/url/simple [200 OK, 6.89KB, 3.8s]
  ✓  response is ok
  ✓  workflow must have failed with the REALTIME_UNREACHABLE failure type

↳ AJAX returned 503 with discontinued_on_empty_response enabled
  POST http://localhost:4310/crawl/url/simple [200 OK, 7.02KB, 3.7s]
  ✓  response is ok
  ✓  workflow must have failed with the REALTIME_UNREACHABLE failure type

┌─────────────────────────┬─────────────────────┬────────────────────┐
│                         │            executed │             failed │
├─────────────────────────┼─────────────────────┼────────────────────┤
│              iterations │                   1 │                  0 │
├─────────────────────────┼─────────────────────┼────────────────────┤
│                requests │                  33 │                  0 │
├─────────────────────────┼─────────────────────┼────────────────────┤
│            test-scripts │                  33 │                  0 │
├─────────────────────────┼─────────────────────┼────────────────────┤
│      prerequest-scripts │                   0 │                  0 │
├─────────────────────────┼─────────────────────┼────────────────────┤
│              assertions │                  62 │                  0 │
├─────────────────────────┴─────────────────────┴────────────────────┤
│ total run duration: 5m 43.1s                                       │
├────────────────────────────────────────────────────────────────────┤
│ total data received: 1.22MB (approx)                               │
├────────────────────────────────────────────────────────────────────┤
│ average response time: 10.4s [min: 5ms, max: 1m 0.7s, s.d.: 17.8s] │
└────────────────────────────────────────────────────────────────────┘
```
