package transfers

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

// TransferFeedItem represents a breaking news wire item.
type TransferFeedItem struct {
	Headline    string `json:"headline"`
	Category    string `json:"category"` // EXCLUSIVE, TWIST, HIJACK, HERE_WE_GO, RUMOR, REJECTED
	IsWonderkid bool   `json:"is_wonderkid"`
	Matchweek   int    `json:"matchweek"`
	Timestamp   string `json:"timestamp"`
}

// CompletedTransfer represents an officially signed transfer deal.
type CompletedTransfer struct {
	PlayerID     string `json:"player_id"`
	PlayerName   string `json:"player_name"`
	PlayerPos    string `json:"player_pos"`
	PlayerOVR    int    `json:"player_ovr"`
	IsWonderkid  bool   `json:"is_wonderkid"`
	SellerID     string `json:"seller_id"`
	SellerName   string `json:"seller_name"`
	SellerShort  string `json:"seller_short"`
	BuyerID      string `json:"buyer_id"`
	BuyerName    string `json:"buyer_name"`
	BuyerShort   string `json:"buyer_short"`
	FeeEUR       int64  `json:"fee_eur"`
	FormattedFee string `json:"formatted_fee"`
	Matchweek    int    `json:"matchweek"`
}

// TransferNegotiation tracks a multi-stage negotiation state machine.
type TransferNegotiation struct {
	NegotiationID    string         `json:"negotiation_id"`
	Player           *models.Player `json:"player"`
	Buyer            *models.Club   `json:"buyer"`
	Seller           *models.Club   `json:"seller"`
	CurrentBid       int64          `json:"current_bid"`
	AskingPrice      int64          `json:"asking_price"`
	CreatedMatchweek int            `json:"created_matchweek"`
	StageIndex       int            `json:"stage_index"` // 1 to 5
	StageName        string         `json:"stage_name"`  // INQUIRY, COUNTER_OFFER, HIJACK_CHECK, TERMS_MEDICAL, COMPLETED, REJECTED, COLLAPSED
	ProgressPct      int            `json:"progress_pct"`
	IsWonderkid      bool           `json:"is_wonderkid"`
	IsHijacked       bool           `json:"is_hijacked"`
	OriginalBuyer    *models.Club   `json:"original_buyer,omitempty"`
	History          []string       `json:"history"`
}

type ClubNetSpend struct {
	ClubID       string   `json:"club_id"`
	ClubName     string   `json:"club_name"`
	ShortName    string   `json:"short_name"`
	PrimaryColor [3]uint8 `json:"primary_color"`
	Spent        int64    `json:"spent"`
	Received     int64    `json:"received"`
	Net          int64    `json:"net"`
}

type TransferRecordsData struct {
	TopSignings         []CompletedTransfer     `json:"top_signings"`
	NetSpend            map[string]ClubNetSpend `json:"net_spend"`
	TotalTransfersCount int                     `json:"total_transfers_count"`
}

var superLeagueClubIDs = map[string]bool{
	"LAL-BAR": true, "LAL-RMA": true, "LAL-ATM": true,
	"EPL-ARS": true, "EPL-LIV": true, "EPL-TOT": true,
	"BUN-BAY": true, "BUN-DOR": true,
	"SEA-INT": true, "SEA-NAP": true, "SEA-MIL": true,
	"FL1-PSG": true, "FRA-PSG": true,
}

func isSuperLeagueClub(clubID string) bool {
	return superLeagueClubIDs[clubID]
}

// CalculateClubWarchest returns the initialized budget between €50M and €250M
// based on club rating and stature.
func CalculateClubWarchest(teamRating int) int64 {
	diff := teamRating - 78
	if diff < 0 {
		diff = 0
	}
	budget := int64(60_000_000 + diff*12_000_000)
	if budget < 50_000_000 {
		budget = 50_000_000
	}
	if budget > 250_000_000 {
		budget = 250_000_000
	}
	return budget
}

// TransferEngine oversees the transfer market, negotiations, roster mutations, and news wire.
type TransferEngine struct {
	mu                    sync.RWMutex
	Clubs                 map[string]*models.Club
	Managers              map[string]*managers.ManagerProfile
	CurrentMatchweek      int
	CurrentDay            int
	CurrentWeek           int
	IsOffSeason           bool
	TransferredThisWindow map[string]bool
	ActiveNegotiations    []*TransferNegotiation
	TransferFeed          []TransferFeedItem
	CompletedTransfers    []CompletedTransfer
	AllTimeTransfers      []CompletedTransfer
	RNG                   *rand.Rand
}

