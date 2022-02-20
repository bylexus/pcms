package commands

import (
	"fmt"

	"alexi.ch/pcms/src/model"
	"golang.org/x/crypto/bcrypt"
)

// creates an encrypted password to be used in the site.users config
func RunPasswordCmd(args model.CmdArgs) {
	for _, pw := range args.FlagSet.Args() {
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err == nil {
			fmt.Printf("%s ==> %s\n", pw, hash)
		}
	}
}
