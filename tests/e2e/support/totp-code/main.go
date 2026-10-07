package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
)

func main() {
	secret, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail(err)
	}
	code, err := totp.GenerateCode(strings.TrimSpace(string(secret)), time.Now())
	if err != nil {
		fail(err)
	}
	_, _ = fmt.Fprintln(os.Stdout, code)
}

func fail(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
