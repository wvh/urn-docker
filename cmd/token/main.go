package main

import (
	"fmt"
	"os"

	"github.com/wvh/urn/pkg/token"
)

func main() {
	fmt.Println("secmsg:", os.Args)

	svc, err := token.NewTokenService([]byte("12345678901234567890123456789012"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	_ = svc
}
