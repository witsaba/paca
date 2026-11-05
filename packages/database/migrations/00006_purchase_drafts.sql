-- The draft of a purchase: the candidate the agent proposes from what the
-- owner said, and the state it waits in.
--
-- The draft is a proposal, not a fact: it is kept by the business, with its
-- state, its revision and the sentence it came from, and it waits for the
-- yes of the owner before anything is written in the book. The transport of
-- the MCP server is stateless, and a candidate that lives only in the
-- context of the model is a candidate that can be changed between the
-- proposal and the yes.
--
-- The lines of a draft, what it still needs, what it was not sure about
-- and what PACA filled by herself are JSONB: they are the reading of a
-- sentence, not the rows of a fact. The money enters the book with the
-- purchase, and not with the draft.
--
-- The rules of the domain are that the sentence is not empty, that the
-- revision grows by one and that the draft expires seven days after the
-- day it was proposed. The table holds the same rules as a defense in
-- depth.

-- +goose Up
CREATE TABLE purchase_drafts (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT      NOT NULL REFERENCES businesses (id),
    state       TEXT        NOT NULL DEFAULT 'proposed',
    revision    INTEGER     NOT NULL DEFAULT 1,
    utterance   TEXT        NOT NULL,
    lines       JSONB       NOT NULL DEFAULT '[]'::jsonb,
    missing     JSONB       NOT NULL DEFAULT '[]'::jsonb,
    ambiguous   JSONB       NOT NULL DEFAULT '[]'::jsonb,
    inferred    JSONB       NOT NULL DEFAULT '[]'::jsonb,
    day         DATE        NULL,
    market      TEXT        NULL,
    invoice     TEXT        NULL,
    proposed_on DATE        NOT NULL,
    expires_on  DATE        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT purchase_drafts_state_check CHECK (state IN ('proposed')),
    CONSTRAINT purchase_drafts_utterance_not_empty_check CHECK (utterance <> ''),
    CONSTRAINT purchase_drafts_revision_check CHECK (revision > 0),
    CONSTRAINT purchase_drafts_expiry_check CHECK (expires_on >= proposed_on),
    CONSTRAINT purchase_drafts_lines_check CHECK (jsonb_typeof(lines) = 'array'),
    CONSTRAINT purchase_drafts_missing_check CHECK (jsonb_typeof(missing) = 'array'),
    CONSTRAINT purchase_drafts_ambiguous_check CHECK (jsonb_typeof(ambiguous) = 'array'),
    CONSTRAINT purchase_drafts_inferred_check CHECK (jsonb_typeof(inferred) = 'array')
);

-- +goose Down
DROP TABLE purchase_drafts;
