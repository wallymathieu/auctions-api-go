package web

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"auction-site-go/internal/domain"
)

// getAuctions returns all auctions
func getAuctions(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := state.GetRepository()
		auctions := domain.GetAuctions(repo)

		// Convert to AuctionListItem
		auctionItems := make([]AuctionListItem, len(auctions))
		for i, auction := range auctions {
			auctionItems[i] = AuctionListItem{
				ID:       auction.ID,
				StartsAt: auction.StartsAt,
				Title:    auction.Title,
				Expiry:   auction.Expiry,
				Currency: auction.Currency,
			}
		}

		respondJSON(w, http.StatusOK, auctionItems)
	}
}

// getAuction returns a specific auction
func getAuction(state *AppState, getCurrentTime func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse auction ID from path
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid auction ID")
			return
		}

		// Get auction from repository
		repo := state.GetRepository()
		entry, ok := repo[domain.AuctionId(id)]
		if !ok {
			respondDomainError(w, domain.NewUnknownAuctionError(domain.AuctionId(id)))
			return
		}

		auction := entry.Auction
		// Advance state to the current time so a winner surfaces once the auction has ended.
		auctionState := entry.State.Increment(getCurrentTime())

		// Get bids
		bids := auctionState.GetBids()
		bidResponses := make([]AuctionBidResponse, len(bids))
		for i, bid := range bids {
			bidResponses[i] = AuctionBidResponse{
				Amount: bid.Amount,
				Bidder: bid.Bidder,
			}
		}

		// Get winner information
		var winner *domain.UserId
		var winnerPrice *int64
		if amount, userId, found := auctionState.TryGetAmountAndWinner(); found {
			winner = &userId
			winnerPrice = &amount
		}

		// Create response
		response := AuctionResponse{
			ID:          auction.ID,
			StartsAt:    auction.StartsAt,
			Title:       auction.Title,
			Expiry:      auction.Expiry,
			Currency:    auction.Currency,
			Bids:        bidResponses,
			Winner:      winner,
			WinnerPrice: winnerPrice,
		}

		respondJSON(w, http.StatusOK, response)
	}
}

// createAuction creates a new auction
func createAuction(state *AppState, onCommand func(domain.Command) error, onEvent func(domain.Event) error, getCurrentTime func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var req AddAuctionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Extract user from JWT
		user, err := extractUserFromRequest(r)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Create auction
		var auctionType domain.AuctionType
		if req.Type.Type != 0 {
			auctionType = req.Type
		} else {
			// Default to English auction
			options := domain.DefaultTimedAscendingOptions()
			auctionType = domain.NewTimedAscendingType(options)
		}

		auction := domain.Auction{
			ID:       req.ID,
			StartsAt: req.StartsAt,
			Title:    req.Title,
			Expiry:   req.EndsAt,
			Seller:   user,
			Type:     auctionType,
			Currency: req.Currency,
		}

		now := getCurrentTime()

		// Reject auctions whose EndsAt is not strictly in the future.
		if !req.EndsAt.After(now) {
			respondDomainError(w, domain.NewAuctionEndsAtInPastError(req.ID))
			return
		}

		// Create command
		cmd := domain.AddAuctionCommand{
			Time:    now,
			Auction: auction,
		}

		if err := onCommand(cmd); err != nil {
			log.Printf("Failed to observe command: %v", err)
			respondError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Handle command
		repo := state.GetRepository()
		event, newRepo, err := domain.Handle(cmd, repo)
		if err != nil {
			respondDomainError(w, err)
			return
		}

		// Update repository
		state.UpdateRepository(newRepo)

		// Call event handler
		if err := onEvent(event); err != nil {
			log.Printf("Failed to observe event: %v", err)
			respondError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Return the event
		respondJSON(w, http.StatusOK, event)
	}
}

// placeBid places a bid on an auction
func placeBid(state *AppState, onCommand func(domain.Command) error, onEvent func(domain.Event) error, getCurrentTime func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse auction ID from path
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid auction ID")
			return
		}

		// Parse request body
		var req BidRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Extract user from JWT
		user, err := extractUserFromRequest(r)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Create bid
		bid := domain.Bid{
			ForAuction: domain.AuctionId(id),
			Bidder:     user,
			At:         getCurrentTime(),
			Amount:     req.Amount,
		}

		// Create command
		cmd := domain.PlaceBidCommand{
			Time: getCurrentTime(),
			Bid:  bid,
		}

		if err := onCommand(cmd); err != nil {
			log.Printf("Failed to observe command: %v", err)
			respondError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Handle command
		repo := state.GetRepository()
		event, newRepo, err := domain.Handle(cmd, repo)
		if err != nil {
			respondDomainError(w, err)
			return
		}

		// Update repository
		state.UpdateRepository(newRepo)

		// Call event handler
		if err := onEvent(event); err != nil {
			log.Printf("Failed to observe event: %v", err)
			respondError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Return the event
		respondJSON(w, http.StatusOK, event)
	}
}

// extractUserFromRequest extracts a user from an HTTP request
func extractUserFromRequest(r *http.Request) (domain.User, error) {
	authHeader := r.Header.Get("x-jwt-payload")
	if authHeader == "" {
		return domain.User{}, errors.New("missing authentication header")
	}

	// In the test, trim any whitespace
	authHeader = strings.TrimSpace(authHeader)

	return DecodeJwtUser(authHeader)
}

// respondJSON responds with a JSON payload
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

// respondError responds with an error message
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ApiError{Message: message})
}