// NewTransferEngine initializes a new transfer engine.
func NewTransferEngine(clubs []*models.Club, mgrs map[string]*managers.ManagerProfile, seed int64) *TransferEngine {
	if seed == 0 {
		seed = 20260907
	}
	clubsMap := make(map[string]*models.Club)
	for _, c := range clubs {
		clubsMap[c.ClubID] = c
	}
	if mgrs == nil {
		mgrs = managers.BuildManagers(clubs)
	}

	te := &TransferEngine{
		Clubs:                 clubsMap,
		Managers:              mgrs,
		CurrentMatchweek:      1,
		CurrentDay:            1,
		CurrentWeek:           1,
		IsOffSeason:           false,
		TransferredThisWindow: make(map[string]bool),
		ActiveNegotiations:    make([]*TransferNegotiation, 0),
		TransferFeed:          make([]TransferFeedItem, 0),
		CompletedTransfers:    make([]CompletedTransfer, 0),
		AllTimeTransfers:      make([]CompletedTransfer, 0),
		RNG:                   rand.New(rand.NewSource(seed)),
	}

	// Initialize warchest balances for each club between €50M and €250M
	for cid, mgr := range mgrs {
		club := clubsMap[cid]
		if club == nil || mgr == nil {
			continue
		}
		mgr.BudgetEur = CalculateClubWarchest(club.OverallTeamRating)
	}

	te.injectInitialRumors()
	return te
}

func (te *TransferEngine) injectInitialRumors() {
	te.TransferFeed = append(te.TransferFeed, TransferFeedItem{
		Headline:    "Summer Window officially opens across the European Super League!",
		Category:    "EXCLUSIVE",
		IsWonderkid: false,
		Matchweek:   1,
		Timestamp:   "MW 1",
	})
	te.TransferFeed = append(te.TransferFeed, TransferFeedItem{
		Headline:    "Scouts descend on Barcelona academy to monitor 14yo prodigy Venjamin Valerio",
		Category:    "RUMOR",
		IsWonderkid: true,
		Matchweek:   1,
		Timestamp:   "MW 1",
	})
}

// IsWindowOpen returns true if currently inside the transfer window.
func (te *TransferEngine) IsWindowOpen() bool {
	if te.IsOffSeason {
		return te.CurrentWeek >= 1 && te.CurrentWeek <= 12
	}
	return (te.CurrentMatchweek >= 1 && te.CurrentMatchweek <= 4) ||
		(te.CurrentMatchweek >= 21 && te.CurrentMatchweek <= 24)
}

// GetWindowName returns human-readable market window status.
func (te *TransferEngine) GetWindowName() string {
	if te.IsOffSeason {
		if te.CurrentWeek >= 1 && te.CurrentWeek <= 12 {
			return fmt.Sprintf("Summer Window (Open - Week %d of 12)", te.CurrentWeek)
		}
		return "Window Closed (Opens at season end)"
	}
	if te.CurrentMatchweek >= 1 && te.CurrentMatchweek <= 4 {
		rem := 5 - te.CurrentMatchweek
		return fmt.Sprintf("Summer Window (Open - %d MW left)", rem)
	} else if te.CurrentMatchweek >= 21 && te.CurrentMatchweek <= 24 {
		rem := 25 - te.CurrentMatchweek
		return fmt.Sprintf("Winter Window (Open - %d MW left)", rem)
	} else if te.CurrentMatchweek < 21 {
		return "Window Closed (Scouting Active - Opens MW 21)"
	}
	return "Window Closed (Scouting for Summer)"
}

