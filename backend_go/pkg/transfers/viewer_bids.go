package transfers

import (
	"fmt"

	"football_sim/pkg/models"
)

// Tier B transfer control (B3): the viewer makes offers, counters, and
// accepts through the same five-stage negotiation FSM the AI uses. Every fee
// keeps the valuation corridor and every completion goes through
// executeTransfer, so no viewer action can bypass the engine's economics.

// ViewerOffer opens a negotiation with the viewer's opening offer. The
// returned reason is empty on success.
func (te *TransferEngine) ViewerOffer(playerID, buyerID string, bidAmount int64) (*TransferNegotiation, string) {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.initiateBidUnlocked(playerID, buyerID, bidAmount)
}

// ViewerRespond steers an active negotiation: "improve" raises the bid (and
// completes the transfer through the FSM when the offer meets the asking
// price), "withdraw" collapses it. The returned reason is empty on success.
func (te *TransferEngine) ViewerRespond(negotiationID, action string, amount int64) (*TransferNegotiation, string) {
	te.mu.Lock()
	defer te.mu.Unlock()

	var neg *TransferNegotiation
	for i := range te.ActiveNegotiations {
		if te.ActiveNegotiations[i].NegotiationID == negotiationID {
			neg = te.ActiveNegotiations[i]
			break
		}
	}
	if neg == nil {
		return nil, "Negotiation not found."
	}
	if neg.StageName == "COLLAPSED" || neg.StageName == "COMPLETED" {
		return neg, "That negotiation is already closed."
	}

	switch action {
	case "withdraw":
		neg.StageName = "COLLAPSED"
		neg.History = append(neg.History, "Withdrawn by the viewer")
		te.prependFeed(TransferFeedItem{
			Headline: fmt.Sprintf("TALKS OFF: %s walk away from the %s deal", neg.Buyer.ShortName, neg.Player.FullName),
			Category: "REJECTED", IsWonderkid: neg.IsWonderkid,
			Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek),
		})
		return neg, ""

	case "improve":
		if amount <= neg.CurrentBid {
			return neg, "An improved offer must exceed the current bid."
		}
		amount = models.ClampValue(amount, neg.Player.OVR, neg.Player.Age, neg.Player.UniverseWonderkid)
		if amount <= neg.CurrentBid {
			return neg, "That offer exceeds the valuation ceiling for this player."
		}
		if !canAfford(neg.Buyer, amount) {
			return neg, "The buying club cannot afford that offer."
		}
		if !te.canAffordWithWage(neg.Buyer, amount, annualWageFor(neg.Player)) {
			return neg, "That offer does not fit under the wage cap."
		}
		neg.CurrentBid = amount
		neg.History = append(neg.History, fmt.Sprintf("Viewer offer: %s", models.FormatCurrency(amount)))
		if amount >= neg.AskingPrice {
			// Accept: complete through the same execution path the FSM uses.
			if te.executeTransfer(neg) {
				neg.StageName = "COMPLETED"
				neg.ProgressPct = 100
				te.removeActiveNegotiation(negotiationID)
				return neg, ""
			}
			return neg, "The transfer could not be completed."
		}
		te.prependFeed(TransferFeedItem{
			Headline: fmt.Sprintf("%s table an improved %s bid for %s", neg.Buyer.ShortName, models.FormatCurrency(amount), neg.Player.FullName),
			Category: "EXCLUSIVE", IsWonderkid: neg.IsWonderkid,
			Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek),
		})
		return neg, ""

	default:
		return neg, "Unknown negotiation action."
	}
}

// removeActiveNegotiation drops a completed negotiation from the active list.
// Caller must hold te.mu.
func (te *TransferEngine) removeActiveNegotiation(negotiationID string) {
	for i := range te.ActiveNegotiations {
		if te.ActiveNegotiations[i].NegotiationID == negotiationID {
			te.ActiveNegotiations = append(te.ActiveNegotiations[:i], te.ActiveNegotiations[i+1:]...)
			return
		}
	}
}
