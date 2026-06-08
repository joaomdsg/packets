package fabric

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
)

// hostUser is the in-process host identity: full publish, so the orchestrator
// keeps minting unchanged. NoAuthUser maps the credential-less in-process
// connection to it, so the existing host connection works without creds.
const hostUser = "packets-host"

// ProducerGrant authorizes one cross-process producer: credentials (User/Pass)
// confined to publish ONLY its own session+instance claim subtree. It is the
// converged claim-submission schema — a producer emits claims, never mints.
type ProducerGrant struct {
	User     string
	Pass     string
	Session  string
	Instance string
}

// StartListening boots the fabric like Start but ALSO binds a TCP listener at
// addr (host:port; port 0 means a random free port) and enforces auth: the
// in-process host publishes anything, while each granted producer authenticates
// and may publish ONLY to its own claim subtree
// (packets.session.<session>.events.<instance>.claim.>) — minted subjects and
// other sessions are denied. Its subscribe is confined to its reply inbox, so a
// producer can neither forge a mint nor read another session's economy. The
// in-process Start path is unaffected.
func StartListening(ctx context.Context, dir, addr string, grants ...ProducerGrant) (*Fabric, error) {
	host, port, err := splitAddr(addr)
	if err != nil {
		return nil, err
	}
	// NoAuthUser maps EVERY credential-less connection — in-process AND external
	// socket — onto hostUser. Confining hostUser to IN_PROCESS connections is what
	// stops an anonymous external client from inheriting full mint privileges:
	// nats-server rejects an external standard connection mapped onto a user whose
	// AllowedConnectionTypes excludes STANDARD. The in-process host (iproc) still
	// matches, so minting is unaffected.
	users := []*server.User{{
		Username:               hostUser, // nil Permissions = full access
		AllowedConnectionTypes: map[string]struct{}{jwt.ConnectionTypeInProcess: {}},
	}}
	for _, g := range grants {
		users = append(users, &server.User{
			Username: g.User,
			Password: g.Pass,
			Permissions: &server.Permissions{
				Publish:   &server.SubjectPermission{Allow: []string{EventSubject(g.Session, g.Instance, StatusClaim, ">")}},
				Subscribe: &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
			},
		})
	}
	return boot(&server.Options{
		StoreDir:   dir,
		Host:       host,
		Port:       port,
		Users:      users,
		NoAuthUser: hostUser,
	})
}

// Addr is the bound listen address (host:port), or "" for an in-process-only
// fabric that listens on no socket.
func (f *Fabric) Addr() string {
	if a := f.ns.Addr(); a != nil {
		return a.String()
	}
	return ""
}

func splitAddr(addr string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("fabric: bad listen addr %q: %v", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("fabric: bad listen port in %q: %v", addr, err)
	}
	if port == 0 {
		port = -1 // a random free port (the embedded server's convention)
	}
	return host, port, nil
}