// RevertToBaselines pulls every valuation a little back toward its long-run
// anchor so match-to-match swings cannot drift the market (Python: revert_to_baselines).
func (te *TransferEngine) RevertToBaselines(pull ...float64) {
	te.mu.Lock()
	defer te.mu.Unlock()
	p := 0.08
	if len(pull) > 0 && pull[0] > 0 {
		p = pull[0]
	}
	for _, club := range te.Clubs {
		if club == nil {
			continue
		}
		for _, pl := range club.Squad {
			if pl == nil || pl.MarketValueEUR <= 0 {
				continue
			}
			anchor := models.BaselineValue(pl.OVR, pl.Age, pl.UniverseWonderkid)
			pl.MarketValueEUR = pl.MarketValueEUR + int64(float64(anchor-pl.MarketValueEUR)*p)
			models.ClampPlayer(pl)
		}
	}
}

// ResetForNewSeason clears in-season talks and refreshes board warchests.
func (te *TransferEngine) ResetForNewSeason() {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.ActiveNegotiations = te.ActiveNegotiations[:0]
	te.CompletedTransfers = te.CompletedTransfers[:0]
	te.CurrentDay = 1
	te.CurrentWeek = 1
	te.CurrentMatchweek = 1
	te.IsOffSeason = false
	te.TransferredThisWindow = make(map[string]bool)
	for cid, manager := range te.Managers {
		club := te.Clubs[cid]
		if club == nil || manager == nil {
			continue
		}
		manager.BudgetEur = CalculateClubWarchest(club.OverallTeamRating)
	}
	te.TransferFeed = append([]TransferFeedItem{{
		Headline:  "New season, clean slate: every Super League squad confirmed, market closed until the finale.",
		Category:  "EXCLUSIVE",
		Matchweek: 1,
		Timestamp: "MW 1",
	}}, te.TransferFeed...)
}

// UpdateDailyMarket advances negotiations and triggers new AI transfer bids
// when the matchweek calendar says the window is open.
func (te *TransferEngine) UpdateDailyMarket() {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.updateDailyMarketUnlocked(te.IsWindowOpen())
}

// AdvanceOpenWindow advances the off-season transfer market by one week (Weeks 1 to 12).
// In each weekly stage, active negotiations progress, solvent Super League clubs
// place bids within their warchests, and the window week increments.
func (te *TransferEngine) AdvanceOpenWindow() {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.IsOffSeason = true
	if te.CurrentWeek > 12 {
		return
	}
	te.advanceWeeklyMarketUnlocked()
}

func (te *TransferEngine) advanceWeeklyMarketUnlocked() {
	var surviving []*TransferNegotiation
	for _, neg := range te.ActiveNegotiations {
		resolved := te.progressNegotiation(neg)
		if !resolved {
			surviving = append(surviving, neg)
		}
	}
	te.ActiveNegotiations = surviving

	// Solvent AI clubs place bids within budget limits
	numBids := 2 + te.RNG.Intn(3)
	for i := 0; i < numBids; i++ {
		te.aiInitiateBid()
	}

	te.CurrentWeek++
	te.CurrentDay++
}

func (te *TransferEngine) updateDailyMarketUnlocked(windowOpen bool) {
	var surviving []*TransferNegotiation
	for _, neg := range te.ActiveNegotiations {
		resolved := te.progressNegotiation(neg)
		if !resolved {
			surviving = append(surviving, neg)
		}
	}
	te.ActiveNegotiations = surviving

	if windowOpen {
		numBids := 1 + te.RNG.Intn(2)
		for i := 0; i < numBids; i++ {
			te.aiInitiateBid()
		}
	}
	te.CurrentDay++
}

