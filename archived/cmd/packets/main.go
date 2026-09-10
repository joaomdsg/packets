// Command packets is the control plane for agent-written code changes: it runs
// confirmed-catch cycles over revisions and serves the Console/Inspector where
// a human watches packets go in-flight → resolved. All CLI logic lives in
// internal/cli; this shell just calls it.
//
//	packets -repo . -base <baseSHA> -fix <fixSHA> -file pkg/auth/session.go -line 42
//	# console at http://localhost:3000
package main

import "github.com/joaomdsg/packets/internal/cli"

func main() { cli.Main() }
