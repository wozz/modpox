package upstream

type Upstream interface {
	Get(string) ([]byte, int, error)
}