func (te *TransferEngine) progressNegotiation(neg *TransferNegotiation) bool {
	if neg == nil || neg.Player == nil || neg.Seller == nil || neg.Buyer == nil {
		return true
	}

	// Strict single transfer per window rule: abort if already moved
	if te.TransferredThisWindow[neg.Player.PlayerID] {
		neg.StageName = "COLLAPSED"
		return true
	}

	// Verify player is still with seller
	found := false
	for _, p := range neg.Seller.Squad {
		if p != nil && p.PlayerID == neg.Player.PlayerID {
			found = true
			break
		}
	}
	if !found {
		neg.StageName = "COLLAPSED"
		return true
	}

	// Verify buyer can still afford the deal
	if buyerMgr, ok := te.Managers[neg.Buyer.ClubID]; ok && buyerMgr != nil {
		if buyerMgr.BudgetEur < neg.CurrentBid {
			neg.StageName = "COLLAPSED"
			te.TransferFeed = append(te.TransferFeed, TransferFeedItem{
				Headline:    fmt.Sprintf("DEAL COLLAPSED: %s cannot fund %s move for %s due to budget limits", neg.Buyer.ShortName, models.FormatCurrency(neg.CurrentBid), neg.Player.FullName),
				Category:    "REJECTED",
				IsWonderkid: neg.IsWonderkid,
				Matchweek:   te.CurrentMatchweek,
				Timestamp:   fmt.Sprintf("Week %d", te.CurrentWeek),
			})
			return true
		}
	}

	neg.StageIndex++
	switch neg.StageIndex {
	case 2:
		// Stage 2: Counter-Offer (40%)
		neg.StageName = "COUNTER_OFFER"
		neg.ProgressPct = 40
		// Seller demands 10-25% more than initial bid
		increase := float64(neg.CurrentBid) * (0.10 + te.RNG.Float64()*0.15)
		newBid := neg.CurrentBid + int64(increase)

		// Ensure buyer's warchest can cover the increase
		if buyerMgr, ok := te.Managers[neg.Buyer.ClubID]; ok && buyerMgr != nil {
			if buyerMgr.BudgetEur < newBid {
				newBid = buyerMgr.BudgetEur
			}
		}
		neg.AskingPrice = newBid
		neg.CurrentBid = newBid
		te.TransferFeed = append(te.TransferFeed, TransferFeedItem{
			Headline:    fmt.Sprintf("%s submit improved bid of %s for %s (%s)", neg.Buyer.ShortName, models.FormatCurrency(neg.CurrentBid), neg.Player.FullName, neg.Seller.ShortName),
			Category:    "TWIST",
			IsWonderkid: neg.IsWonderkid,
			Matchweek:   te.CurrentMatchweek,
			Timestamp:   fmt.Sprintf("Week %d", te.CurrentWeek),
		})
		return false

	case 3:
		// Stage 3: Hijack Check (60%)
		neg.StageName = "HIJACK_CHECK"
		neg.ProgressPct = 60
		// 15% chance of rival club hijack if player OVR >= 82 or is wonderkid
		if (neg.Player.OVR >= 82 || neg.IsWonderkid) && te.RNG.Float64() < 0.15 {
			var rivals []*models.Club
			hijackCost := int64(float64(neg.CurrentBid) * 1.15)
			for _, c := range te.Clubs {
				if c == nil || c.ClubID == neg.Buyer.ClubID || c.ClubID == neg.Seller.ClubID {
					continue
				}
				// Wonderkids may only transfer between 12 Super League clubs
				if (neg.IsWonderkid || strings.HasPrefix(neg.Player.PlayerID, "WK_")) && !isSuperLeagueClub(c.ClubID) {
					continue
				}
				if rMgr, ok := te.Managers[c.ClubID]; ok && rMgr != nil {
					if rMgr.BudgetEur >= hijackCost {
						rivals = append(rivals, c)
					}
				}
			}
			if len(rivals) > 0 {
				hijacker := rivals[te.RNG.Intn(len(rivals))]
				neg.OriginalBuyer = neg.Buyer
				neg.Buyer = hijacker
				neg.IsHijacked = true
				neg.CurrentBid = hijackCost
				te.TransferFeed = append(te.TransferFeed, TransferFeedItem{
					Headline:    fmt.Sprintf("HIJACK TWIST! %s hijack %s deal for %s with late %s offer!", hijacker.ShortName, neg.OriginalBuyer.ShortName, neg.Player.FullName, models.FormatCurrency(neg.CurrentBid)),
					Category:    "HIJACK",
					IsWonderkid: neg.IsWonderkid,
					Matchweek:   te.CurrentMatchweek,
					Timestamp:   fmt.Sprintf("Week %d", te.CurrentWeek),
				})
			}
		}
		return false

	case 4:
		// Stage 4: Personal Terms & Medical (80%)
		neg.StageName = "TERMS_MEDICAL"
		neg.ProgressPct = 80
		te.TransferFeed = append(te.TransferFeed, TransferFeedItem{
			Headline:    fmt.Sprintf("Medical booked: %s arrives at %s training ground ahead of final signature", neg.Player.FullName, neg.Buyer.ShortName),
			Category:    "EXCLUSIVE",
			IsWonderkid: neg.IsWonderkid,
			Matchweek:   te.CurrentMatchweek,
			Timestamp:   fmt.Sprintf("Week %d", te.CurrentWeek),
		})
		return false

	case 5:
		// Stage 5: HERE WE GO & Official Completion (100%)
		neg.StageName = "COMPLETED"
		neg.ProgressPct = 100
		te.executeTransfer(neg)
		return true

	default:
		return true
	}
}

