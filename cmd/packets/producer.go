package main

import (
	"fmt"
	"strings"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/fabric"
)

// producerFlag collects repeatable -producer specs into producer grants.
type producerFlag struct{ grants []fabric.ProducerGrant }

func (p *producerFlag) String() string {
	users := make([]string, len(p.grants))
	for i, g := range p.grants {
		users[i] = g.Session + ":" + g.User // never print the password
	}
	return strings.Join(users, " ")
}

func (p *producerFlag) Set(v string) error {
	g, err := parseProducerSpec(v)
	if err != nil {
		return err
	}
	p.grants = append(p.grants, g)
	return nil
}

// parseProducerSpec parses a "key:user:pass" spec into a producer grant confined
// to session "key" (producer == session key). Only the first two colons split
// the fields, so a password may itself contain colons. All three fields are
// required; a half spec fails fast rather than authorizing a malformed producer.
func parseProducerSpec(spec string) (fabric.ProducerGrant, error) {
	parts := strings.SplitN(spec, ":", 3)
	if len(parts) != 3 {
		return fabric.ProducerGrant{}, fmt.Errorf("producer %q: want key:user:pass", spec)
	}
	key, user, pass := parts[0], parts[1], parts[2]
	if key == "" || user == "" || pass == "" {
		return fabric.ProducerGrant{}, fmt.Errorf("producer %q: key, user and pass are all required", spec)
	}
	return app.NewProducerGrant(key, user, pass), nil
}
