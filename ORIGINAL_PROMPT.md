You are continuing development on:

Repository:
ayzeeyan/football_sim

Work from the CURRENT `main` branch.

The project has already evolved beyond a 12-club simulator. The long-term direction is now:

BUILD A DEEP, LIVING EUROPEAN FOOTBALL MANAGEMENT SIMULATOR INSPIRED BY THE DEPTH AND FEELING OF EA FC CAREER MODE / FOOTBALL MANAGEMENT GAMES, WHILE KEEPING ITS OWN ORIGINAL UI, SYSTEMS, AND IDENTITY.

This is not a request to copy EA FC’s assets, UI, branding, or proprietary presentation.

The objective is to create a polished, deterministic, persistent football universe where leagues, cups, players, managers, transfers, morale, development, injuries, finances, form, tactical decisions, and long-term careers all interact naturally.

You have broad creative freedom.

You are allowed to:
- redesign weak frontend areas
- improve backend architecture
- introduce new football systems
- replace obsolete assumptions
- create better abstractions
- improve simulation realism
- improve UX substantially
- add supporting data/state/models
- remove dead or contradictory systems
- add tests and validation

But do NOT:
- randomly rewrite working systems without first understanding them
- add shallow gimmicks that do not affect the simulation
- make frontend state authoritative over backend state
- break deterministic simulation
- weaken tests to make failing features pass
- silently discard save data

============================================================
0. START WITH A FULL PRODUCT + ARCHITECTURE AUDIT
============================================================

Before implementing features:

1. Checkout latest main.
2. Read AGENTS.md.
3. Inspect:
   - backend models
   - tournament architecture
   - competition scheduling
   - transfer engine
   - save format
   - player stats
   - manager logic
   - match engine
   - frontend navigation
   - league UI
   - squad UI
   - transfer UI
   - matchday UI
   - awards
   - inbox/news
4. Run:
   backend:
   go test ./...
   go vet ./...
   go test -race ./...

   frontend:
   run the actual test command from package.json / CI
   npm run build

Then create an internal implementation plan.

Do not assume previous features are correct merely because they exist.

============================================================
PRIMARY PRODUCT DIRECTION
============================================================

The finished product should feel like an interconnected football world rather than a set of disconnected screens.

The player should be able to:

- follow all major leagues
- inspect clubs and squads
- watch the transfer market evolve
- see player careers develop
- understand why a player is unhappy
- see why a manager picked a lineup
- follow league title races
- follow relegation battles
- watch European qualification evolve
- see cup runs
- track player form
- observe injuries and fatigue
- see managers fired
- see clubs rise and decline over years
- follow awards and records
- simulate weeks/months/seasons while the world remains coherent

Think:

"career-mode football universe"

not:

"fixture generator with some menus"

============================================================
1. WORLD STRUCTURE — TOP FIVE LEAGUES
============================================================

Support these domestic leagues as first-class competitions:

England:
Premier League

Spain:
La Liga

Germany:
Bundesliga

Italy:
Serie A

France:
Ligue 1

Use actual league-specific club counts and formats.

Premier League:
20 clubs
38 league matches

La Liga:
20 clubs
38 league matches

Serie A:
20 clubs
38 league matches

Bundesliga:
18 clubs
34 league matches

Ligue 1:
18 clubs
34 league matches

League schedules must be deterministic home-and-away double round robins.

Do not artificially repeat league cycles to inflate appearance totals.

A club should naturally reach 50–60 matches through:

league
domestic cups
European competitions

============================================================
2. DOMESTIC CUPS
============================================================

Implement proper domestic cup systems.

England:
FA Cup
EFL Cup

Spain:
Copa del Rey

Germany:
DFB-Pokal

Italy:
Coppa Italia

France:
Coupe de France

Support:

knockout draws
round progression
penalties
extra time where appropriate
cup histories
competition stats
winner tracking
prize/prestige effects
qualification consequences

Create reusable cup infrastructure rather than hardcoding every cup independently.

============================================================
3. EUROPEAN COMPETITIONS
============================================================

Build a proper cross-league European competition architecture.

Support:

UEFA Champions League
UEFA Europa League
UEFA Conference League

These should be reusable instances of a shared competition system.

Possible model:

