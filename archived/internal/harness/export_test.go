package harness

// SetAgentImage swaps the agent container image (normally "packets-agent") so an
// integration test can point RunContainer at a fake-claude image. It returns a
// restore func; not safe for concurrent use (tests that call it run serially).
func SetAgentImage(img string) (restore func()) {
	old := agentImage
	agentImage = img
	return func() { agentImage = old }
}
