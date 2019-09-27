package upstream

// Upstream is a go modules data source
// returns raw data, http status code, error
type Upstream interface {
	Get(string) ([]byte, int, error)
}
