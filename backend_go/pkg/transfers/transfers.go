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

const (
	TransferWindowWeeks          = 12
	transferIncomeReinvestmentPct = 100
	minimumActiveWarchest         = int64(5_000_000)
)

// TransferFeedItem represents a breaking news wire item.
type TransferFeedItem struct {
	Headline    string `json:"headline"`
	Category    string `json:"category"`
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
	StageIndex       int            `json:"stage_index"`
	StageName        string         `json:"stage_name"`
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

func isSuperLeagueClub(clubID string) bool { return superLeagueClubIDs[clubID] }
func isCanonicalWonderkid(p *models.Player) bool {
	return p != nil && (p.UniverseWonderkid || strings.HasPrefix(p.PlayerID, "WK_"))
}

// CalculateClubWarchest is retained for compatibility with existing callers.
// New club-owned budgets are established through calculateClubWindowBudget.
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

func calculateClubWindowBudget(club *models.Club) int64 {
	if club == nil {
		return 0
	}
	id := club.Identity.Clamp()
	millions := int64(20 + (id.FinancialPower*13)/10 + (id.Reputation*3)/10)
	if millions < 25 {
		millions = 25
	}
	if millions > 180 {
		millions = 180
	}
	budget := millions * models.EuroMillion
	if club.Finances.Balance > 0 && budget > club.Finances.Balance {
		budget = club.Finances.Balance
	}
	return budget
}

func (te *TransferEngine) syncManagerBudget(clubID string) {
	club := te.Clubs[clubID]
	mgr := te.Managers[clubID]
	if club != nil && mgr != nil {
		mgr.BudgetEur = club.Finances.TransferBudget
	}
}

func canAfford(club *models.Club, fee int64) bool {
	return club != nil && fee > 0 && club.Finances.TransferBudget >= fee && club.Finances.Balance >= fee
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

func NewTransferEngine(clubs []*models.Club, mgrs map[string]*managers.ManagerProfile, seed int64) *TransferEngine {
	if seed == 0 {
		seed = 20260907
	}
	clubsMap := make(map[string]*models.Club)
	for _, c := range clubs {
		if c == nil {
			continue
		}
		if c.Identity.IsZero() {
			c.Identity = models.DefaultClubIdentity(c.ClubID, c.OverallTeamRating)
			if c.Finances == (models.ClubFinances{}) {
				c.Finances = models.InitialClubFinances(c.Identity)
			}
		}
		clubsMap[c.ClubID] = c
	}
	if mgrs == nil {
		mgrs = managers.BuildManagers(clubs)
	}
	te := &TransferEngine{
		Clubs: clubsMap, Managers: mgrs, CurrentMatchweek: 1, CurrentDay: 1, CurrentWeek: 1,
		TransferredThisWindow: make(map[string]bool), ActiveNegotiations: []*TransferNegotiation{},
		TransferFeed: []TransferFeedItem{}, CompletedTransfers: []CompletedTransfer{}, AllTimeTransfers: []CompletedTransfer{},
		RNG: rand.New(rand.NewSource(seed)),
	}
	for cid := range clubsMap {
		te.syncManagerBudget(cid)
	}
	te.injectInitialRumors()
	return te
}

func (te *TransferEngine) injectInitialRumors() {
	te.TransferFeed = append(te.TransferFeed,
		TransferFeedItem{Headline: "The offseason transfer market will open after the final matchweek.", Category: "EXCLUSIVE", Matchweek: 1, Timestamp: "MW 1"},
		TransferFeedItem{Headline: "Scouts continue monitoring the league's elite academy prospects", Category: "RUMOR", IsWonderkid: true, Matchweek: 1, Timestamp: "MW 1"},
	)
}

func (te *TransferEngine) IsWindowOpen() bool {
	return te.IsOffSeason && te.CurrentWeek >= 1 && te.CurrentWeek <= TransferWindowWeeks
}

func (te *TransferEngine) GetWindowName() string {
	if te.IsWindowOpen() {
		return fmt.Sprintf("Summer Window (Open - Week %d of %d)", te.CurrentWeek, TransferWindowWeeks)
	}
	if te.IsOffSeason && te.CurrentWeek > TransferWindowWeeks {
		return "Transfer Window Complete"
	}
	return "Window Closed (Opens after season completion)"
}

func (te *TransferEngine) RevertToBaselines(pull ...float64) {
	te.mu.Lock()
	defer te.mu.Unlock()
	p := 0.08
	if len(pull) > 0 && pull[0] > 0 {
		p = pull[0]
	}
	for _, club := range te.Clubs {
		if club == nil { continue }
		for _, pl := range club.Squad {
			if pl == nil || pl.MarketValueEUR <= 0 { continue }
			anchor := models.BaselineValue(pl.OVR, pl.Age, pl.UniverseWonderkid)
			pl.MarketValueEUR += int64(float64(anchor-pl.MarketValueEUR) * p)
			models.ClampPlayer(pl)
		}
	}
}

// BeginOffSeasonWindow is the single boundary that opens a new transfer window.
// Eligibility markers reset here—not weekly, monthly, on transfer, or at new season.
func (te *TransferEngine) BeginOffSeasonWindow() {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.IsOffSeason = true
	te.CurrentWeek = 1
	te.CurrentDay = 1
	te.ActiveNegotiations = te.ActiveNegotiations[:0]
	te.CompletedTransfers = te.CompletedTransfers[:0]
	te.TransferredThisWindow = make(map[string]bool)
	for cid, club := range te.Clubs {
		if club == nil { continue }
		club.Finances.TransferBudget = calculateClubWindowBudget(club)
		if club.Finances.TransferBudget < 0 { club.Finances.TransferBudget = 0 }
		te.syncManagerBudget(cid)
	}
	te.prependFeed(TransferFeedItem{Headline: "Summer transfer window opens: Week 1 of 12.", Category: "EXCLUSIVE", Timestamp: "Week 1"})
}

// ResetForNewSeason closes the completed offseason but deliberately preserves
// the just-finished window's transfer markers and club finances. Those markers
// reset only when BeginOffSeasonWindow starts the next window.
func (te *TransferEngine) ResetForNewSeason() {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.ActiveNegotiations = te.ActiveNegotiations[:0]
	te.CompletedTransfers = te.CompletedTransfers[:0]
	te.CurrentDay = 1
	te.CurrentMatchweek = 1
	te.IsOffSeason = false
	for cid := range te.Clubs { te.syncManagerBudget(cid) }
	te.prependFeed(TransferFeedItem{Headline: "New season squads confirmed. The transfer market is closed until season completion.", Category: "EXCLUSIVE", Matchweek: 1, Timestamp: "MW 1"})
}

func (te *TransferEngine) UpdateDailyMarket() {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.updateDailyMarketUnlocked(te.IsWindowOpen())
}

func (te *TransferEngine) AdvanceOpenWindow() {
	te.mu.Lock()
	defer te.mu.Unlock()
	if !te.IsOffSeason || te.CurrentWeek > TransferWindowWeeks { return }
	te.advanceWeeklyMarketUnlocked()
}

func (te *TransferEngine) advanceWeeklyMarketUnlocked() {
	var surviving []*TransferNegotiation
	for _, neg := range te.ActiveNegotiations {
		if !te.progressNegotiation(neg) { surviving = append(surviving, neg) }
	}
	te.ActiveNegotiations = surviving
	numBids := 2 + te.RNG.Intn(3)
	for i := 0; i < numBids; i++ { te.aiInitiateBid() }
	te.CurrentWeek++
	te.CurrentDay++
}

func (te *TransferEngine) updateDailyMarketUnlocked(windowOpen bool) {
	var surviving []*TransferNegotiation
	for _, neg := range te.ActiveNegotiations {
		if !te.progressNegotiation(neg) { surviving = append(surviving, neg) }
	}
	te.ActiveNegotiations = surviving
	if windowOpen {
		for i, n := 0, 1+te.RNG.Intn(2); i < n; i++ { te.aiInitiateBid() }
	}
	te.CurrentDay++
}

func (te *TransferEngine) progressNegotiation(neg *TransferNegotiation) bool {
	if neg == nil || neg.Player == nil || neg.Seller == nil || neg.Buyer == nil { return true }
	if te.TransferredThisWindow[neg.Player.PlayerID] { neg.StageName = "COLLAPSED"; return true }
	found := false
	for _, p := range neg.Seller.Squad { if p != nil && p.PlayerID == neg.Player.PlayerID { found = true; break } }
	if !found { neg.StageName = "COLLAPSED"; return true }
	if isCanonicalWonderkid(neg.Player) && !isSuperLeagueClub(neg.Buyer.ClubID) { neg.StageName = "COLLAPSED"; return true }
	if !canAfford(neg.Buyer, neg.CurrentBid) {
		neg.StageName = "COLLAPSED"
		te.prependFeed(TransferFeedItem{Headline: fmt.Sprintf("DEAL COLLAPSED: %s cannot fund %s move for %s due to budget limits", neg.Buyer.ShortName, models.FormatCurrency(neg.CurrentBid), neg.Player.FullName), Category: "REJECTED", IsWonderkid: neg.IsWonderkid, Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek)})
		return true
	}

	neg.StageIndex++
	switch neg.StageIndex {
	case 2:
		neg.StageName, neg.ProgressPct = "COUNTER_OFFER", 40
		newBid := neg.CurrentBid + int64(float64(neg.CurrentBid)*(0.10+te.RNG.Float64()*0.15))
		available := neg.Buyer.Finances.TransferBudget
		if neg.Buyer.Finances.Balance < available { available = neg.Buyer.Finances.Balance }
		if newBid > available { newBid = available }
		if newBid <= 0 { neg.StageName = "COLLAPSED"; return true }
		neg.AskingPrice, neg.CurrentBid = newBid, newBid
		te.prependFeed(TransferFeedItem{Headline: fmt.Sprintf("%s submit improved bid of %s for %s (%s)", neg.Buyer.ShortName, models.FormatCurrency(newBid), neg.Player.FullName, neg.Seller.ShortName), Category: "TWIST", IsWonderkid: neg.IsWonderkid, Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek)})
		return false
	case 3:
		neg.StageName, neg.ProgressPct = "HIJACK_CHECK", 60
		if (neg.Player.OVR >= 82 || neg.IsWonderkid) && te.RNG.Float64() < 0.15 {
			hijackCost := int64(float64(neg.CurrentBid) * 1.15)
			var rivals []*models.Club
			for _, c := range te.Clubs {
				if c == nil || c.ClubID == neg.Buyer.ClubID || c.ClubID == neg.Seller.ClubID { continue }
				if isCanonicalWonderkid(neg.Player) && !isSuperLeagueClub(c.ClubID) { continue }
				if canAfford(c, hijackCost) { rivals = append(rivals, c) }
			}
			if len(rivals) > 0 {
				hijacker := rivals[te.RNG.Intn(len(rivals))]
				neg.OriginalBuyer, neg.Buyer, neg.IsHijacked, neg.CurrentBid = neg.Buyer, hijacker, true, hijackCost
				te.prependFeed(TransferFeedItem{Headline: fmt.Sprintf("HIJACK TWIST! %s hijack %s deal for %s with late %s offer!", hijacker.ShortName, neg.OriginalBuyer.ShortName, neg.Player.FullName, models.FormatCurrency(hijackCost)), Category: "HIJACK", IsWonderkid: neg.IsWonderkid, Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek)})
			}
		}
		return false
	case 4:
		neg.StageName, neg.ProgressPct = "TERMS_MEDICAL", 80
		te.prependFeed(TransferFeedItem{Headline: fmt.Sprintf("Medical booked: %s arrives at %s training ground ahead of final signature", neg.Player.FullName, neg.Buyer.ShortName), Category: "EXCLUSIVE", IsWonderkid: neg.IsWonderkid, Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek)})
		return false
	case 5:
		if !te.executeTransfer(neg) { neg.StageName = "COLLAPSED"; return true }
		neg.StageName, neg.ProgressPct = "COMPLETED", 100
		return true
	default:
		return true
	}
}

