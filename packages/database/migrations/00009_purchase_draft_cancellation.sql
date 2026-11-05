-- The cancellation of a draft, and the state it ends in.
--
-- A draft starts in the state `proposed` and it waits there for the yes of
-- the owner. It ends in one of these: `confirmed` when the owner confirmed
-- it, `cancelled` when she dropped it, and `expired` when nobody answered
-- it.
--
-- The expiry is computed by the domain when the draft is read: a draft that
-- was neither confirmed nor cancelled seven days after the day it was
-- proposed reads as `expired`, and the table needs no job to expire
-- anything. The state of a row is the state it was written with, and the
-- CHECK holds the whole vocabulary of the domain as a defense in depth.
--
-- A draft never returns to `proposed`: the state it ends in is the state it
-- keeps, and the row that is not waiting any more is not cancelled again.

-- +goose Up
ALTER TABLE purchase_drafts DROP CONSTRAINT purchase_drafts_state_check;
ALTER TABLE purchase_drafts ADD CONSTRAINT purchase_drafts_state_check
    CHECK (state IN ('proposed', 'confirmed', 'cancelled', 'expired'));

-- +goose Down
ALTER TABLE purchase_drafts DROP CONSTRAINT purchase_drafts_state_check;
ALTER TABLE purchase_drafts ADD CONSTRAINT purchase_drafts_state_check
    CHECK (state IN ('proposed', 'confirmed'));