EuropeanCompetition {
  id
  name
  participants
  league_phase
  table
  knockout_rounds
  qualification_rules
  prestige
  prize_money
}

Domestic results should determine next-season qualification.

European results should affect:

club reputation
finances
manager evaluation
player morale
transfer attraction
awards

============================================================
4. SHARED SEASON CALENDAR
============================================================

Replace any assumptions that one matchweek equals one match.

A club may play:

Saturday — league
Wednesday — Champions League
Sunday — league
Wednesday — domestic cup

Build a calendar capable of handling:

league fixtures
domestic cup fixtures
European fixtures
transfer windows
winter break where applicable
international-break placeholders if added later

Use actual dates or a coherent week/date abstraction.

The UI should eventually show a calendar resembling:

AUG
League
League
UCL
League
Cup

SEP
League
League
UCL
League

etc.

Fixture congestion should become a real management factor.

============================================================
5. TRANSFER WINDOW STATE MACHINE — FIX COMPLETELY
============================================================

There is an existing serious transfer-window lifecycle bug.

Observed failures include:

window repeatedly returning to Week 1

and sometimes:

window advancing beyond Week 12

Fix this at the backend state-machine level.

Summer transfer window should behave exactly:

Week 1
Week 2
...
Week 12
CLOSED

Never expose:

Week 13/12
Week 14/12

Never reset to Week 1 unless a NEW summer window begins.

Audit:

BeginOffSeasonWindow
AdvanceOpenWindow
CurrentWeek
CurrentDay
IsOffSeason
SeasonPhase
FinalizeSeasonTransition
ResetForNewSeason
Sim Week
Sim Month
Sim Season
save/load

Required behavior:

Sim Week:
+1 transfer week

Sim Month:
up to 4 remaining weeks

Sim Season:
process every remaining transfer week before next season begins

Save Week 7
reload
still Week 7

Do not patch this only in React.

Backend state is authoritative.

============================================================
6. WINTER TRANSFER WINDOW
============================================================

Once summer window logic is reliable, introduce a January window.

Refactor transfer lifecycle away from:

"offseason = transfer window"

toward:

TransferWindow {
  type
  open
  current_period
  start
  end
}

Potential window types:

SUMMER
WINTER
CLOSED

Different windows may have different transfer activity levels.

Summer:
major squad building

Winter:
smaller corrective market

============================================================
7. PLAYER MORALE
============================================================

Add persistent morale:

0–100

Suggested states:

90–100:
Excellent

75–89:
Happy

55–74:
Content

35–54:
Unhappy

0–34:
Very Unhappy

Morale should respond gradually to actual football events.

Positive examples:

starts
minutes
good performances
goals
assists
winning
trophies
manager trust
playing preferred position
good development

Negative examples:

repeated benching
broken playing-time expectations
poor club form
being used badly out of position
transfer request denied
losing important games
manager conflict
not making matchday squad

Keep changes bounded.

Do not make morale jump 20 points because of one match.

============================================================
8. SQUAD ROLES / PLAYING-TIME EXPECTATIONS
============================================================

Add meaningful squad roles:

Crucial
Important
Rotation
Squad
Prospect

Role determines expected playing time.

Example:

90 OVR Crucial:
expects frequent starts

84 OVR Important:
expects regular participation

77 OVR Rotation:
expects meaningful rotation

71 OVR Squad:
accepts limited appearances

young Prospect:
expects development opportunities rather than constant starts

Track playing time across a rolling recent-match window.

Potential fields:

available_matches
starts
appearances
minutes
expected_minutes
actual_minutes
satisfaction

This should feed morale.

============================================================
9. TRANSFER REQUESTS
============================================================

Very unhappy players should sometimes request a transfer.

Potential causes:

lack of playing time
club below player's level
broken squad-role expectations
manager relationship
lack of European football
club decline

Transfer request should:

persist
appear in inbox
affect selling willingness
affect buyer interest

It should NOT guarantee a transfer.

Players may withdraw requests if circumstances improve.

============================================================
10. FITNESS, SHARPNESS AND MORALE SHOULD BE SEPARATE
============================================================

Do not combine these systems.

Fitness:
physical readiness

Sharpness:
match readiness

