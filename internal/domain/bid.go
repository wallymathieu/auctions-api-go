package domain

import (
	"time"
)

// Bid represents a bid in an auction
type Bid struct {
	ForAuction AuctionId `json:"auction"`
	Bidder     User      `json:"user"`
	At         time.Time `json:"at"`
	Amount     Amount    `json:"amount"`
}

// NewBid creates a new bid
func NewBid(auctionId AuctionId, bidder User, at time.Time, amount Amount) Bid {
	return Bid{
		ForAuction: auctionId,
		Bidder:     bidder,
		At:         at,
		Amount:     amount,
	}
}
