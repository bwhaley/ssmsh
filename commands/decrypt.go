package commands

import (
	"github.com/abiosoft/ishell"
)

const decryptUsage string = `
decrypt usage: decrypt
Toggles decryption of SecureString parameter values. Default is false.
`

// decrypt toggles parameter decryption for SecureString values
func decrypt(c *ishell.Context) {
	ps.Decrypt = !ps.Decrypt
	shell.Println("Decrypt is", ps.Decrypt)
}