Morale:
psychological satisfaction

Fitness decreases with:

minutes
congested fixtures
injury recovery

Fitness recovers through:

rest

Sharpness increases through:

meaningful competitive minutes

Sharpness declines through:

long periods without playing

Morale reacts to:

role
performance
results
treatment

These should create meaningful squad-management choices.

============================================================
11. PLAYER FORM
============================================================

Add recent form as a separate concept from OVR.

Use rolling recent-match ratings.

Form may consider:

rating
goals
assists
clean sheets
mistakes
minutes

Form should affect:

AI lineup selection
award weighting
transfer interest
morale
news narratives

Do not directly permanently increase OVR because of temporary good form.

============================================================
12. BETTER LINEUP AI
============================================================

Starting XI selection should become tactical and situational.

Use explicit tactical slots.

Example 4-3-3:

GK

LB
LCB
RCB
RB

LCM
CDM/CM
RCM

LW
ST
RW

Selection factors:

slot compatibility
OVR
form
fitness
sharpness
morale
injury
suspension
competition importance
manager philosophy
fixture congestion

Avoid:

two RBs taking RB/CB unnecessarily
no right winger because a stronger central player exists
duplicate overlapping tactical positions

Return:

player
+
assigned tactical slot

Frontend should render assigned slot rather than guessing from natural position.

============================================================
13. ROTATION AI
============================================================

Managers must rotate intelligently.

Example:

Premier League Saturday
Champions League Tuesday
Premier League Saturday

AI should sometimes rest players.

Factors:

fitness
recent minutes
sharpness
role
OVR
competition importance
opponent quality
injury risk

A club with 55 matches should not automatically use the exact same XI 55 times.

============================================================
14. MANAGER PERSONALITY SHOULD MATTER
============================================================

Manager styles should have actual football consequences.

Examples:

Youth Developer:
more likely to use prospects

Veteran Trust:
prefers experienced players

High Press:
values pace/stamina
higher fatigue/injury risk

Possession:
values passing/composure

Counter:
values pace/transitions

Aggressive Market:
more transfer activity

Financially Conservative:
avoids expensive transfers

Managers should influence:

XI
rotation
transfers
player development
player morale
tactical choices

============================================================
15. MANAGER JOB SECURITY
============================================================

Board expectations should become richer.

Consider:

club reputation
squad strength
finances
last season
European participation

Example expectations:

Real Madrid:
title challenge
deep Champions League run

Arsenal:
top-four/title challenge
European progress

mid-table club:
top half

relegation candidate:
survival

BoardPatience should determine tolerance.

Manager sackings should produce:

news
history
replacement
tactical changes

============================================================
16. CLUB IDENTITY SHOULD INFLUENCE BEHAVIOR
============================================================

Existing identity fields such as:

Reputation
HistoricalPrestige
FinancialPower
BoardPatience
AcademyQuality
RecruitmentAmbition
YouthPreference
TransferAggressiveness
SellingTendency

should increasingly affect simulation decisions.

Examples:

high YouthPreference:
more academy/prospect minutes

high TransferAggressiveness:
more bids

high SellingTendency:
more willing to accept offers

high FinancialPower:
larger budgets

high HistoricalPrestige:
more resistant reputation decline

============================================================
17. CLUB FINANCES
============================================================

Continue improving finances gradually.

Core fields:

Balance
TransferBudget

Possible additions:

WageBudget
WageBill
PrizeMoney
CompetitionRevenue

Do not immediately build a full accounting simulator.

But success should matter financially.

Champions League run:
major financial benefit

domestic cup:
smaller benefit

relegation:
large negative effect

============================================================
18. TRANSFER AI
============================================================

Transfers should become sporting decisions rather than random transactions.

Buying logic:

position weakness
depth
injuries
player quality
potential
age
value
finances
manager system
club ambition

Selling logic:

player importance
replacement
offer premium
club finances
player morale
transfer request
SellingTendency

Player acceptance:

club reputation
league prestige
European football
expected squad role
competition for position
current morale

Players should sometimes reject moves.

============================================================
19. TRANSFER DEADLINE DAY
============================================================

Consider creating more activity near transfer deadlines.

Example:

last two transfer weeks:

