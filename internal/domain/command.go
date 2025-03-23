package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Command interface represents a command in the system
type Command interface {
	GetTime() time.Time
}

// AddAuctionCommand represents a command to add a new auction
type AddAuctionCommand struct {
	Time    time.Time `json:"at"`
	Auction Auction   `json:"auction"`
}

// GetTime returns the time of the command
func (c AddAuctionCommand) GetTime() time.Time {
	return c.Time
}

// PlaceBidCommand represents a command to place a bid on an auction
type PlaceBidCommand struct {
	Time time.Time `json:"at"`
	Bid  Bid       `json:"bid"`
}

// GetTime returns the time of the command
func (c PlaceBidCommand) GetTime() time.Time {
	return c.Time
}

// Event interface represents an event in the system
type Event interface {
	GetTime() time.Time
}

// AuctionAddedEvent represents an event indicating an auction was added
type AuctionAddedEvent struct {
	Time    time.Time `json:"at"`
	Auction Auction   `json:"auction"`
}

// GetTime returns the time of the event
func (e AuctionAddedEvent) GetTime() time.Time {
	return e.Time
}

// BidAcceptedEvent represents an event indicating a bid was accepted
type BidAcceptedEvent struct {
	Time time.Time `json:"at"`
	Bid  Bid       `json:"bid"`
}

// GetTime returns the time of the event
func (e BidAcceptedEvent) GetTime() time.Time {
	return e.Time
}

// UnmarshalJSON implements json.Unmarshaler interface for Command
func UnmarshalCommand(data []byte) (Command, error) {
	var typeCheck struct {
		Type string `json:"$type"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, err
	}

	switch typeCheck.Type {
	case "AddAuction":
		var cmd AddAuctionCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	case "PlaceBid":
		var cmd PlaceBidCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, err
		}
		return cmd, nil
	default:
		return nil, fmt.Errorf("unknown command type: %s", typeCheck.Type)
	}
}

// MarshalJSON implements json.Marshaler interface for AddAuctionCommand
func (c AddAuctionCommand) MarshalJSON() ([]byte, error) {
	type addAuctionCommandJSON struct {
		Type    string   `json:"$type"`
		Time    time.Time `json:"at"`
		Auction Auction  `json:"auction"`
	}
	return json.Marshal(addAuctionCommandJSON{
		Type:    "AddAuction",
		Time:    c.Time,
		Auction: c.Auction,
	})
}

// MarshalJSON implements json.Marshaler interface for PlaceBidCommand
func (c PlaceBidCommand) MarshalJSON() ([]byte, error) {
	type placeBidCommandJSON struct {
		Type string    `json:"$type"`
		Time time.Time `json:"at"`
		Bid  Bid       `json:"bid"`
	}
	return json.Marshal(placeBidCommandJSON{
		Type: "PlaceBid",
		Time: c.Time,
		Bid:  c.Bid,
	})
}

// UnmarshalJSON implements json.Unmarshaler interface for Event
func UnmarshalEvent(data []byte) (Event, error) {
	var typeCheck struct {
		Type string `json:"$type"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, err
	}

	switch typeCheck.Type {
	case "AuctionAdded":
		var evt AuctionAddedEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return nil, err
		}
		return evt, nil
	case "BidAccepted":
		var evt BidAcceptedEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return nil, err
		}
		return evt, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", typeCheck.Type)
	}
}

// MarshalJSON implements json.Marshaler interface for AuctionAddedEvent
func (e AuctionAddedEvent) MarshalJSON() ([]byte, error) {
	type auctionAddedEventJSON struct {
		Type    string   `json:"$type"`
		Time    time.Time `json:"at"`
		Auction Auction  `json:"auction"`
	}
	return json.Marshal(auctionAddedEventJSON{
		Type:    "AuctionAdded",
		Time:    e.Time,
		Auction: e.Auction,
	})
}

// MarshalJSON implements json.Marshaler interface for BidAcceptedEvent
func (e BidAcceptedEvent) MarshalJSON() ([]byte, error) {
	type bidAcceptedEventJSON struct {
		Type string    `json:"$type"`
		Time time.Time `json:"at"`
		Bid  Bid       `json:"bid"`
	}
	return json.Marshal(bidAcceptedEventJSON{
		Type: "BidAccepted",
		Time: e.Time,
		Bid:  e.Bid,
	})
}

// Repository represents a repository of auctions
type Repository map[AuctionId]struct {
	Auction Auction
	State   State
}

// EventsToAuctionStates folds a list of events into a repository
func EventsToAuctionStates(events []Event) Repository {
	repo := make(Repository)
	
	for _, event := range events {
		switch e := event.(type) {
		case AuctionAddedEvent:
			auction := e.Auction
			state := auction.CreateEmptyState()
			repo[auction.ID] = struct {
				Auction Auction
				State   State
			}{
				Auction: auction,
				State:   state,
			}
		case BidAcceptedEvent:
			bid := e.Bid
			if entry, ok := repo[bid.ForAuction]; ok {
				nextState, _ := entry.State.AddBid(bid)
				repo[bid.ForAuction] = struct {
					Auction Auction
					State   State
				}{
					Auction: entry.Auction,
					State:   nextState,
				}
			}
		}
	}
	
	return repo
}

// Handle processes a command against a repository
func Handle(cmd Command, repo Repository) (Event, Repository, error) {
	switch c := cmd.(type) {
	case AddAuctionCommand:
		auction := c.Auction
		if _, exists := repo[auction.ID]; exists {
			return nil, repo, NewAuctionAlreadyExistsError(auction.ID)
		}
		
		// Create new state
		state := auction.CreateEmptyState()
		
		// Add to repository
		newRepo := copyRepository(repo)
		newRepo[auction.ID] = struct {
			Auction Auction
			State   State
		}{
			Auction: auction,
			State:   state,
		}
		
		return AuctionAddedEvent{
			Time:    c.Time,
			Auction: auction,
		}, newRepo, nil
		
	case PlaceBidCommand:
		bid := c.Bid
		auctionId := bid.ForAuction
		
		entry, exists := repo[auctionId]
		if !exists {
			return nil, repo, NewUnknownAuctionError(auctionId)
		}
		
		// Validate bid
		if err := entry.Auction.ValidateBid(bid); err != nil {
			return nil, repo, err
		}
		
		// Add bid to state
		nextState, err := entry.State.AddBid(bid)
		if err != nil {
			return nil, repo, err
		}
		
		// Update repository
		newRepo := copyRepository(repo)
		newRepo[auctionId] = struct {
			Auction Auction
			State   State
		}{
			Auction: entry.Auction,
			State:   nextState,
		}
		
		return BidAcceptedEvent{
			Time: c.Time,
			Bid:  bid,
		}, newRepo, nil
	}
	
	return nil, repo, fmt.Errorf("unknown command type")
}

// copyRepository creates a copy of the repository
func copyRepository(repo Repository) Repository {
	newRepo := make(Repository)
	for k, v := range repo {
		newRepo[k] = v
	}
	return newRepo
}

// GetAuctions returns all auctions in the repository
func GetAuctions(repo Repository) []Auction {
	auctions := make([]Auction, 0, len(repo))
	for _, entry := range repo {
		auctions = append(auctions, entry.Auction)
	}
	return auctions
}
