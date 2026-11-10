package learning

import (
	"fmt"

	"github.com/witsaba/paca/packages/domain/business"
)

// Log is the learning log of a business: the entries it wrote, in the order
// they were written.
//
// The log is never rewritten: the entries of the corrections are entries of
// their own, and the chain of the answers says which word is the last one.
type Log struct {
	// Entries are the entries of the log, in the order they were written.
	Entries []Entry
}

// TheLog answers the log of the business.
//
// The rule of the domain: the log is read in the order the entries were
// written, and the entries that answer another one are in it: the log never
// hides the past, and it is never rewritten.
func TheLog(entries []Entry) Log {
	return Log{Entries: entries}
}

// Current answers the entries that are the LAST WORD of their chain.
//
// The rule of the domain: an entry is current when NO OTHER ENTRY ANSWERS IT.
// The entry that answers another one is the correction of it, and the entry
// that nothing answers is the last word of its chain — the entry that was
// never corrected, and the entry that corrected it. The corrections of the
// past stay in the log, and the last word is what the owner already knows:
// it is what Paca tells her.
//
// The order of the log is kept, and the answer of a log with no entry is an
// empty list and not nil.
func (l Log) Current() []Entry {
	answered := make(map[ID]bool, len(l.Entries))
	for _, one := range l.Entries {
		if one.Answers != 0 {
			answered[one.Answers] = true
		}
	}

	current := make([]Entry, 0, len(l.Entries))

	for _, one := range l.Entries {
		if answered[one.ID] {
			continue
		}

		current = append(current, one)
	}

	return current
}

// Answering answers the entry of the log that a new entry answers, and it
// holds the rules of the answer:
//
//   - the entry that is answered must be there (ErrEntryNotFound);
//   - it must be of the same business (ErrEntryOfAnotherBusiness);
//   - it must not have been answered already (ErrAlreadyAnswered): the log is
//     a chain and not a tree, and two answers of the same entry would leave
//     the owner with two truths.
//
// The use case asks the log for the entry, and the log decides: a correction
// is a NEW entry that answers the old one, and the old one keeps its own
// words.
func (l Log) Answering(target ID, own business.ID) (Entry, error) {
	answered, found := l.theEntry(target)
	if !found {
		return Entry{}, fmt.Errorf("%w: the log of the business %d holds no entry %d", ErrEntryNotFound, own, target)
	}

	if answered.BusinessID != own {
		return Entry{}, fmt.Errorf("%w: the entry %d is of the business %d and the log is of the business %d",
			ErrEntryOfAnotherBusiness, target, answered.BusinessID, own)
	}

	for _, one := range l.Entries {
		if one.Answers == target {
			return Entry{}, fmt.Errorf("%w: the entry %d of the business %d is answered by the entry %d",
				ErrAlreadyAnswered, target, own, one.ID)
		}
	}

	return answered, nil
}

// theEntry answers the entry of the log that carries that identity.
func (l Log) theEntry(id ID) (Entry, bool) {
	for _, one := range l.Entries {
		if one.ID == id {
			return one, true
		}
	}

	return Entry{}, false
}
