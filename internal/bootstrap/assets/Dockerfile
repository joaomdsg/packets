FROM {{BASE}}

RUN apt-get update && apt-get install -y --no-install-recommends \
      curl gnupg ca-certificates git jq \
 && install -d -m 0755 /etc/apt/keyrings \
 && curl -fsSL https://downloads.claude.ai/keys/claude-code.asc \
      -o /etc/apt/keyrings/claude-code.asc \
 && echo "deb [signed-by=/etc/apt/keyrings/claude-code.asc] \
https://downloads.claude.ai/claude-code/apt/stable stable main" \
      > /etc/apt/sources.list.d/claude-code.list \
 && apt-get update && apt-get install -y claude-code \
 && rm -rf /var/lib/apt/lists/*

# Some base images (e.g. ubuntu:24.04) ship a preexisting uid-1000 user;
# --user 1000 at run time requires agent to own that uid, so displace
# whatever is already there first.
RUN existing="$(getent passwd 1000 | cut -d: -f1)"; \
    [ -n "$existing" ] && [ "$existing" != "agent" ] && userdel -r "$existing" 2>/dev/null; \
    useradd -m -u 1000 agent

# Claude Code needs its config dir writable (auth/session state, hook
# tampering aside) but --read-only at run time only leaves tmpfs mounts
# writable. So the shipped settings/hooks live root-owned and read-only
# here, and the entrypoint reseeds a tmpfs at $CLAUDE_CONFIG_DIR from this
# copy on every container start (docs/claude-code-facts.md).
COPY settings.json /opt/packets/claude-config/settings.json
COPY hooks/ /opt/packets/claude-config/hooks/
RUN chmod +x /opt/packets/claude-config/hooks/*.sh \
 && chown -R root:root /opt/packets/claude-config \
 && chmod -R a-w /opt/packets/claude-config

COPY entrypoint.sh /opt/packets/entrypoint.sh
RUN chmod +x /opt/packets/entrypoint.sh

USER agent
WORKDIR /work
ENV CLAUDE_CONFIG_DIR=/home/agent/.claude-config
ENTRYPOINT ["/opt/packets/entrypoint.sh"]
