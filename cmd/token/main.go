package main

import (
	"fmt"
	"os"

	"github.com/wvh/urn/pkg/secmsg"
)

func main() {
	fmt.Println("secmsg:", os.Args)

	svc, err := secmsg.NewMessageService([]byte("12345678901234567890123456789012"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	_ = svc
}
