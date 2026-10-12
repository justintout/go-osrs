package hiscores

import (
	"context"
	"errors"

	"github.com/justintout/go-osrs"
)

// Account is a player's account type, inferred from which boards list them.
type Account struct {
	// Mode is Regular, Ironman, Hardcore, or Ultimate.
	Mode Mode `json:"mode"`
	// Former lists modes the player has left: de-ironed, died as a
	// hardcore, or dropped ultimate status.
	Former []Mode `json:"former,omitempty"`
	// Player is the Regular hiscores entry, which lists every account.
	Player *Player `json:"player"`
}

// LookupAccount infers a player's account type. It makes up to four
// requests: Regular, then Ironman, Ultimate, and Hardcore only for
// ironmen.
//
// A board stops updating when a player leaves its mode, so lower Overall
// XP on that board means the player left. A player who left a mode and has
// not gained XP since still reads as that mode. Group ironmen are not
// detected.
func (c *Client) LookupAccount(ctx context.Context, player string) (*Account, error) {
	regular, err := c.Lookup(ctx, player, Regular)
	if err != nil {
		return nil, err
	}
	acct := &Account{Mode: Regular, Player: regular}

	iron, err := c.lookupIfListed(ctx, player, Ironman)
	if err != nil || iron == nil {
		return acct, err
	}
	ironXP := iron.Overall().XP
	if regular.Overall().XP > ironXP {
		acct.Former = append(acct.Former, Ironman)
	} else {
		acct.Mode = Ironman
	}

	for _, m := range []Mode{Ultimate, Hardcore} {
		p, err := c.lookupIfListed(ctx, player, m)
		if err != nil {
			return nil, err
		}
		if p == nil {
			continue
		}
		if acct.Mode == Ironman && p.Overall().XP == ironXP {
			acct.Mode = m
		} else {
			acct.Former = append(acct.Former, m)
		}
	}
	return acct, nil
}

// lookupIfListed returns nil without error when the player is not on the
// mode's board.
func (c *Client) lookupIfListed(ctx context.Context, player string, mode Mode) (*Player, error) {
	p, err := c.Lookup(ctx, player, mode)
	if errors.Is(err, osrs.ErrNotFound) {
		return nil, nil
	}
	return p, err
}