func (te *TransferEngine) executeTransfer(neg *TransferNegotiation) {
	if neg == nil || neg.Player == nil || neg.Seller == nil || neg.Buyer == nil {
		return
	}

	// Strict single-transfer rule: cannot transfer twice in the same window
	if te.TransferredThisWindow[neg.Player.PlayerID] {
		return
	}
	te.TransferredThisWindow[neg.Player.PlayerID] = true

	// 1. Remove player from seller squad
	var newSellerSquad []*models.Player
	for _, p := range neg.Seller.Squad {
		if p != nil && p.PlayerID != neg.Player.PlayerID {
			newSellerSquad = append(newSellerSquad, p)
		}
	}
	neg.Seller.Squad = newSellerSquad

	// 2. Add player to buyer squad
	neg.Player.ClubID = neg.Buyer.ClubID
	if !neg.Player.UniverseWonderkid && !strings.HasPrefix(neg.Player.PlayerID, "WK_") {
		// Permanent transfer: new parent club
		neg.Player.OriginalClubID = neg.Buyer.ClubID
	}
	// For canonical wonderkids, OriginalClubID remains untouched so they return home at season reset.
	neg.Buyer.Squad = append(neg.Buyer.Squad, neg.Player)

	// 3. Update warchests
	if buyerMgr, ok := te.Managers[neg.Buyer.ClubID]; ok && buyerMgr != nil {
		buyerMgr.BudgetEur -= neg.CurrentBid
		if buyerMgr.BudgetEur < 0 {
			buyerMgr.BudgetEur = 0
		}
	}
	if sellerMgr, ok := te.Managers[neg.Seller.ClubID]; ok && sellerMgr != nil {
		sellerMgr.BudgetEur += neg.CurrentBid
	}

	completed := CompletedTransfer{
		PlayerID:     neg.Player.PlayerID,
		PlayerName:   neg.Player.FullName,
		PlayerPos:    neg.Player.Position,
		PlayerOVR:    neg.Player.OVR,
		IsWonderkid:  neg.IsWonderkid,
		SellerID:     neg.Seller.ClubID,
		SellerName:   neg.Seller.ClubName,
		SellerShort:  neg.Seller.ShortName,
		BuyerID:      neg.Buyer.ClubID,
		BuyerName:    neg.Buyer.ClubName,
		BuyerShort:   neg.Buyer.ShortName,
		FeeEUR:       neg.CurrentBid,
		FormattedFee: models.FormatCurrency(neg.CurrentBid),
		Matchweek:    te.CurrentMatchweek,
	}

	te.CompletedTransfers = append(te.CompletedTransfers, completed)
	te.AllTimeTransfers = append(te.AllTimeTransfers, completed)

	weekOrMw := fmt.Sprintf("Week %d", te.CurrentWeek)
	if te.CurrentWeek <= 1 && te.CurrentMatchweek > 1 {
		weekOrMw = fmt.Sprintf("MW %d", te.CurrentMatchweek)
	}
	te.TransferFeed = append([]TransferFeedItem{
		{
			Headline:    fmt.Sprintf("HERE WE GO: %s signs for %s in %s deal from %s!", neg.Player.FullName, neg.Buyer.ShortName, models.FormatCurrency(neg.CurrentBid), neg.Seller.ShortName),
			Category:    "HERE_WE_GO",
			IsWonderkid: neg.IsWonderkid,
			Matchweek:   te.CurrentMatchweek,
			Timestamp:   weekOrMw,
		},
	}, te.TransferFeed...)
}

