package main

import (
	"context"

	"github.com/wozz/modpox"
)

func main() {
	s := modpox.NewServer()
	s.Start()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	<-ctx.Done()
}
