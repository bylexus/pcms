package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// creates an encrypted password to be used in the site.users config
func runPasswordCmd(args CmdArgs) {
	for _, pw := range args.flagSet.Args() {
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err == nil {
			fmt.Printf("%s ==> %s\n", pw, hash)
		}
	}
}