func (te *TransferEngine) aiInitiateBid() {
	var clubs []*models.Club
	for _, c := range te.Clubs {
		clubs = append(clubs, c)
	}
	if len(clubs) < 2 {
		return
	}

	// Filter solvent buyers with at least €5M available warchest
	var solventClubs []*models.Club
	for _, c := range clubs {
		mgr := te.Managers[c.ClubID]
		if mgr != nil && mgr.BudgetEur >= 5_000_000 {
			solventClubs = append(solventClubs, c)
		}
	}
	if len(solventClubs) == 0 {
		return
	}

	buyer := solventClubs[te.RNG.Intn(len(solventClubs))]
	buyerMgr := te.Managers[buyer.ClubID]
	if buyerMgr == nil || buyerMgr.BudgetEur <= 0 {
		return
	}

	var sellers []*models.Club
	for _, c := range clubs {
		if c.ClubID != buyer.ClubID && len(c.Squad) > 15 {
			sellers = append(sellers, c)
		}
	}
	if len(sellers) == 0 {
		return
	}
	seller := sellers[te.RNG.Intn(len(sellers))]

	// Target a player from seller
	var validTargets []*models.Player
	for _, p := range seller.Squad {
		if p == nil {
			continue
		}
		// Strict single-transfer per window rule
		if te.TransferredThisWindow[p.PlayerID] {
			continue
		}
		// Avoid targeting players already negotiating
		alreadyNeg := false
		for _, n := range te.ActiveNegotiations {
			if n.Player != nil && n.Player.PlayerID == p.PlayerID {
				alreadyNeg = true
				break
			}
		}
		if alreadyNeg {
			continue
		}
		// Wonderkid constraint: canonical prodigies only transfer between 12 Super League clubs
		isWK := p.UniverseWonderkid || strings.HasPrefix(p.PlayerID, "WK_")
		if isWK {
			if !isSuperLeagueClub(buyer.ClubID) || !isSuperLeagueClub(seller.ClubID) {
				continue
			}
		}
		// Budget constraint: estimated bid cannot exceed buyer's warchest
		baseVal := models.BaselineValue(p.OVR, p.Age, p.UniverseWonderkid)
		estBid := int64(float64(baseVal) * 1.10)
		if estBid > buyerMgr.BudgetEur {
			continue
		}
		if p.OVR >= 76 || isWK {
			validTargets = append(validTargets, p)
		}
	}
	if len(validTargets) == 0 {
		return
	}

	target := validTargets[te.RNG.Intn(len(validTargets))]
	baseVal := models.BaselineValue(target.OVR, target.Age, target.UniverseWonderkid)
	initialBid := int64(float64(baseVal) * (0.95 + te.RNG.Float64()*0.20))
	if initialBid > buyerMgr.BudgetEur {
		initialBid = buyerMgr.BudgetEur
	}
	if initialBid <= 0 {
		return
	}

	negID := fmt.Sprintf("NEG_%s_%s_%d", buyer.ClubID, target.PlayerID, te.CurrentDay)
	neg := &TransferNegotiation{
		NegotiationID:    negID,
		Player:           target,
		Buyer:            buyer,
		Seller:           seller,
		CurrentBid:       initialBid,
		AskingPrice:      initialBid,
		CreatedMatchweek: te.CurrentMatchweek,
		StageIndex:       1,
		StageName:        "INQUIRY",
		ProgressPct:      20,
		IsWonderkid:      target.UniverseWonderkid,
		History:          []string{fmt.Sprintf("Initial bid: %s", models.FormatCurrency(initialBid))},
	}
	te.ActiveNegotiations = append(te.ActiveNegotiations, neg)

	weekOrMw := fmt.Sprintf("Week %d", te.CurrentWeek)
	if te.CurrentWeek <= 1 && te.CurrentMatchweek > 1 {
		weekOrMw = fmt.Sprintf("MW %d", te.CurrentMatchweek)
	}
	te.TransferFeed = append([]TransferFeedItem{
		{
			Headline:    fmt.Sprintf("%s open talks with %s for %s with opening %s bid", buyer.ShortName, seller.ShortName, target.FullName, models.FormatCurrency(initialBid)),
			Category:    "EXCLUSIVE",
			IsWonderkid: target.UniverseWonderkid,
			Matchweek:   te.CurrentMatchweek,
			Timestamp:   weekOrMw,
		},
	}, te.TransferFeed...)
}

