package purchase

import (
	"fmt"

	"github.com/witsaba/paca/packages/domain/ledger"
)

// The state machine of the draft.
//
// A draft starts in `proposed` and it waits there for the yes of the owner.
// It ends in one of three states: `confirmed` when the owner confirmed it,
// `cancelled` when she dropped it, and `expired` when nobody answered it.
// It never returns to `proposed`: a draft that ended does not wait again.
//
// The expiry is computed by the domain when the draft is read: a draft that
// was neither confirmed nor cancelled seven days after the day it was
// proposed reads as `expired`. The storage needs no job to expire anything,
// and the row keeps the state it was written with.

// ReadsAs answers the state the draft reads as on the day of the caller.
//
// The rule of the domain: a draft that waits reads as `proposed`, and a
// draft that was neither confirmed nor cancelled seven days after the day
// it was proposed reads as `expired`. A draft that ended reads as the state
// it ended in.
func (d Draft) ReadsAs(today ledger.Date) State {
	if d.State != StateProposed {
		return d.State
	}

	if today.After(d.ExpiresOn) {
		return StateExpired
	}

	return StateProposed
}

// Cancel drops a draft the owner does not want.
//
// The rules of the state machine hold here:
//
//   - only a draft that waits is cancelled: a draft that is confirmed, that
//     is cancelled already or that expired does not wait any more, and the
//     domain refuses it;
//   - a draft never returns to proposed: the state it ends in is the state
//     it keeps.
//
// The error is ErrNotCancellable when the draft does not wait any more, and
// ErrExpiredDraft when it expired.
func Cancel(draft Draft, today ledger.Date) (Draft, error) {
	switch draft.State {
	case StateProposed:
	case StateCancelled:
		return Draft{}, fmt.Errorf("%w: the draft is cancelled already", ErrNotCancellable)
	case StateConfirmed:
		return Draft{}, fmt.Errorf("%w: the draft is confirmed already", ErrNotCancellable)
	default:
		return Draft{}, fmt.Errorf("%w: the draft is in the state %q", ErrNotCancellable, draft.State)
	}

	if today.After(draft.ExpiresOn) {
		return Draft{}, fmt.Errorf("%w: the draft expired the day %v", ErrExpiredDraft, draft.ExpiresOn)
	}

	cancelled := draft
	cancelled.State = StateCancelled

	return cancelled, nil
}

// Pending answers the drafts that still wait for the yes of the owner.
//
// The rule of the domain: a pending draft is one that reads as `proposed`
// on the day of the caller. A draft that expired and a draft that ended do
// not wait any more, and they are not pending.
//
// The list is empty and not nothing: the agent reads a list.
func Pending(drafts []Draft, today ledger.Date) []Draft {
	pending := make([]Draft, 0, len(drafts))

	for _, one := range drafts {
		if one.ReadsAs(today) == StateProposed {
			pending = append(pending, one)
		}
	}

	return pending
}
