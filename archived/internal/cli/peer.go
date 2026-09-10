package cli

import (
	"fmt"
	"strings"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/fabric"
)

// peerFlag collects repeatable -peer specs into peer grants.
type peerFlag struct{ grants []fabric.Grant }

func (p *peerFlag) String() string {
	users := make([]string, len(p.grants))
	for i, g := range p.grants {
		users[i] = g.Session + ":" + g.User // never print the password
	}
	return strings.Join(users, " ")
}

func (p *peerFlag) Set(v string) error {
	g, err := parsePeerSpec(v)
	if err != nil {
		return err
	}
	p.grants = append(p.grants, g)
	return nil
}

// parsePeerSpec parses a "key:user:pass" spec into a peer grant confined
// to session "key" (peer == session key). Only the first two colons split
// the fields, so a password may itself contain colons. All three fields are
// required; a half spec fails fast rather than authorizing a malformed peer.
func parsePeerSpec(spec string) (fabric.Grant, error) {
	parts := strings.SplitN(spec, ":", 3)
	if len(parts) != 3 {
		return fabric.Grant{}, fmt.Errorf("peer %q: want key:user:pass", spec)
	}
	key, user, pass := parts[0], parts[1], parts[2]
	if key == "" || user == "" || pass == "" {
		return fabric.Grant{}, fmt.Errorf("peer %q: key, user and pass are all required", spec)
	}
	return app.NewGrant(key, user, pass), nil
}