// GetTransferRecords compiles top all-time signings and club net spends.
func (te *TransferEngine) GetTransferRecords() TransferRecordsData {
	te.mu.RLock()
	defer te.mu.RUnlock()

	all := make([]CompletedTransfer, len(te.AllTimeTransfers))
	copy(all, te.AllTimeTransfers)

	sort.Slice(all, func(i, j int) bool {
		return all[i].FeeEUR > all[j].FeeEUR
	})

	top := all
	if len(top) > 10 {
		top = top[:10]
	}

	netSpend := make(map[string]ClubNetSpend)
	for _, club := range te.Clubs {
		netSpend[club.ClubID] = ClubNetSpend{
			ClubID:       club.ClubID,
			ClubName:     club.ClubName,
			ShortName:    club.ShortName,
			PrimaryColor: club.PrimaryColor,
		}
	}

	for _, t := range te.AllTimeTransfers {
		if b, ok := netSpend[t.BuyerID]; ok {
			b.Spent += t.FeeEUR
			b.Net -= t.FeeEUR
			netSpend[t.BuyerID] = b
		}
		if s, ok := netSpend[t.SellerID]; ok {
			s.Received += t.FeeEUR
			s.Net += t.FeeEUR
			netSpend[t.SellerID] = s
		}
	}

	return TransferRecordsData{
		TopSignings:         top,
		NetSpend:            netSpend,
		TotalTransfersCount: len(te.AllTimeTransfers),
	}
}

// InitiateBid starts a negotiation for a player on behalf of a buyer.
func (te *TransferEngine) InitiateBid(playerID string, buyerID string, bidAmount int64) *TransferNegotiation {
	te.mu.Lock()
	defer te.mu.Unlock()

	if te.TransferredThisWindow[playerID] {
		return nil
	}

	buyer := te.Clubs[buyerID]
	if buyer == nil {
		return nil
	}

	var target *models.Player
	var seller *models.Club
	for _, c := range te.Clubs {
		for _, p := range c.Squad {
			if p.PlayerID == playerID {
				target = p
				seller = c
				break
			}
		}
		if target != nil {
			break
		}
	}
	if target == nil || seller == nil || seller.ClubID == buyerID {
		return nil
	}

	// Wonderkid constraint: canonical prodigies only transfer between 12 Super League clubs
	if (target.UniverseWonderkid || strings.HasPrefix(target.PlayerID, "WK_")) && (!isSuperLeagueClub(buyer.ClubID) || !isSuperLeagueClub(seller.ClubID)) {
		return nil
	}

	if bidAmount <= 0 {
		baseVal := models.BaselineValue(target.OVR, target.Age, target.UniverseWonderkid)
		bidAmount = int64(float64(baseVal) * 1.05)
	}

	// Warchest check: cannot exceed buyer's available budget
	if buyerMgr, ok := te.Managers[buyerID]; ok && buyerMgr != nil {
		if buyerMgr.BudgetEur < bidAmount {
			return nil
		}
	}

	negID := fmt.Sprintf("NEG_%s_%s_%d", buyer.ClubID, target.PlayerID, te.CurrentDay)
	neg := &TransferNegotiation{
		NegotiationID:    negID,
		Player:           target,
		Buyer:            buyer,
		Seller:           seller,
		CurrentBid:       bidAmount,
		AskingPrice:      bidAmount,
		CreatedMatchweek: te.CurrentMatchweek,
		StageIndex:       1,
		StageName:        "INQUIRY",
		ProgressPct:      20,
		IsWonderkid:      target.UniverseWonderkid,
		History:          []string{fmt.Sprintf("Initial bid: %s", models.FormatCurrency(bidAmount))},
	}
	te.ActiveNegotiations = append(te.ActiveNegotiations, neg)

	weekOrMw := fmt.Sprintf("Week %d", te.CurrentWeek)
	if te.CurrentWeek <= 1 && te.CurrentMatchweek > 1 {
		weekOrMw = fmt.Sprintf("MW %d", te.CurrentMatchweek)
	}
	te.TransferFeed = append([]TransferFeedItem{
		{
			Headline:    fmt.Sprintf("%s open talks with %s for %s with opening %s bid", buyer.ShortName, seller.ShortName, target.FullName, models.FormatCurrency(bidAmount)),
			Category:    "EXCLUSIVE",
			IsWonderkid: target.UniverseWonderkid,
			Matchweek:   te.CurrentMatchweek,
			Timestamp:   weekOrMw,
		},
	}, te.TransferFeed...)

	return neg
}