// executeTransfer performs final backend commit validation and only mutates
// transfer eligibility/finances after every invariant has passed.
func (te *TransferEngine) executeTransfer(neg *TransferNegotiation) bool {
	if neg == nil || neg.Player == nil || neg.Seller == nil || neg.Buyer == nil || neg.CurrentBid <= 0 { return false }
	p := neg.Player
	if te.TransferredThisWindow[p.PlayerID] || neg.Seller.ClubID == neg.Buyer.ClubID { return false }
	if isCanonicalWonderkid(p) && !isSuperLeagueClub(neg.Buyer.ClubID) { return false }
	if !canAfford(neg.Buyer, neg.CurrentBid) { return false }
	found := false
	for _, sp := range neg.Seller.Squad { if sp != nil && sp.PlayerID == p.PlayerID { found = true; break } }
	if !found { return false }
	for _, bp := range neg.Buyer.Squad { if bp != nil && bp.PlayerID == p.PlayerID { return false } }

	newSellerSquad := make([]*models.Player, 0, len(neg.Seller.Squad)-1)
	for _, sp := range neg.Seller.Squad { if sp != nil && sp.PlayerID != p.PlayerID { newSellerSquad = append(newSellerSquad, sp) } }
	neg.Seller.Squad = newSellerSquad
	p.ClubID = neg.Buyer.ClubID
	// OriginalClubID is historical metadata. Permanent transfers never rewrite it.
	neg.Buyer.Squad = append(neg.Buyer.Squad, p)

	neg.Buyer.Finances.TransferBudget -= neg.CurrentBid
	neg.Buyer.Finances.Balance -= neg.CurrentBid
	neg.Seller.Finances.Balance += neg.CurrentBid
	reinvest := neg.CurrentBid * transferIncomeReinvestmentPct / 100
	neg.Seller.Finances.TransferBudget += reinvest
	if neg.Seller.Finances.TransferBudget > neg.Seller.Finances.Balance { neg.Seller.Finances.TransferBudget = neg.Seller.Finances.Balance }
	te.syncManagerBudget(neg.Buyer.ClubID)
	te.syncManagerBudget(neg.Seller.ClubID)
	te.TransferredThisWindow[p.PlayerID] = true
	neg.Seller.RecalculateRatings()
	neg.Buyer.RecalculateRatings()

	completed := CompletedTransfer{PlayerID: p.PlayerID, PlayerName: p.FullName, PlayerPos: p.Position, PlayerOVR: p.OVR, IsWonderkid: neg.IsWonderkid, SellerID: neg.Seller.ClubID, SellerName: neg.Seller.ClubName, SellerShort: neg.Seller.ShortName, BuyerID: neg.Buyer.ClubID, BuyerName: neg.Buyer.ClubName, BuyerShort: neg.Buyer.ShortName, FeeEUR: neg.CurrentBid, FormattedFee: models.FormatCurrency(neg.CurrentBid), Matchweek: te.CurrentMatchweek}
	te.CompletedTransfers = append(te.CompletedTransfers, completed)
	te.AllTimeTransfers = append(te.AllTimeTransfers, completed)
	te.prependFeed(TransferFeedItem{Headline: fmt.Sprintf("HERE WE GO: %s signs for %s in %s deal from %s!", p.FullName, neg.Buyer.ShortName, models.FormatCurrency(neg.CurrentBid), neg.Seller.ShortName), Category: "HERE_WE_GO", IsWonderkid: neg.IsWonderkid, Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek)})
	return true
}