// domainErrorRenderer maps a domain error code to an HTTP status and a
// payload builder over the error's structured Data. The web layer is the
// only place that knows how to render domain codes for clients.
type domainErrorRenderer struct {
	status  int
	payload func(data interface{}) map[string]interface{}
}

// withAuctionId returns a renderer that produces {type, auctionId} payloads.
func withAuctionId(typeName string, status int) domainErrorRenderer {
	return domainErrorRenderer{
		status: status,
		payload: func(data interface{}) map[string]interface{} {
			return map[string]interface{}{"type": typeName, "auctionId": data}
		},
	}
}

// Wire type names mirror the reference (F#) implementation. ErrorAuctionEndsAtInPast
// shares the "AuctionHasEnded" wire value with ErrorAuctionHasEnded, so the lookup
// resolves both Go-side codes through the same renderer entry.
var domainErrorRenderers = map[domain.ErrorType]domainErrorRenderer{
	domain.ErrorAuctionNotFound:       withAuctionId("AuctionNotFound", http.StatusNotFound),
	domain.ErrorAuctionAlreadyExists: withAuctionId("AuctionAlreadyExists", http.StatusBadRequest),
	domain.ErrorAuctionHasEnded:      withAuctionId("AuctionHasEnded", http.StatusBadRequest),
	domain.ErrorAuctionHasNotStarted: withAuctionId("AuctionHasNotStarted", http.StatusBadRequest),
	domain.ErrorAlreadyPlacedBid: {
		status: http.StatusBadRequest,
		payload: func(_ interface{}) map[string]interface{} {
			return map[string]interface{}{"type": "AlreadyPlacedBid"}
		},
	},
	domain.ErrorSellerCannotPlaceBids: {
		status: http.StatusBadRequest,
		payload: func(data interface{}) map[string]interface{} {
			resp := map[string]interface{}{"type": "SellerCannotPlaceBids"}
			if d, ok := data.(map[string]interface{}); ok {
				for k, v := range d {
					resp[k] = v
				}
			}
			return resp
		},
	},
	domain.ErrorMustPlaceBidOverHighest: {
		status: http.StatusBadRequest,
		payload: func(data interface{}) map[string]interface{} {
			return map[string]interface{}{"type": "MustPlaceBidOverHighestBid", "amount": data}
		},
	},
}

// respondDomainError translates a domain error into the typed HTTP error
// envelope expected by API clients: {"type": "...", ...}. Non-domain errors
// and unmapped codes are logged and returned as a generic 500 so internal
// details never leak across the boundary.
func respondDomainError(w http.ResponseWriter, err error) {
	domainErr, ok := err.(domain.DomainError)
	if !ok {
		log.Printf("non-domain error at HTTP boundary: %v", err)
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	renderer, ok := domainErrorRenderers[domainErr.Type]
	if !ok {
		log.Printf("unmapped domain error code at HTTP boundary: %v", domainErr)
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondJSON(w, renderer.status, renderer.payload(domainErr.Data))
}
