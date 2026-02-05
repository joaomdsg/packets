FROM debian:bookworm-slim

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  git \
  sqlite3 \
  gcc \
  libc6-dev \
  procps \
  file \
  build-essential \
  && rm -rf /var/lib/apt/lists/*

# Install Go
ARG GO_VERSION=1.23.4
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
  | tar -C /usr/local -xzf -
ENV PATH="/usr/local/go/bin:${PATH}"

# Install Node.js (required for Claude Code)
ARG NODE_VERSION=22
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
  && apt-get install -y nodejs \
  && rm -rf /var/lib/apt/lists/*

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
  | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
  && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
  && echo "deb [arch=$(dpkg --print-architecture) \
  signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] \
  https://cli.github.com/packages stable main" \
  > /etc/apt/sources.list.d/github-cli.list \
  && apt-get update \
  && apt-get install -y gh \
  && rm -rf /var/lib/apt/lists/*

# Install Docker CLI
RUN curl -fsSL https://download.docker.com/linux/debian/gpg \
  | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
  && echo "deb [arch=$(dpkg --print-architecture) \
  signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] \
  https://download.docker.com/linux/debian bookworm stable" \
  > /etc/apt/sources.list.d/docker.list \
  && apt-get update \
  && apt-get install -y docker-ce-cli \
  && rm -rf /var/lib/apt/lists/*

# Install Claude Code CLI via npm
RUN npm install -g @anthropic-ai/claude-code

# Create work directory
RUN mkdir -p /work
WORKDIR /app

# Copy go modules first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 go build -o /usr/local/bin/agntpr ./cmd/agntpr

# Copy entrypoint script
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create non-root user for security (Claude Code requires non-root)
RUN useradd -m -s /bin/bash agent \
  && mkdir -p /work /data \
  && chown -R agent:agent /work /data

# Set environment for agent user
ENV WORK_DIR=/work
ENV HOME=/home/agent

# Switch to non-root user
USER agent
WORKDIR /home/agent

# Install Homebrew to /work/.homebrew (must be run as non-root user)
ENV HOMEBREW_PREFIX=/work/.homebrew
ENV HOMEBREW_CELLAR=/work/.homebrew/Cellar
ENV HOMEBREW_REPOSITORY=/work/.homebrew
ENV HOMEBREW_NO_AUTO_UPDATE=1
RUN mkdir -p /work/.homebrew \
  && git clone https://github.com/Homebrew/brew /work/.homebrew \
  && echo 'eval "$(/work/.homebrew/bin/brew shellenv)"' >> ~/.bashrc

# Add Homebrew to PATH
ENV PATH="/work/.homebrew/bin:/work/.homebrew/sbin:${PATH}"

ENTRYPOINT ["entrypoint.sh"]
