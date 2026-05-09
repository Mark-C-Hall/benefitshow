// Package stv implements Single Transferable Vote with the Droop quota for
// the benefitshow election. Surplus transfers use the Gregory fractional
// method; eliminations transfer ballots at their current weight.
package stv

import "sort"

// Run computes the STV ordering for the given ballots and seat count.
// Each ballot is exactly ten ranked song IDs.
//
// Returns song IDs in election order, then eliminated candidates in reverse
// elimination order, so the result page can show a full ranking. The output
// length equals the number of distinct candidates that appeared on at least
// one ballot.
//
// Ties at election or elimination time are broken by lower song ID first
// (spec §10: "ascending").
func Run(ballots [][10]int, seats int) []int {
	state := make([]ballotState, len(ballots))
	candidates := map[int]bool{}
	for i, b := range ballots {
		state[i] = ballotState{ranks: b, weight: 1.0, cursor: 0}
		for _, id := range b {
			candidates[id] = true
		}
	}

	quota := float64(len(ballots)/(seats+1) + 1)

	elected := make([]int, 0, seats)
	eliminated := make([]int, 0)
	electedSet := map[int]bool{}
	eliminatedSet := map[int]bool{}

	for len(elected) < seats {
		totals := map[int]float64{}
		contributors := map[int][]*ballotState{}
		for id := range candidates {
			if !electedSet[id] && !eliminatedSet[id] {
				totals[id] = 0
			}
		}
		if len(totals) == 0 {
			break
		}
		for i := range state {
			id, ok := state[i].current(electedSet, eliminatedSet)
			if !ok {
				continue
			}
			totals[id] += state[i].weight
			contributors[id] = append(contributors[id], &state[i])
		}

		active := make([]int, 0, len(totals))
		for id := range totals {
			active = append(active, id)
		}

		seatsLeft := seats - len(elected)
		if len(active) <= seatsLeft {
			sortByVotes(active, totals, true)
			for _, id := range active {
				elected = append(elected, id)
				electedSet[id] = true
			}
			break
		}

		var aboveQuota []int
		for _, id := range active {
			if totals[id] >= quota {
				aboveQuota = append(aboveQuota, id)
			}
		}

		if len(aboveQuota) > 0 {
			sortByVotes(aboveQuota, totals, true)
			winner := aboveQuota[0]
			elected = append(elected, winner)
			electedSet[winner] = true

			votes := totals[winner]
			factor := 0.0
			if votes > 0 {
				factor = (votes - quota) / votes
			}
			for _, b := range contributors[winner] {
				b.weight *= factor
				b.cursor++
			}
			continue
		}

		sortByVotes(active, totals, false)
		loser := active[0]
		eliminated = append(eliminated, loser)
		eliminatedSet[loser] = true
		for _, b := range contributors[loser] {
			b.cursor++
		}
	}

	out := make([]int, 0, len(elected)+len(eliminated))
	out = append(out, elected...)
	for i := len(eliminated) - 1; i >= 0; i-- {
		out = append(out, eliminated[i])
	}
	return out
}

// sortByVotes sorts ids by vote count, descending if highFirst else ascending.
// Ties are broken by lower song ID (deterministic, matches spec §10).
func sortByVotes(ids []int, totals map[int]float64, highFirst bool) {
	sort.Slice(ids, func(i, j int) bool {
		vi, vj := totals[ids[i]], totals[ids[j]]
		if vi != vj {
			if highFirst {
				return vi > vj
			}
			return vi < vj
		}
		return ids[i] < ids[j]
	})
}

type ballotState struct {
	ranks  [10]int
	weight float64
	cursor int
}

// current returns the ballot's current active preference, lazily skipping
// past candidates that have since been elected or eliminated. Returns
// (0, false) if the ballot is exhausted.
func (b *ballotState) current(elected, eliminated map[int]bool) (int, bool) {
	for b.cursor < len(b.ranks) {
		id := b.ranks[b.cursor]
		if !elected[id] && !eliminated[id] {
			return id, true
		}
		b.cursor++
	}
	return 0, false
}