func (te *TransferEngine) aiInitiateBid() {
	var clubs []*models.Club
	for _, c := range te.Clubs { clubs = append(clubs, c) }
	if len(clubs) < 2 { return }
	var solvent []*models.Club
	for _, c := range clubs { if canAfford(c, minimumActiveWarchest) { solvent = append(solvent, c) } }
	if len(solvent) == 0 { return }
	buyer := solvent[te.RNG.Intn(len(solvent))]
	var sellers []*models.Club
	for _, c := range clubs { if c.ClubID != buyer.ClubID && len(c.Squad) > 15 { sellers = append(sellers, c) } }
	if len(sellers) == 0 { return }
	seller := sellers[te.RNG.Intn(len(sellers))]

	var validTargets []*models.Player
	for _, p := range seller.Squad {
		if p == nil || te.TransferredThisWindow[p.PlayerID] { continue }
		alreadyNeg := false
		for _, n := range te.ActiveNegotiations { if n != nil && n.Player != nil && n.Player.PlayerID == p.PlayerID { alreadyNeg = true; break } }
		if alreadyNeg { continue }
		if isCanonicalWonderkid(p) && (!isSuperLeagueClub(buyer.ClubID) || !isSuperLeagueClub(seller.ClubID)) { continue }
		baseVal := models.BaselineValue(p.OVR, p.Age, p.UniverseWonderkid)
		estBid := int64(float64(baseVal) * 1.10)
		if !canAfford(buyer, estBid) { continue }
		if p.OVR >= 76 || isCanonicalWonderkid(p) { validTargets = append(validTargets, p) }
	}
	if len(validTargets) == 0 { return }
	target := validTargets[te.RNG.Intn(len(validTargets))]
	baseVal := models.BaselineValue(target.OVR, target.Age, target.UniverseWonderkid)
	initialBid := int64(float64(baseVal) * (0.95 + te.RNG.Float64()*0.20))
	available := buyer.Finances.TransferBudget
	if buyer.Finances.Balance < available { available = buyer.Finances.Balance }
	if initialBid > available { initialBid = available }
	if initialBid <= 0 { return }
	neg := &TransferNegotiation{NegotiationID: fmt.Sprintf("NEG_%s_%s_%d", buyer.ClubID, target.PlayerID, te.CurrentDay), Player: target, Buyer: buyer, Seller: seller, CurrentBid: initialBid, AskingPrice: initialBid, CreatedMatchweek: te.CurrentMatchweek, StageIndex: 1, StageName: "INQUIRY", ProgressPct: 20, IsWonderkid: isCanonicalWonderkid(target), History: []string{fmt.Sprintf("Initial bid: %s", models.FormatCurrency(initialBid))}}
	te.ActiveNegotiations = append(te.ActiveNegotiations, neg)
	te.prependFeed(TransferFeedItem{Headline: fmt.Sprintf("%s open talks with %s for %s with opening %s bid", buyer.ShortName, seller.ShortName, target.FullName, models.FormatCurrency(initialBid)), Category: "EXCLUSIVE", IsWonderkid: neg.IsWonderkid, Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek)})
}

