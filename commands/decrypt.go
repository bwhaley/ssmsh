package commands

import (
	"strconv"

	"github.com/abiosoft/ishell"
)

const decryptUsage string = `
decrypt usage: decrypt
Toggles decryption of SecureString parameter values. Default is false.
`

const decryptError = "value for decrypt must be boolean"

// decrypt determines parameter decryption for SecureString values
func decrypt(c *ishell.Context) {
	if len(c.Args) == 1 {
		v, err := strconv.ParseBool(c.Args[0])
		if err != nil {
			shell.Println(decryptError)
			return
		}

		switch v {
		case true:
			ps.Decrypt = true
		case false:
			ps.Decrypt = false
		default:
			shell.Println(decryptError)
		}
	} else if len(c.Args) > 1 {
		shell.Println(decryptError)
	}
	shell.Println("Decrypt is", ps.Decrypt)
}
