package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// UserId is a unique identifier for a user
type UserId string

// AuctionId is a unique identifier for an auction
type AuctionId int64

// User represents either a buyer/seller or support user
type User struct {
	ID   UserId `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "BuyerOrSeller" or "Support"
}

// NewBuyerOrSeller creates a new buyer or seller user
func NewBuyerOrSeller(id UserId, name string) User {
	return User{
		ID:   id,
		Name: name,
		Type: "BuyerOrSeller",
	}
}

// NewSupport creates a new support user
func NewSupport(id UserId) User {
	return User{
		ID:   id,
		Type: "Support",
	}
}

// MarshalJSON implements json.Marshaler interface
func (u User) MarshalJSON() ([]byte, error) {
	var s string
	if u.Type == "BuyerOrSeller" {
		s = fmt.Sprintf("BuyerOrSeller|%s|%s", u.ID, u.Name)
	} else {
		s = fmt.Sprintf("Support|%s", u.ID)
	}
	return json.Marshal(s)
}

// UnmarshalJSON implements json.Unmarshaler interface
func (u *User) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parts := strings.Split(s, "|")
	if len(parts) == 0 {
		return fmt.Errorf("invalid user format: %s", s)
	}

	u.Type = parts[0]
	if u.Type == "BuyerOrSeller" {
		if len(parts) != 3 {
			return fmt.Errorf("invalid BuyerOrSeller format: %s", s)
		}
		u.ID = UserId(parts[1])
		u.Name = parts[2]
	} else if u.Type == "Support" {
		if len(parts) != 2 {
			return fmt.Errorf("invalid Support format: %s", s)
		}
		u.ID = UserId(parts[1])
	} else {
		return fmt.Errorf("unknown user type: %s", u.Type)
	}

	return nil
}

// Domain errors
type ErrorType string

const (
	ErrorAuctionNotFound          ErrorType = "AuctionNotFound"
	ErrorAuctionAlreadyExists    ErrorType = "AuctionAlreadyExists"
	ErrorAuctionHasEnded         ErrorType = "AuctionHasEnded"
	ErrorAuctionHasNotStarted    ErrorType = "AuctionHasNotStarted"
	ErrorAuctionEndsAtInPast     ErrorType = "AuctionHasEnded"
	ErrorSellerCannotPlaceBids   ErrorType = "SellerCannotPlaceBids"
	ErrorMustPlaceBidOverHighest ErrorType = "MustPlaceBidOverHighestBid"
	ErrorAlreadyPlacedBid        ErrorType = "AlreadyPlacedBid"
)

// DomainError carries a stable code (Type) and optional structured Data.
// Rendering of human-readable messages is the responsibility of the outer
// (e.g. HTTP) layer; this struct deliberately holds no free-form text.
type DomainError struct {
	Type ErrorType
	Data interface{}
}

func (e DomainError) Error() string {
	return string(e.Type)
}

// NewUnknownAuctionError creates a new UnknownAuction error
func NewUnknownAuctionError(id AuctionId) error {
	return DomainError{
		Type: ErrorAuctionNotFound,
		Data: id,
	}
}

// NewAuctionAlreadyExistsError creates a new AuctionAlreadyExists error
func NewAuctionAlreadyExistsError(id AuctionId) error {
	return DomainError{
		Type: ErrorAuctionAlreadyExists,
		Data: id,
	}
}

// NewAuctionHasEndedError creates a new AuctionHasEnded error
func NewAuctionHasEndedError(id AuctionId) error {
	return DomainError{
		Type: ErrorAuctionHasEnded,
		Data: id,
	}
}

// NewAuctionHasNotStartedError creates a new AuctionHasNotStarted error
func NewAuctionHasNotStartedError(id AuctionId) error {
	return DomainError{
		Type: ErrorAuctionHasNotStarted,
		Data: id,
	}
}

// NewSellerCannotPlaceBidsError creates a new SellerCannotPlaceBids error
func NewSellerCannotPlaceBidsError(userId UserId, auctionId AuctionId) error {
	return DomainError{
		Type: ErrorSellerCannotPlaceBids,
		Data: map[string]interface{}{
			"userId":    userId,
			"auctionId": auctionId,
		},
	}
}

// NewAuctionEndsAtInPastError is returned when an auction is created with an
// EndsAt that is not strictly in the future relative to the current time.
func NewAuctionEndsAtInPastError(id AuctionId) error {
	return DomainError{
		Type: ErrorAuctionEndsAtInPast,
		Data: id,
	}
}

// NewMustPlaceBidOverHighestError creates a new MustPlaceBidOverHighest error
func NewMustPlaceBidOverHighestError(amount int64) error {
	return DomainError{
		Type: ErrorMustPlaceBidOverHighest,
		Data: amount,
	}
}

// NewAlreadyPlacedBidError creates a new AlreadyPlacedBid error
func NewAlreadyPlacedBidError() error {
	return DomainError{
		Type: ErrorAlreadyPlacedBid,
	}
}