func (te *TransferEngine) GetTransferRecords() TransferRecordsData {
	te.mu.RLock(); defer te.mu.RUnlock()
	all := append([]CompletedTransfer(nil), te.AllTimeTransfers...)
	sort.Slice(all, func(i, j int) bool { return all[i].FeeEUR > all[j].FeeEUR })
	top := all; if len(top) > 10 { top = top[:10] }
	netSpend := make(map[string]ClubNetSpend)
	for _, club := range te.Clubs { netSpend[club.ClubID] = ClubNetSpend{ClubID: club.ClubID, ClubName: club.ClubName, ShortName: club.ShortName, PrimaryColor: club.PrimaryColor} }
	for _, tr := range te.AllTimeTransfers {
		if b, ok := netSpend[tr.BuyerID]; ok { b.Spent += tr.FeeEUR; b.Net -= tr.FeeEUR; netSpend[tr.BuyerID] = b }
		if s, ok := netSpend[tr.SellerID]; ok { s.Received += tr.FeeEUR; s.Net += tr.FeeEUR; netSpend[tr.SellerID] = s }
	}
	return TransferRecordsData{TopSignings: top, NetSpend: netSpend, TotalTransfersCount: len(te.AllTimeTransfers)}
}

func (te *TransferEngine) InitiateBid(playerID string, buyerID string, bidAmount int64) *TransferNegotiation {
	te.mu.Lock(); defer te.mu.Unlock()
	if !te.IsWindowOpen() || te.TransferredThisWindow[playerID] { return nil }
	buyer := te.Clubs[buyerID]; if buyer == nil { return nil }
	var target *models.Player; var seller *models.Club
	for _, c := range te.Clubs {
		for _, p := range c.Squad { if p != nil && p.PlayerID == playerID { target, seller = p, c; break } }
		if target != nil { break }
	}
	if target == nil || seller == nil || seller.ClubID == buyerID { return nil }
	if isCanonicalWonderkid(target) && (!isSuperLeagueClub(buyer.ClubID) || !isSuperLeagueClub(seller.ClubID)) { return nil }
	if bidAmount <= 0 { bidAmount = int64(float64(models.BaselineValue(target.OVR, target.Age, target.UniverseWonderkid)) * 1.05) }
	if !canAfford(buyer, bidAmount) { return nil }
	neg := &TransferNegotiation{NegotiationID: fmt.Sprintf("NEG_%s_%s_%d", buyer.ClubID, target.PlayerID, te.CurrentDay), Player: target, Buyer: buyer, Seller: seller, CurrentBid: bidAmount, AskingPrice: bidAmount, CreatedMatchweek: te.CurrentMatchweek, StageIndex: 1, StageName: "INQUIRY", ProgressPct: 20, IsWonderkid: isCanonicalWonderkid(target), History: []string{fmt.Sprintf("Initial bid: %s", models.FormatCurrency(bidAmount))}}
	te.ActiveNegotiations = append(te.ActiveNegotiations, neg)
	te.prependFeed(TransferFeedItem{Headline: fmt.Sprintf("%s open talks with %s for %s with opening %s bid", buyer.ShortName, seller.ShortName, target.FullName, models.FormatCurrency(bidAmount)), Category: "EXCLUSIVE", IsWonderkid: neg.IsWonderkid, Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek)})
	return neg
}