higher bid frequency
more urgent selling
more replacement signings
more transfer news

Keep it deterministic and believable.

============================================================
20. LOANS
============================================================

Loans are highly recommended once playing time matters.

Loan state should explicitly track:

parent club
temporary club
start season
return season
optional future buy clause if later added

Permanent ownership must remain separate from temporary registration.

A prospect blocked at a top club should sometimes seek a loan.

Loan development should depend on actual minutes.

============================================================
21. PLAYER DEVELOPMENT
============================================================

Preserve realistic annual growth.

Approximate target:

+1 / +2 normal
+3 strong
+4 rare
+5 exceptional maximum

Playing time may influence development.

But:

training
playing time
match XP
mentorship
form

must NEVER stack beyond the annual +5 ceiling.

Keep underlying attributes coherent with OVR.

============================================================
22. COMPETITION-SPECIFIC PLAYER STATS
============================================================

Track stats separately.

Example:

Player:
Vinícius Júnior

La Liga:
31 apps
19 goals
9 assists

Champions League:
10 apps
7 goals
4 assists

Copa del Rey:
4 apps
2 goals

TOTAL:
45 apps
28 goals
13 assists

Store:

appearances
starts
minutes
goals
assists
ratings
cards

by competition.

This should feed:

awards
UI
player profiles
records

============================================================
23. PLAYER PROFILE OVERHAUL
============================================================

Improve the player page substantially.

Potential information:

photo
club
nationality
age
position
OVR
potential
value

Morale
Fitness
Sharpness
Form

Squad role

season:
starts
apps
minutes
goals
assists
average rating

competition breakdown

career history

transfer history

trophies

development graph

injury history

Use real backend data only.

============================================================
24. SQUAD HUB OVERHAUL
============================================================

Create a genuinely useful squad management screen.

Suggested table columns:

Player
Position
OVR
Age
Role
Morale
Form
Fitness
Sharpness
Starts
Apps
Minutes
Goals
Assists
Value

Allow:

sorting
filtering
position groups

Keep the formation view.

Provide obvious visual states for:

injured
suspended
unhappy
excellent form
fatigued

============================================================
25. COMPETITION HUB
============================================================

This should become one of the strongest frontend areas.

Navigation concept:

Competitions

England
Premier League
FA Cup
EFL Cup

Spain
La Liga
Copa del Rey

Germany
Bundesliga
DFB-Pokal

Italy
Serie A
Coppa Italia

France
Ligue 1
Coupe de France

Europe
Champions League
Europa League
Conference League

Each competition page should support:

Overview
Table / Bracket
Fixtures
Results
Stats
History

============================================================
26. IMPROVE LEAGUE PAGE UX
============================================================

The current League page already has a strong visual direction.

Evolve it further.

Recommended structure:

Top league header:
logo
league name
country
current matchweek
season

Main table:
position
club crest
club
P
W
D
L
GF
GA
GD
Pts
form
qualification/status

Right sidebar:

This Week's Fixtures
Top Scorers
Top Assists
Quick Actions

Below table:

Title Race
European Race
Relegation Battle

Bottom:

Season Timeline

Use subtle qualification bands:

Champions League
Europa League
Conference League
Relegation

Do not clutter the page with giant empty panels.

============================================================
27. GLOBAL HOME DASHBOARD
============================================================

Build a football-world dashboard.

Possible cards:

Premier League leader
La Liga leader
Bundesliga leader
Serie A leader
Ligue 1 leader

Champions League status

Top scorer in Europe

Biggest transfers

Major injuries

Manager sackings

Wonderkid watch

Upcoming big fixtures

This should make the universe feel alive.

============================================================
28. MATCHDAY HUB
============================================================

Improve matchday presentation.

Pre-match:

venue
competition
table positions
form
expected XI
injuries
tactical matchup
key player

Live:

score
clock
momentum
possession
shots
events
pitch
ratings

Post-match:

score
stats
ratings
MOTM
x-style shot map if already available
standings impact
player morale changes
injuries
development

Fix any remaining:

Unknown player events
bad event identity
layout overlaps
truncated player labels

============================================================
29. MATCH EVENT QUALITY
============================================================

Ensure events preserve stable player identity.

Each event should include:

player_id
name
club
side
minute
event_type