func (te *TransferEngine) prependFeed(item TransferFeedItem) {
	te.TransferFeed = append([]TransferFeedItem{item}, te.TransferFeed...)
	if len(te.TransferFeed) > 120 {
		te.TransferFeed = te.TransferFeed[:120]
	}
}

// TriggerSpecificBid opens a Super League negotiation for the given clubs and
// player. An existing talk for that player is returned unchanged. The fee is
// derived from value, not a client-supplied amount.
func (te *TransferEngine) TriggerSpecificBid(buyerID, sellerID, playerID string) *TransferNegotiation {
	te.mu.Lock()
	defer te.mu.Unlock()

	if te.TransferredThisWindow[playerID] {
		return nil
	}

	buyer := te.Clubs[buyerID]
	seller := te.Clubs[sellerID]
	if buyer == nil {
		return nil
	}
	if seller == nil && playerID != "" {
		for _, c := range te.Clubs {
			for _, p := range c.Squad {
				if p.PlayerID == playerID {
					seller = c
					break
				}
			}
			if seller != nil {
				break
			}
		}
	}
	if buyer == nil || seller == nil || buyer.ClubID == seller.ClubID {
		return nil
	}

	var player *models.Player
	for _, p := range seller.Squad {
		if p.PlayerID == playerID {
			player = p
			break
		}
	}
	if player == nil {
		return nil
	}

	// Wonderkid constraint: canonical prodigies only transfer between 12 Super League clubs
	if (player.UniverseWonderkid || strings.HasPrefix(player.PlayerID, "WK_")) && (!isSuperLeagueClub(buyer.ClubID) || !isSuperLeagueClub(seller.ClubID)) {
		return nil
	}

	for _, active := range te.ActiveNegotiations {
		if active != nil && active.Player != nil && active.Player.PlayerID == player.PlayerID {
			return active
		}
	}

	mult := 1.10
	if player.UniverseWonderkid {
		mult = 1.30
	}
	initialBid := int64(float64(player.MarketValueEUR) * mult)
	cap := int64(float64(models.BaselineValue(player.OVR, player.Age, player.UniverseWonderkid)) * 2.5)
	if cap > 0 && initialBid > cap {
		initialBid = cap
	}
	if initialBid < 1 {
		initialBid = models.BaselineValue(player.OVR, player.Age, player.UniverseWonderkid)
	}

	// Warchest check: buyer must afford initial bid
	if buyerMgr, ok := te.Managers[buyerID]; ok && buyerMgr != nil {
		if buyerMgr.BudgetEur < initialBid {
			return nil
		}
	}

	neg := &TransferNegotiation{
		NegotiationID:    fmt.Sprintf("NEG_%s_%s_%d", buyer.ClubID, player.PlayerID, te.CurrentDay),
		Player:           player,
		Buyer:            buyer,
		Seller:           seller,
		CurrentBid:       initialBid,
		AskingPrice:      initialBid,
		CreatedMatchweek: te.CurrentMatchweek,
		StageIndex:       1,
		StageName:        "INQUIRY",
		ProgressPct:      20,
		IsWonderkid:      player.UniverseWonderkid,
		History:          []string{fmt.Sprintf("Initial bid: %s", models.FormatCurrency(initialBid))},
	}
	te.ActiveNegotiations = append([]*TransferNegotiation{neg}, te.ActiveNegotiations...)
	weekOrMw := fmt.Sprintf("Week %d", te.CurrentWeek)
	if te.CurrentWeek <= 1 && te.CurrentMatchweek > 1 {
		weekOrMw = fmt.Sprintf("MW %d", te.CurrentMatchweek)
	}
	te.prependFeed(TransferFeedItem{
		Headline:    fmt.Sprintf("EXCLUSIVE: %s submit official offer to %s for %s worth %s!", buyer.ClubName, seller.ClubName, player.FullName, models.FormatCurrency(initialBid)),
		Category:    "EXCLUSIVE",
		IsWonderkid: player.UniverseWonderkid,
		Matchweek:   te.CurrentMatchweek,
		Timestamp:   weekOrMw,
	})
	return neg
}
