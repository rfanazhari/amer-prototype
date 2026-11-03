package interviewreport

// active reports whether the state is pending or running.
func (s State) active() bool { return s == StatePending || s == StateRunning }

// consistent reports whether the generation's state, reason, statements, and times are coherent.
func (g Generation) consistent(number int) bool {
	if g.ID == "" || g.Number != number || g.CreatedAt.IsZero() || g.UpdatedAt.Before(g.CreatedAt) {
		return false
	}
	switch g.State {
	case StatePending, StateRunning:
		return g.FailureReason == "" && len(g.Statements) == 0
	case StateSucceeded:
		if g.FailureReason != "" {
			return false
		}
		for _, st := range g.Statements {
			if !st.valid() {
				return false
			}
		}
		return true
	case StateFailed:
		return g.FailureReason != "" && len(g.Statements) == 0
	}
	return false
}

// consistent reports whether identities, generation numbering, and the single active generation hold.
func (s Snapshot) consistent() bool {
	if s.ID == "" || s.ExecutionID == "" || s.AssignmentID == "" || s.CreatedAt.IsZero() {
		return false
	}
	active := 0
	for i, g := range s.Generations {
		if !g.consistent(i + 1) {
			return false
		}
		if g.State.active() {
			active++
		}
	}
	return active <= 1
}

// detach copies the statement slice so callers cannot mutate the aggregate through it.
func (g Generation) detach() Generation {
	if g.Statements != nil {
		g.Statements = append([]Statement(nil), g.Statements...)
	}
	return g
}

// detach copies every generation so callers cannot mutate the aggregate through the snapshot.
func (s Snapshot) detach() Snapshot {
	if s.Generations != nil {
		gens := make([]Generation, len(s.Generations))
		for i, g := range s.Generations {
			gens[i] = g.detach()
		}
		s.Generations = gens
	}
	return s
}