func (te *TransferEngine) prependFeed(item TransferFeedItem) {
	te.TransferFeed = append([]TransferFeedItem{item}, te.TransferFeed...)
	if len(te.TransferFeed) > 120 { te.TransferFeed = te.TransferFeed[:120] }
}

func (te *TransferEngine) TriggerSpecificBid(buyerID, sellerID, playerID string) *TransferNegotiation {
	te.mu.Lock(); defer te.mu.Unlock()
	if !te.IsWindowOpen() || te.TransferredThisWindow[playerID] { return nil }
	buyer, seller := te.Clubs[buyerID], te.Clubs[sellerID]
	if buyer == nil { return nil }
	if seller == nil && playerID != "" {
		for _, c := range te.Clubs { for _, p := range c.Squad { if p != nil && p.PlayerID == playerID { seller = c; break } }; if seller != nil { break } }
	}
	if seller == nil || buyer.ClubID == seller.ClubID { return nil }
	var player *models.Player
	for _, p := range seller.Squad { if p != nil && p.PlayerID == playerID { player = p; break } }
	if player == nil { return nil }
	if isCanonicalWonderkid(player) && (!isSuperLeagueClub(buyer.ClubID) || !isSuperLeagueClub(seller.ClubID)) { return nil }
	for _, active := range te.ActiveNegotiations { if active != nil && active.Player != nil && active.Player.PlayerID == player.PlayerID { return active } }
	mult := 1.10; if isCanonicalWonderkid(player) { mult = 1.30 }
	initialBid := int64(float64(player.MarketValueEUR) * mult)
	cap := int64(float64(models.BaselineValue(player.OVR, player.Age, player.UniverseWonderkid)) * 2.5)
	if cap > 0 && initialBid > cap { initialBid = cap }
	if initialBid < 1 { initialBid = models.BaselineValue(player.OVR, player.Age, player.UniverseWonderkid) }
	if !canAfford(buyer, initialBid) { return nil }
	neg := &TransferNegotiation{NegotiationID: fmt.Sprintf("NEG_%s_%s_%d", buyer.ClubID, player.PlayerID, te.CurrentDay), Player: player, Buyer: buyer, Seller: seller, CurrentBid: initialBid, AskingPrice: initialBid, CreatedMatchweek: te.CurrentMatchweek, StageIndex: 1, StageName: "INQUIRY", ProgressPct: 20, IsWonderkid: isCanonicalWonderkid(player), History: []string{fmt.Sprintf("Initial bid: %s", models.FormatCurrency(initialBid))}}
	te.ActiveNegotiations = append([]*TransferNegotiation{neg}, te.ActiveNegotiations...)
	te.prependFeed(TransferFeedItem{Headline: fmt.Sprintf("EXCLUSIVE: %s submit official offer to %s for %s worth %s!", buyer.ClubName, seller.ClubName, player.FullName, models.FormatCurrency(initialBid)), Category: "EXCLUSIVE", IsWonderkid: neg.IsWonderkid, Matchweek: te.CurrentMatchweek, Timestamp: fmt.Sprintf("Week %d", te.CurrentWeek)})
	return neg
}