Examples:

goal
assist
penalty
own goal
yellow
red
substitution
injury

Frontend should never normally show:

Unknown

unless loading legacy broken data.

============================================================
30. NEWS + INBOX
============================================================

Make the football world generate meaningful stories.

Examples:

transfer request
manager sacking
major signing
title race
relegation fight
cup upset
wonderkid breakout
injury
award
club record
player milestone
European qualification

Stories must be generated from actual state.

Avoid generic random filler.

============================================================
31. PLAYER INTERACTIONS
============================================================

Optional but encouraged.

Simple player requests can deepen morale.

Examples:

"I need more playing time."

"I want to go on loan."

"I want to leave."

"I'm happy with my role."

"I want a bigger role."

Keep this lightweight.

Do not build a giant dialogue simulator.

============================================================
32. RIVALRIES
============================================================

Expand derby/rivalry effects.

Derbies may influence:

morale
manager pressure
news importance
match intensity
attendance
fan expectations

Examples:

El Clásico
North London Derby
Manchester Derby
Der Klassiker
Derby della Madonnina

Keep effects moderate.

============================================================
33. COMPETITION IMPORTANCE
============================================================

Every match should have an importance score.

Examples:

League early season:
medium

Domestic cup early round:
lower

title decider:
very high

European knockout:
very high

final:
maximum

AI uses this for:

lineup strength
rotation
tactical risk
star-player usage

============================================================
34. AWARDS EXPANSION
============================================================

Preserve:

Ballon d'Or
Player of the Season
Golden Boy
Team of the Season
Manager of the Year

Add per-competition awards where useful:

Premier League Golden Boot
La Liga Pichichi-style top scorer
Bundesliga top scorer
Serie A top scorer
Ligue 1 top scorer

League Player of the Season

Young Player

European Golden Boot

Champions League Player of the Season

Award calculations should use:

performance
ratings
goals
assists
appearances
team success

not just raw goals.

============================================================
35. RECORDS + HISTORY
============================================================

Track:

league champions
cup winners
European winners
top scorers
record points
record transfers
club trophy history
player career goals
player career appearances
Ballon d'Or winners

Create a proper historical archive.

Long-term simulation should feel meaningful.

============================================================
36. SAVE SYSTEM
============================================================

This growing world needs explicit save versioning.

Persist:

leagues
cups
European competitions
calendar
transfers
morale
fitness
sharpness
roles
form
manager state
club finances
stats
history

Old saves should either:

migrate safely

or:

load as explicitly identified legacy saves

Never panic.

============================================================
37. WORLD VALIDATOR
============================================================

Expand ValidateWorldState.

Validate:

unique club IDs
unique player IDs
one permanent owning club per player
valid loans
valid league membership
fixture references
competition membership
table correctness
calendar sanity
transfer-window state
finances
morale 0–100
fitness bounds
sharpness bounds
role validity
manager references
European qualification
no duplicate fixture IDs
no player on two teams simultaneously

============================================================
38. PERFORMANCE
============================================================

The world may now contain around:

96 clubs
2000+ players
multiple competitions

Do not repeatedly scan every player for simple lookups.

Use indexes where appropriate:

clubByID
playerByID
competitionByID

Avoid unnecessary deep copies in hot paths.

Keep simulation responsive.

============================================================
39. DETERMINISTIC SIMULATION
============================================================

Determinism remains mandatory.

Same:

seed
database
user actions

must produce the same meaningful world.

Use subsystem RNG streams.

Suggested ownership:

matches
transfers
development
injuries
managers
competition draws
academy
news

Never choose RNG-indexed data from unsorted Go map iteration.

============================================================
40. FRONTEND DESIGN DIRECTION
============================================================

The visual direction should evolve toward:

premium
football-focused
data-rich
dark green
cream
brass/gold
clean typography
professional spacing

Use:

club crests
competition logos
subtle photography where appropriate
small contextual icons

Do not copy EA FC's exact design.

Create an original football-management identity.

============================================================
41. FRONTEND INFORMATION ARCHITECTURE
============================================================

Strong candidate navigation:

Home
Match
Leagues
Competitions
Squads
Transfers
Players
Inbox
History

Potential global header:

