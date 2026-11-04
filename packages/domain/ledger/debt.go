package ledger

import (
	"fmt"
	"sort"

	"github.com/witsaba/paca/packages/domain/account"
)

// Debt is what the person owes, or what the person is owed.
//
// The debt is an account and never a table of its own: it is an account
// that owes or is owed, with the other side of the money named, and the
// amount is what that account still holds. A positive amount means the debt
// is open: what the person owes for a liability, what the person is owed
// for an asset.
type Debt struct {
	// Account is the account of the debt: its kind, its name, the other
	// side of the money and the day it falls due.
	Account account.Account

	// Amount is what the debt still holds. It is positive.
	Amount Amount
}

// AcceptsPayment answers whether a payment may be paid against the debt.
//
// The payment must be positive: money that does not move pays nothing. And
// it may not pass what the debt still holds: a payment bigger than the debt
// is money the person does not owe.
//
// The error is ErrInvalidAmount when the payment is not positive, and
// ErrPaymentTooLarge when it passes the amount of the debt.
func (d Debt) AcceptsPayment(payment Amount) error {
	if payment <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidAmount, payment)
	}

	if payment > d.Amount {
		return fmt.Errorf("%w: the payment of %d passes the debt of %d", ErrPaymentTooLarge, payment, d.Amount)
	}

	return nil
}

// Debts returns the debts that the accounts of a book hold.
//
// Only the accounts that carry a debt are debts: an account that owes or is
// owed and that names the other side of the money. The amount of the debt
// is the balance of the account, read the way the person reads it: what the
// person still owes for a liability, what the person is still owed for an
// asset.
//
// Only the open debts are in the answer: a debt that was paid, or that was
// never opened, holds nothing and the person does not read it. An account
// that carries no balance holds nothing as well, and it is not in the
// answer either.
//
// The debts come first by the day they fall due, and the debt without a day
// comes last: the person reads what is due before what has no date. Two
// debts of the same day come by name, so the answer is the same every time
// it is read.
func Debts(accounts []account.Account, balances []Balance) []Debt {
	byAccount := make(map[account.ID]Balance, len(balances))
	for _, balance := range balances {
		byAccount[balance.AccountID] = balance
	}

	debts := make([]Debt, 0, len(accounts))

	for _, candidate := range accounts {
		if !candidate.IsDebt() {
			continue
		}

		balance, found := byAccount[candidate.ID]
		if !found {
			continue
		}

		amount := balance.Normal()
		if amount <= 0 {
			continue
		}

		debts = append(debts, Debt{Account: candidate, Amount: amount})
	}

	sort.SliceStable(debts, func(i, j int) bool {
		return comesFirst(debts[i], debts[j])
	})

	return debts
}

// comesFirst reports whether a debt is read before another one.
//
// The debt with a day falls due before the debt without one, the debt that
// falls due earlier falls due before the other one, and two debts of the
// same day are read by name.
func comesFirst(one, other Debt) bool {
	first, second := one.Account.DueDate, other.Account.DueDate

	switch {
	case first == nil && second == nil:
		return one.Account.Name < other.Account.Name
	case first == nil:
		return false
	case second == nil:
		return true
	case first.Before(*second):
		return true
	case second.Before(*first):
		return false
	default:
		return one.Account.Name < other.Account.Name
	}
}
