package radiusctl

// Shared state set from main.go before any command runs.
var (
	client  *Client
	jsonOut bool
)

// SetClient stores the API client used by all subcommands.
func SetClient(c *Client) { client = c }

// SetJSONOut toggles JSON output mode.
func SetJSONOut(v bool) { jsonOut = v }
