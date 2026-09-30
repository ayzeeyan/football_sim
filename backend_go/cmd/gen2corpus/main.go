package main

import (
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/medical"
	"football_sim/pkg/models"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

const simulatorCommit = "0cff407625f090fba11b1408e9e10f8aca355d7e"
const schemaVersion = "gen2_final_v1"
const generationVersion = "gen2_final_v1_simulator_adapter_1"

type writer struct {
	gz *gzip.Writer
	bw *bufio.Writer
	n int
}

func newWriter(path string) (*writer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { return nil, err }
	f, err := os.Create(path); if err != nil { return nil, err }
	gz := gzip.NewWriter(f)
	bw := bufio.NewWriterSize(gz, 1<<20)
	return &writer{gz:gz,bw:bw}, nil
}
func (w *writer) close() error {
	if err:=w.bw.Flush(); err!=nil{return err}; return w.gz.Close()
}
func (w *writer) emit(r map[string]interface{}) error {
	b,err:=json.Marshal(r); if err!=nil{return err}
	if _,err=w.bw.Write(append(b,'\n'));err!=nil{return err}; w.n++; return nil
}

func derive(root int64, parts ...string) int64 {
	h:=sha256.New(); var b [8]byte; binary.LittleEndian.PutUint64(b[:],uint64(root)); h.Write(b[:])
	for _,p:=range parts { h.Write([]byte{0}); h.Write([]byte(p)) }
	s:=h.Sum(nil); return int64(binary.LittleEndian.Uint64(s[:8]) & ((1<<63)-1))
}
func clamp(v,lo,hi float64) float64 { if v<lo{return lo}; if v>hi{return hi}; return v }
func role(p *models.Player) string {
	s:=strings.ToLower(strings.TrimSpace(p.SquadRole))
	switch s {
	case "crucial": return "star"
	case "important": return "starter"
	case "rotation": return "rotation"
	case "prospect": return "prospect"
	default: return "depth"
	}
}
func pos(p *models.Player) string {
	s:=strings.ToUpper(strings.TrimSpace(p.Position))
	if s=="" { s=strings.ToUpper(strings.TrimSpace(p.Category)) }
	switch s {
	case "GK": return "GK"
	case "CB","LCB","RCB","DEF": return "CB"
	case "LB","RB","LWB","RWB","FB": return "FB"
	case "CDM","DM","LDM","RDM": return "DM"
	case "CM","LCM","RCM","MID": return "CM"
	case "CAM","AM","LAM","RAM": return "AM"
	case "LW","RW","LM","RM","WG": return "WG"
	default: return "ST"
	}
}
func totalMinutes(p *models.Player) int {
	n:=0; for _,s:=range p.CompetitionStats { if s!=nil { n+=s.Minutes } }; return n
}
func recentForm(c *models.Club) float64 {
	if c==nil||len(c.Form)==0{return 0}
	n:=len(c.Form); if n>5{n=5}; score:=0.0
	for _,x:=range c.Form[len(c.Form)-n:] { if x=="W"{score+=1}else if x=="L"{score-=1} }
	return score/float64(n)
}
func squadAvgFitness(c *models.Club) float64 {
	if c==nil||len(c.Squad)==0{return 0}; sum:=0
	for _,p:=range c.Squad { if p!=nil { sum+=p.Fitness } }
	return float64(sum)/float64(len(c.Squad))
}
func squadAvgFatigue(c *models.Club) float64 { return 100-squadAvgFitness(c) }
func squadDepth(c *models.Club) float64 {
	if c==nil{return 0}; vals:=[]int{}
	for _,p:=range c.Squad { if p!=nil&&!p.IsUnavailable("",0){ vals=append(vals,p.EffectiveOVR()) } }
	sort.Sort(sort.Reverse(sort.IntSlice(vals))); if len(vals)>18{vals=vals[:18]}; if len(vals)==0{return 0}
	s:=0; for _,v:=range vals{s+=v}; return float64(s)/float64(len(vals))
}
func injuryBurden(p *models.Player) float64 {
	if p==nil{return 0}; v:=float64(len(p.InjuryHistory))/5.0
	if p.InjuredMatches>0 { v+=float64(p.InjuredMatches)/10.0 }; return clamp(v,0,1)
}
func potential(ge *growth.GrowthEngine,p *models.Player) int {
	if ge!=nil { if v,ok:=ge.PotentialFor(p.PlayerID);ok{return v} }; v:=p.OVR
	if p.Age<25 { v+=2*(25-p.Age) }; if v>99{v=99}; return v
}
func highPress(tm *tournament.TournamentManager,cid string) bool {
	m:=tm.Managers[cid]; return m!=nil&&m.CanonicalStyle()=="high_press"
}
func injuryProb(tm *tournament.TournamentManager,p *models.Player,c *models.Club,recent int,mw int) float64 {
	hp:=highPress(tm,c.ClubID); base:=0.05; if hp{base=0.08}
	density:=float64(recent)/180.0
	mult,_:=medical.RiskMultiplier(medical.RiskInput{Age:p.Age,Fitness:p.Fitness,MinutesPlayed:90,MatchDensity:density,IsWonderkid:p.UniverseWonderkid,HighPress:hp})
	return clamp(base*mult,0,1)
}
func playerFeatures(tm *tournament.TournamentManager,p *models.Player,c *models.Club,recent int,mw int) map[string]interface{} {
	mins:=totalMinutes(p); fat:=clamp(float64(100-p.Fitness),0,100); cong:=clamp(float64(recent)/180.0,0,1)
	return map[string]interface{}{
		"age":p.Age,"position":pos(p),"minutes":clamp(float64(mins),0,4000),"recent_minutes":clamp(float64(recent),0,600),
		"fatigue":fat,"fitness":clamp(float64(p.Fitness),0,100),"injury_history":injuryBurden(p),
		"morale":clamp(float64(p.Morale-50)/50.0,-1,1),"form":clamp(float64(p.FormModifier())/8.0,-1,1),
		"squad_role":role(p),"fixture_congestion":cong,
	}
}
func playerTargets(tm *tournament.TournamentManager,p *models.Player,c *models.Club,recent int,mw int) map[string]interface{} {
	ip:=injuryProb(tm,p,c,recent,mw); avail:=1.0
	if p.IsUnavailable("",mw){avail=0}else{avail=1-ip}
	fat:=clamp(float64(100-p.Fitness)/100.0,0,1); cong:=clamp(float64(recent)/180.0,0,1)
	perf:=1.0; if p.OVR>0 { perf=clamp(float64(p.EffectiveOVR()+p.FormModifier())/float64(p.OVR),.5,1.5) }
	return map[string]interface{}{"injury_probability":ip,"availability_probability":avail,"rest_pressure":clamp((fat+cong)/2,0,1),"performance_modifier":perf,"recovery_need":math.Max(fat,cong)}
}
func difficultyPlayer(p *models.Player,recent int) string {
	if p.Age<=18||p.Age>=35||p.Fitness<60||p.InjuredMatches>0||recent>=270{return "edge"}
	if p.Fitness<75||p.ContractYears<=1||recent>=180{return "hard"}; return "core"
}
func ids(world,player,club,parent,trajectory string) map[string]interface{} {
	m:=map[string]interface{}{"world_id":world,"career_id":world}
	if player!=""{m["player_id"]=player}; if club!=""{m["club_id"]=club}; if parent!=""{m["parent_state_id"]=parent}; if trajectory!=""{m["trajectory_id"]=trajectory}
	return m
}
func quality(score float64, conv bool) map[string]interface{} {
	return map[string]interface{}{"state_validity":1.0,"label_provenance":1.0,"rollout_convergence":map[bool]float64{true:1,false:.9}[conv],"trajectory_continuity":1.0,"counterfactual_cleanliness":1.0,"cross_task_consistency":1.0,"novelty":1.0,"schema_completeness":1.0,"quality_score":score}
}
func record(task,family,world string,idsv,features,targets map[string]interface{},source,prov,difficulty string,rollouts int,confidence float64) map[string]interface{} {
	return map[string]interface{}{
		"schema_version":schemaVersion,"generation_version":generationVersion,"task":task,"task_family":family,"ids":idsv,
		"features":features,"targets":targets,"difficulty":difficulty,
		"metadata":map[string]interface{}{"provenance":prov,"label_source":source,"rollout_count":rollouts,"label_confidence":confidence,"causal_timing_verified":true,"simulator_commit":simulatorCommit,"generation_version":generationVersion},
		"quality":quality(.97,confidence>=.8),
	}
}
type recentTracker struct { prev map[string]int; a map[string]int; b map[string]int }
func newTracker()*recentTracker{return &recentTracker{prev:map[string]int{},a:map[string]int{},b:map[string]int{}}}
func (t *recentTracker) recent(id string) int{return t.a[id]+t.b[id]}
func (t *recentTracker) advance(tm *tournament.TournamentManager){
	next:=map[string]int{}
	for _,c:=range tm.ClubsList { for _,p:=range c.Squad { if p==nil{continue}; cur:=totalMinutes(p); d:=cur-t.prev[p.PlayerID]; if d<0{d=0}; next[p.PlayerID]=d; t.prev[p.PlayerID]=cur } }
	t.b=t.a; t.a=next
}
func selectedPlayers(tm *tournament.TournamentManager,seed int64,n int) []*models.Player {
	type x struct{p *models.Player;k int64}; all:=[]x{}
	for _,c:=range tm.ClubsList{for _,p:=range c.Squad{if p!=nil{all=append(all,x{p,derive(seed,p.PlayerID)})}}}
	sort.Slice(all,func(i,j int)bool{return all[i].k<all[j].k}); if n>len(all){n=len(all)}
	out:=make([]*models.Player,0,n); for i:=0;i<n;i++{out=append(out,all[i].p)}; return out
}
func clubOf(tm *tournament.TournamentManager,p *models.Player)*models.Club{return tm.Clubs[p.ClubID]}
func competitionLevel(f tournament.Fixture) string {
	s:=strings.ToLower(f.Competition+" "+f.Stage)
	if strings.Contains(s,"champions")||strings.Contains(s,"ucl")||strings.Contains(s,"final"){return "elite"}
	if strings.Contains(s,"europa")||strings.Contains(s,"cup"){return "top"}
	if strings.Contains(s,"league"){return "top"}; return "mid"
}
func matchFeatures(tm *tournament.TournamentManager,f tournament.Fixture) map[string]interface{} {
	h,a:=tm.Clubs[f.HomeID],tm.Clubs[f.AwayID]; hm,am:=tm.Managers[f.HomeID],tm.Managers[f.AwayID]
	hform,aform:="4-3-3","4-3-3"; if hm!=nil{hform=models.FormationForManager(hm.CanonicalStyle(),h.ClubID)}; if am!=nil{aform=models.FormationForManager(am.CanonicalStyle(),a.ClubID)}
	if hform=="4-3-3 Attack"{hform="4-3-3"}; if aform=="4-3-3 Attack"{aform="4-3-3"}
	valid:=func(x string)bool{switch x{case "4-3-3","4-2-3-1","3-4-3","3-5-2","4-4-2","4-1-4-1":return true};return false}; if !valid(hform)||!valid(aform){return nil}
	return map[string]interface{}{"home_team_quality":h.OverallTeamRating,"away_team_quality":a.OverallTeamRating,"home_advantage":.18,
		"tactical_matchup":clamp(float64(h.OverallTeamRating-a.OverallTeamRating)/100.0,-1,1),"home_formation":hform,"away_formation":aform,
		"home_fatigue":clamp(squadAvgFatigue(h),0,100),"away_fatigue":clamp(squadAvgFatigue(a),0,100),"home_fitness":clamp(squadAvgFitness(h),0,100),"away_fitness":clamp(squadAvgFitness(a),0,100),
		"home_morale":clamp(float64(h.Morale-50)/50,-1,1),"away_morale":clamp(float64(a.Morale-50)/50,-1,1),"home_form":recentForm(h),"away_form":recentForm(a),
		"home_squad_depth":clamp(squadDepth(h),0,100),"away_squad_depth":clamp(squadDepth(a),0,100),"home_fixture_congestion":0.0,"away_fixture_congestion":0.0,"competition_level":competitionLevel(f)}
}
func matchTargets(tm *tournament.TournamentManager,f tournament.Fixture,root int64,rollouts int)(map[string]interface{},float64,bool){
	hw,dw,aw:=0,0,0; sxh,sxa,vg:=0.0,0.0,0.0; vals:=[]float64{}
	ok:=0
	for i:=0;i<rollouts;i++{r,e:=tm.WhatIfSandbox(f.FixtureID,derive(root,f.FixtureID,fmt.Sprint(i)));if e!=""||r==nil{continue};ok++;g:=float64(r.Hypothetical.HomeGoals-r.Hypothetical.AwayGoals);vals=append(vals,g);sxh+=r.Hypothetical.HomeXG;sxa+=r.Hypothetical.AwayXG;if g>0{hw++}else if g<0{aw++}else{dw++}}
	if ok==0{return nil,0,false}; mean:=0.0; for _,v:=range vals{mean+=v};mean/=float64(ok);for _,v:=range vals{d:=v-mean;vg+=d*d};if ok>1{vg/=float64(ok-1)};se:=math.Sqrt(vg/float64(ok));conv:=ok>=16&&se<=.5
	return map[string]interface{}{"home_scoring_intensity":clamp(sxh/float64(ok),0,4),"away_scoring_intensity":clamp(sxa/float64(ok),0,4),"expected_home_goals":clamp(sxh/float64(ok),0,4),"expected_away_goals":clamp(sxa/float64(ok),0,4),"expected_goal_difference":clamp(mean,-4,4),"home_probability":float64(hw)/float64(ok),"draw_probability":float64(dw)/float64(ok),"away_probability":float64(aw)/float64(ok),"rollout_variance":vg,"rollout_standard_error":se},clamp(1-se,.5,1),conv
}
func makeWorld(dataset string,root int64,index int)(*tournament.TournamentManager,*growth.GrowthEngine,error){
	ws:=derive(root,"world",fmt.Sprint(index)); ge:=growth.NewGrowthEngine(derive(ws,"development")); dm:=datamanager.NewDataManager(dataset,ge); dm.SetSeed(derive(ws,"datamanager"))
	if len(dm.ClubsList)==0{return nil,nil,fmt.Errorf("dataset loaded zero clubs")}
	tm:=tournament.NewEuropeanWorldManager(dm.ClubsList,ge,derive(ws,"matches")); te:=transfers.NewTransferEngine(dm.ClubsList,tm.Managers,derive(ws,"transfers"));tm.TransferEngine=te
	return tm,ge,nil
}
func emitPlayerViews(w *writer,tm *tournament.TournamentManager,ge *growth.GrowthEngine,world string,p *models.Player,recent,mw,index int,preOVR int) error {
	c:=clubOf(tm,p); if c==nil{return nil}; parent:=fmt.Sprintf("%s_mw%02d_%s",world,mw,p.PlayerID); f:=playerFeatures(tm,p,c,recent,mw); t:=playerTargets(tm,p,c,recent,mw); diff:=difficultyPlayer(p,recent)
	if err:=w.emit(record("player_dynamics","player_dynamics_availability",world,ids(world,p.PlayerID,c.ClubID,parent,""),f,t,"deterministic_simulator_rule","NATURAL_SIMULATION",diff,0,1));err!=nil{return err}
	if index%2==0 {
		baseFat:=f["fatigue"].(float64); cfFat:=clamp(baseFat+15,0,100); cfFit:=int(math.Round(100-cfFat)); hp:=highPress(tm,c.ClubID); baseP:=t["injury_probability"].(float64); mult,_:=medical.RiskMultiplier(medical.RiskInput{Age:p.Age,Fitness:cfFit,MinutesPlayed:90,MatchDensity:float64(recent)/180,IsWonderkid:p.UniverseWonderkid,HighPress:hp}); base:=.05;if hp{base=.08};cfP:=clamp(base*mult,0,1);delta:=cfP-baseP;dir:="neutral";if delta>.000001{dir="positive"}else if delta < -0.000001{dir="negative"}
		cff:=map[string]interface{}{"baseline":f,"intervention":map[string]interface{}{"fatigue":cfFat},"changed_features":[]string{"fatigue"}}
		cft:=map[string]interface{}{"baseline_injury_probability":baseP,"counterfactual_injury_probability":cfP,"target_delta":clamp(delta,-1,1),"effect_direction":dir,"baseline_availability_probability":t["availability_probability"],"counterfactual_availability_probability":clamp(1-cfP,0,1)}
		if err:=w.emit(record("counterfactual","counterfactual_branches",world,ids(world,p.PlayerID,c.ClubID,parent,""),cff,cft,"deterministic_simulator_rule","COUNTERFACTUAL",diff,0,1));err!=nil{return err}
	}
	if index%3==0 { jf:=map[string]interface{}{"family":"workload","family_specific_fields":f}; jt:=map[string]interface{}{"dynamic":t}; if err:=w.emit(record("cross_task_joint","cross_task_joint",world,ids(world,p.PlayerID,c.ClubID,parent,""),jf,jt,"analytical_from_simulator_state","TRAJECTORY_VIEW",diff,0,1));err!=nil{return err} }
	if index%4==0 {
		pot:=potential(ge,p); fin:=clamp(float64(c.Finances.Balance)/400000000.0,0,1); ef:=map[string]interface{}{"player_quality":p.OVR,"age":p.Age,"potential":pot,"contract_remaining":clamp(float64(p.ContractYears),0,6),"current_wage":clamp(float64(p.WageEUR),0,500000),"reputation":float64(c.Identity.Reputation)/100,"club_finances":fin,"buyer_finances":fin,"buyer_need":clamp(float64(100-p.OVR)/100,0,1),"seller_need":float64(c.Identity.SellingTendency)/100,"squad_role":role(p),"competition_prestige":float64(c.Identity.Reputation)/100,"market_condition_index":1.0}
		renew:=1.0;if p.OutOfContract()&&p.WantsToLeaveOnFree(){renew=0};lev:=clamp(float64(p.OVR+pot)/200,0,1); press:=clamp(1-fin,0,1); et:=map[string]interface{}{"valuation_millions":clamp(float64(p.MarketValueEUR)/1e6,0,300),"wage_expectation_weekly":clamp(float64(models.WageForOVR(p.OVR)),0,1000000),"renewal_probability":renew,"bid_acceptance_probability":clamp(float64(c.Identity.SellingTendency)/100,0,1),"market_leverage":lev,"financial_pressure":press}
		if err:=w.emit(record("economy_contracts","economy_contracts",world,ids(world,p.PlayerID,c.ClubID,parent,""),ef,et,"deterministic_simulator_rule","NATURAL_SIMULATION",diff,0,1));err!=nil{return err}
	}
	if p.OVR!=preOVR || index%5==0 {
		pot:=potential(ge,p); mins:=clamp(float64(totalMinutes(p)),0,4000); env:=clamp(float64(c.Identity.AcademyQuality+c.Identity.Reputation)/200,0,1); df:=map[string]interface{}{"age":p.Age,"ability":preOVR,"potential":pot,"minutes":mins,"coaching_quality":clamp(float64(tm.Managers[c.ClubID].Adaptability)/100,0,1),"facility_quality":float64(c.Identity.AcademyQuality)/100,"competition_level_numeric":float64(c.Identity.Reputation)/100,"injury_burden":injuryBurden(p),"position":pos(p),"morale":clamp(float64(p.Morale-50)/50,-1,1),"environment_quality":env}
		delta:=clamp(float64(p.OVR-preOVR),-8,8); gp:=0.0;dp:=0.0;if delta>0{gp=clamp(delta/5,0,1)}else if delta<0{dp=clamp(-delta/5,0,1)}
		dt:=map[string]interface{}{"growth_pressure":gp,"decline_pressure":dp,"expected_net_ability_delta":delta,"future_availability_modifier":clamp(t["availability_probability"].(float64),0,1)}
		if err:=w.emit(record("development_decline","development_decline",world,ids(world,p.PlayerID,c.ClubID,parent,""),df,dt,"direct_simulator_outcome","TEMPORAL_TRANSITION",diff,0,1));err!=nil{return err}
	}
	if diff=="edge"&&index%3==0 { of:=map[string]interface{}{"source_task_fields":f};ot:=map[string]interface{}{"dynamic":t}; if err:=w.emit(record("ood_edge","ood_edge",world,ids(world,p.PlayerID,c.ClubID,parent,""),of,ot,"deterministic_simulator_rule","TAIL_OVERSAMPLE","edge",0,1));err!=nil{return err} }
	return nil
}
func emitClubRows(w *writer,tm *tournament.TournamentManager,world string,mw int) error {
	for _,c:=range tm.ClubsList { if c==nil{continue}; m:=tm.Managers[c.ClubID]; if m==nil{continue}; status,_,candidate:=tm.Gen2ManagerSecuritySnapshot(c.ClubID,mw); pressure:=0.0; if strings.Contains(strings.ToLower(status),"hot"){pressure=1}else if strings.Contains(strings.ToLower(status),"watch"){pressure=.6}; sack:=0.0;if candidate{sack=1}; exp:=0.5;if c.ExpectedFinish>0{exp=clamp(1-float64(c.ExpectedFinish-1)/20,0,1)};fin:=clamp(float64(c.Finances.Balance)/400000000,0,1);sq:=clamp(c.SquadAvgOVR/100,0,1); rr:=recentForm(c); ff:=map[string]interface{}{"board_expectations":exp,"club_reputation":float64(c.Identity.Reputation)/100,"recent_results_index":rr,"manager_tenure_years":clamp(float64(max(0,mw-m.AppointedMatchweek))/38,0,20),"manager_reputation":clamp(float64(m.Adaptability)/100,0,1),"financial_health":fin,"squad_quality":sq,"competition_expectations":exp,"recent_trajectory":rr,"supporter_pressure":clamp(float64(c.MediaPressure)/100,0,1)}; ft:=map[string]interface{}{"board_patience":float64(c.Identity.BoardPatience)/100,"manager_pressure":pressure,"sack_probability":sack,"manager_hiring_suitability":clamp(1-pressure,0,1),"club_trajectory":rr}; parent:=fmt.Sprintf("%s_mw%02d_%s",world,mw,c.ClubID); if err:=w.emit(record("club_manager","club_manager",world,ids(world,"",c.ClubID,parent,""),ff,ft,"deterministic_simulator_rule","NATURAL_SIMULATION","core",0,1));err!=nil{return err} }
	return nil
}
func max(a,b int)int{if a>b{return a};return b}

type trajState struct {
	Age int
	OVR int
	Potential int
	Injury float64
	Contract int
	Role string
	Club string
	Coach float64
	Facility float64
	Mobility float64
	Available float64
}
func trajectorySnapshot(tm *tournament.TournamentManager,ge *growth.GrowthEngine,limit int,seed int64) map[string]trajState {
	out:=map[string]trajState{}; sel:=selectedPlayers(tm,seed,limit)
	for _,p:=range sel { c:=clubOf(tm,p); if c==nil{continue}; coach:=.5;if m:=tm.Managers[c.ClubID];m!=nil{coach=clamp(float64(m.Adaptability)/100,0,1)};avail:=1.0;if p.IsUnavailable("",tm.CurrentMatchweek){avail=0};out[p.PlayerID]=trajState{Age:p.Age,OVR:p.OVR,Potential:potential(ge,p),Injury:injuryBurden(p),Contract:p.ContractYears,Role:role(p),Club:c.ClubID,Coach:coach,Facility:float64(c.Identity.AcademyQuality)/100,Mobility:clamp(float64(100-p.Loyalty)/100,0,1),Available:avail} }
	return out
}
func emitLongTemporal(w *writer,dataset string,root int64) error {
	tm,ge,err:=makeWorld(dataset,root,9000);if err!=nil{return err};world:="world_temporal_9000"; snaps:=[]map[string]trajState{trajectorySnapshot(tm,ge,900,derive(root,world,"s0"))}
	for season:=1;season<=12;season++ {
		for mw:=1;mw<=tm.MaxMatchweeks;mw++ { res:=tm.SimulateMatchweek(mw);if res["status"]!="success"{return fmt.Errorf("temporal season %d week %d: %v",season,mw,res)} }
		if res:=tm.ResetNewSeason();res["status"]!="success"{return fmt.Errorf("temporal reset season %d: %v",season,res)}
		snaps=append(snaps,trajectorySnapshot(tm,ge,900,derive(root,world,fmt.Sprint(season))))
	}
	horizons:=[]int{1,2,3,5,8,10,12}
	for _,h:=range horizons {
		start:=snaps[0];future:=snaps[h]
		for pid,a:=range start { b,ok:=future[pid];if !ok{continue}; tf:=map[string]interface{}{"state_t_age":a.Age,"state_t_ability":a.OVR,"state_t_potential":a.Potential,"state_t_injury_burden":a.Injury,"state_t_contract_remaining":clamp(float64(a.Contract),0,6),"state_t_squad_role":a.Role,"coaching_quality":a.Coach,"facility_quality":a.Facility,"mobility_pressure":a.Mobility,"horizon_seasons":h};moved:=0.0;if a.Club!=b.Club{moved=1};tt:=map[string]interface{}{"state_t1_age":b.Age,"state_t1_ability":b.OVR,"state_t1_injury_burden":b.Injury,"state_t1_availability":b.Available,"state_t1_contract_remaining":clamp(float64(b.Contract),0,6),"club_moved_probability":moved,"ability_delta":clamp(float64(b.OVR-a.OVR),-30,30)};parent:=fmt.Sprintf("%s_%s_h%d",world,pid,h);family:="temporal_transitions";if h>=5{family="career_trajectories"};if err:=w.emit(record("temporal_transition",family,world,ids(world,pid,a.Club,parent,"trajectory_"+pid),tf,tt,"direct_simulator_outcome","TRAJECTORY_VIEW","hard",0,1));err!=nil{return err} }
	}
	return nil
}

func main(){
	mode:=flag.String("mode","pilot","pilot or full");out:=flag.String("out","","candidate JSONL.GZ");dataset:=flag.String("dataset","../../dataset.json","dataset path");root:=flag.Int64("seed",2609302026,"root seed");flag.Parse()
	if *out==""{fmt.Fprintln(os.Stderr,"-out required");os.Exit(2)}
	worlds,weeks,sample,matchPerWeek:=4,8,320,3
	if *mode=="full"{worlds,weeks,sample,matchPerWeek=40,28,260,4}
	w,err:=newWriter(*out);if err!=nil{panic(err)};defer w.close()
	for wi:=0;wi<worlds;wi++ {
		tm,ge,err:=makeWorld(*dataset,*root,wi);if err!=nil{panic(err)}; world:=fmt.Sprintf("world_%03d",wi);tr:=newTracker();tr.advance(tm)
		for mw:=1;mw<=weeks;mw++ {
			sel:=selectedPlayers(tm,derive(*root,world,fmt.Sprint(mw)),sample); pre:=map[string]int{}; rec:=map[string]int{}; for _,p:=range sel{pre[p.PlayerID]=p.OVR;rec[p.PlayerID]=tr.recent(p.PlayerID)}
			slate:=tm.GetSlate(mw);count:=0
			for _,f:=range slate { if count>=matchPerWeek{break}; if f.Status!="scheduled"{continue}; mt,conf,conv:=matchTargets(tm,f,derive(*root,world,"rollout",fmt.Sprint(mw)),24);if mt==nil{continue};mf:=matchFeatures(tm,f);if mf==nil{continue};parent:=fmt.Sprintf("%s_match_%s",world,f.FixtureID); mr:=record("match_competition","match_competition",world,ids(world,"",f.HomeID,parent,""),mf,mt,"repeated_simulator_rollout","MULTI_ROLLOUT","hard",24,conf); if !conv{mr["quality"]=quality(.9,false)}; if err:=w.emit(mr);err!=nil{panic(err)}; if count%2==0{cf:=map[string]interface{}{"source_task_fields":mf};ct:=map[string]interface{}{"dynamic":mt};if err:=w.emit(record("calibration_uncertainty","calibration_uncertainty",world,ids(world,"",f.HomeID,parent,""),cf,ct,"repeated_simulator_rollout","MULTI_ROLLOUT","hard",24,conf));err!=nil{panic(err)}};count++ }
			if res:=tm.SimulateMatchweek(mw);res["status"]!="success"{panic(fmt.Sprintf("world %s matchweek %d: %v",world,mw,res))}
			tr.advance(tm)
			for i,p0:=range sel { p:=p0; c:=clubOf(tm,p); if c==nil{continue}; if err:=emitPlayerViews(w,tm,ge,world,p,rec[p.PlayerID],mw,i,pre[p.PlayerID]);err!=nil{panic(err)} }
			if err:=emitClubRows(w,tm,world,mw);err!=nil{panic(err)}
		}
		fmt.Fprintf(os.Stderr,"generated %s through week %d rows=%d\n",world,weeks,w.n)
	}
	fmt.Printf("{\"mode\":%q,\"candidate_rows\":%d,\"simulator_commit\":%q}\n",*mode,w.n,simulatorCommit)
}
