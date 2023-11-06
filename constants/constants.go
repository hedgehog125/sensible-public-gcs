package constants

const NORMAL_PERMISSION = 0700
const GCS_KEY_PATH = "secret/gcs.json"
const RANDOM_QUERY_LENGTH = 15_000

var PROXIED_HEADERS = []string{
	"age",
	"cache-control",
	"expires",
	"last-modified",
	"etag",
	"vary",
	"connection",
	"keep-alive",
	"content-disposition",
	"content-length",
	"content-type",
	"content-encoding",
	"content-language",
	"accept-ranges",
	"content-range",
	"transfer-encoding",
	"date",
}