current season
current date
next fixture
Continue button
notifications
simulation controls

Avoid forcing users to understand internal simulation concepts like:

"MW scheduler state"

Use football language.

============================================================
42. CONTINUE BUTTON
============================================================

Consider replacing several simulation actions with a primary:

Continue

button.

Example:

Continue
"Simulate to next important event"

Possible stopping points:

next watched-club match
transfer offer
player request
injury
cup draw
transfer deadline
season event

Keep Week / Month / Season simulation available as secondary controls.

This will make the product feel much more like a career simulator.

============================================================
43. SMALL UX DETAILS
============================================================

Improve:

hover states
empty states
loading states
tooltips
keyboard navigation
responsive layout
text truncation
scroll behavior
modal sizing
table readability

Avoid huge empty cards.

Avoid information being hidden below fixed-height containers.

============================================================
44. RESPONSIVENESS
============================================================

Primary target:

desktop

But test at least:

1366×768
1920×1080

Key screens should not break:

League
Competition Hub
Squads
Transfers
Match
Awards
Player Profile

============================================================
45. CREATIVE AUTHORITY
============================================================

You have explicit permission to introduce additional features if they improve the central goal:

"A living European football career universe."

Potential creative additions include:

captaincy
player leadership
homegrown status
club registration rules
academy intake
loan reports
deadline-day feed
club power rankings
dynamic rivalries
dynamic player roles
player promises
clean-sheet tracking
manager tactical evolution
UEFA coefficients
stadium attendance
club fan expectations
club season previews
media pressure
team chemistry
player versatility
position retraining

Do not implement everything simply because it is listed.

Choose systems that interact meaningfully with the rest of the simulation.

============================================================
46. IMPLEMENT IN COHERENT PHASES
============================================================

Recommended sequence:

PHASE A — World
Top 5 leagues
domestic cups
shared calendar
competition hub

PHASE B — Europe
Champions League
Europa League
Conference League
qualification

PHASE C — Transfers
summer state-machine fix
winter window
transfer AI
destination logic
loans

PHASE D — Squad Management
morale
roles
fitness
sharpness
form
rotation

PHASE E — Frontend
Home Dashboard
League UI
Competition Hub
Squad Hub
Player Profile
Transfer Centre

PHASE F — Career Depth
manager objectives
player requests
records
news
history
rivalries

Each phase should be:

implemented
tested
validated
committed

before spreading into too many unfinished systems.

============================================================
47. MULTI-SEASON SOAK
============================================================

Maintain a long-run automated simulation.

Eventually target 10 complete seasons.

Verify every year:

all five leagues complete
cups complete
European competitions complete
qualification works
transfer windows close correctly
no Week 13 summer transfer state
finances remain valid
ownership remains valid
loans return correctly
morale remains bounded
fitness remains bounded
development <= +5 annually
managers remain valid
save/reload works
history persists
next season starts correctly

Never weaken assertions to hide world-state corruption.

============================================================
48. REQUIRED FINAL ENGINEERING QUALITY
============================================================

Backend:

go test ./...
go vet ./...
go test -race ./...

Frontend:

run actual frontend test suite
npm run build

Also manually smoke-test:

Home
League page
Competition Hub
Squad Hub
Player Profile
Match
Transfer Centre
Awards
Inbox
season rollover
summer transfer Week 12
winter transfer window

============================================================
49. FINAL REPORT
============================================================

When a development phase is complete, report:

features added
backend architecture changes
frontend improvements
bugs found
bugs fixed
competition systems
transfer changes
player systems
manager systems
save migration
determinism
world validation
performance considerations
multi-season simulation result
Go tests
vet
race detector
frontend tests
frontend build
remaining limitations

============================================================
FINAL PRODUCT PRINCIPLE
============================================================

Every feature should answer at least one of these questions:

Does it make clubs feel more alive?

Does it make players feel more like careers rather than numbers?

Does it create meaningful management decisions?

Does it make competitions feel connected?

Does it make long-term simulation more interesting?

Does it make the frontend easier and more enjoyable to use?

If the answer is no, do not add it merely because it sounds impressive.

The target is not maximum feature count.

The target is:

A coherent, believable, attractive, persistent European football world that becomes more interesting the longer it is simulated.