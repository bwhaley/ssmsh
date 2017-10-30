package cmd

import (
	"fmt"
	"os"
)

// http://tldp.org/LDP/abs/html/exitcodes.html
// https://www.ibm.com/support/knowledgecenter/en/SS2L6K_6.0.4/com.ibm.team.scm.doc/topics/r_scm_cli_retcodes.html
const (
	ExitSuccess = iota
	ExitError
	ExitBadConnection
	ExitInvalidInput
	ExitBadFeature
	ExitInterrupted
	ExitBadArgs = 128
)

func ExitWithError(code int, err error) {
	if err != nil {
		// fmt.Fprintln(os.Stdout, "Error: ", err)
		fmt.Printf("Error: %s \n", err.Error())
	}

	os.Exit(code)
}
