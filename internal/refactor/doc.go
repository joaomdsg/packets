// Package refactor holds the adversarial refactor trace. It has no production
// code of its own: it is an acceptance suite
// that runs real refactors — a large rename, a neutral move, an extract-module
// — through the existing pipe (diff, mutation, reanchor, catch) and asserts the
// CURRENT carnage as expected-failure baselines. These baselines quantify the
// damage the re-anchoring/refactor-aware work must absorb, and they fail loudly
// (turning green) the day the pipe learns to handle behavior-preserving change.
package refactor
