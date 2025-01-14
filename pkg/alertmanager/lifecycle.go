// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/cortexproject/cortex/blob/master/pkg/alertmanager/lifecycle.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Cortex Authors.

package alertmanager

import (
	"github.com/grafana/dskit/ring"
)

func (am *MultitenantAlertmanager) OnRingInstanceRegister(_ *ring.BasicLifecycler, ringDesc ring.Desc, instanceExists bool, instanceID string, instanceDesc ring.InstanceDesc) (ring.InstanceState, ring.Tokens) {
	// When we initialize the alertmanager instance in the ring we want to start from
	// a clean situation, so whatever is the state we set it JOINING, while we keep existing
	// tokens (if any).
	var tokens []uint32
	if instanceExists {
		tokens = instanceDesc.GetTokens()
	}

	_, takenTokens := ringDesc.TokensFor(instanceID)
	newTokens := ring.GenerateTokens(RingNumTokens-len(tokens), takenTokens)

	// Tokens sorting will be enforced by the parent caller.
	tokens = append(tokens, newTokens...)

	return ring.JOINING, tokens
}

func (am *MultitenantAlertmanager) OnRingInstanceTokens(_ *ring.BasicLifecycler, _ ring.Tokens) {}
func (am *MultitenantAlertmanager) OnRingInstanceStopping(_ *ring.BasicLifecycler)              {}
func (am *MultitenantAlertmanager) OnRingInstanceHeartbeat(_ *ring.BasicLifecycler, _ *ring.Desc, _ *ring.InstanceDesc) {
}
