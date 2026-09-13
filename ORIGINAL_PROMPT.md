You are continuing development on:

Repository:
ayzeeyan/football_sim

Work from the CURRENT `main` branch.

The project already has a functioning football simulation foundation with clubs, players, live matches, transfers, wonderkids, awards, managers, club identity, finances, persistence, cups, and long-term simulation.

Your new mission is to begin a major WORLD EXPANSION phase.

The primary objective is:

TURN THE PROJECT FROM A 12-CLUB SUPER-LEAGUE SIMULATOR INTO A LIVING EUROPEAN FOOTBALL CAREER-MODE WORLD INSPIRED BY THE DEPTH OF EA FC CAREER MODE.

Do NOT copy EA FC assets, branding, UI, proprietary data presentation, or copyrighted designs.

The goal is to achieve a similar feeling of:

- multiple real leagues
- domestic competitions
- European qualification
- transfers
- squad management
- morale
- playing-time expectations
- player development
- managerial decisions
- dynamic club strength
- season-to-season consequences
- rich frontend presentation

You have significant creative freedom.

You are explicitly encouraged to:
- improve systems that feel shallow
- redesign weak frontend areas
- add coherent football-simulation features
- add supporting models/state where needed
- improve simulation realism
- add tests and validation
- simplify bad architecture
- remove obsolete Super League assumptions

However:

DO NOT recklessly rewrite healthy systems.

Inspect first, preserve working foundations, and evolve the project coherently.

============================================================
PHASE 0 — INSPECT THE CURRENT MAIN BRANCH FIRST
============================================================

Before changing anything:

git checkout main
git pull
git status

Read:

AGENTS.md
README
current architecture
database / dataset structure
all competition code
season scheduler
transfer engine
club/player models
save format
frontend navigation
current CI workflow

Run:

cd backend_go
go test ./...
go vet ./...
go test -race ./...

cd ../frontend

Run the actual frontend test command from package.json / CI.

Then:

npm run build

Do not assume previous code is correct because it merged.

Create a short internal implementation plan after inspection.

============================================================
PRIMARY GOAL — BUILD THE TOP FIVE EUROPEAN LEAGUES
============================================================

This is the highest-priority feature.

Replace the concept of one isolated 12-club Super League with a real multi-league football ecosystem.

Target leagues:

ENGLAND
Premier League

SPAIN
La Liga

GERMANY
Bundesliga

ITALY
Serie A

FRANCE
Ligue 1

FIRST inspect the existing database.

Use the clubs and players already present in the database wherever possible.

Do NOT duplicate teams that already exist.

Do NOT create alternate IDs for the same club.

ClubID must remain stable and globally unique.

If the database already contains complete squads for these leagues, use them.

If some clubs are missing, build the architecture so missing content can be added cleanly rather than hardcoding assumptions around only the currently available teams.

============================================================
1. REAL DOMESTIC LEAGUE FORMATS
============================================================

Each league must have its own competition object/state.

Use realistic scheduling.

Premier League:
20 clubs
38 league matches per club

La Liga:
20 clubs
38 matches

Serie A:
20 clubs
38 matches

Bundesliga:
18 clubs
34 matches

Ligue 1:
18 clubs
34 matches

Each league uses:

home-and-away double round robin

Each pair of clubs meets:

once at home
once away

No fake repeated league cycles to increase appearance totals.

Generic formula:

for N clubs:

matches per club = 2 × (N - 1)

total fixtures = N × (N - 1)

The scheduler must support:

even-sized leagues
odd-sized leagues
deterministic fixture generation
home/away balance
fixture validation
rescheduling when necessary
multiple competitions sharing a calendar

Do not make unrelated competition generation consume the match RNG stream.

============================================================
2. DOMESTIC CUPS
============================================================

Each country needs its major domestic cup.

England:
FA Cup

Spain:
Copa del Rey

Germany:
DFB-Pokal

Italy:
Coppa Italia

France:
Coupe de France

England should ALSO support the EFL Cup if the database/simulation scale makes it practical.

Domestic cups must:

have actual knockout brackets
track rounds
track winners
appear on the calendar
count toward appearances
affect trophies
affect reputation
affect manager evaluation
affect morale
affect European qualification where appropriate

Do not fake cup winners from league position.

Matches must actually be simulated.

Support:

single-leg knockout
extra time where appropriate
penalty shootouts
home/away assignment
round progression

Make bracket generation deterministic from the universe seed.

============================================================
3. UEFA CHAMPIONS LEAGUE
============================================================

Build a real cross-league UEFA Champions League system.

Do not constrain it to the previous 12 clubs.

The competition must draw qualified clubs from the domestic leagues.

Target a modern Champions League structure.

Preferred structure:

League Phase
→ knockout qualification
→ knockout rounds
→ final

If implementing the full modern 36-team format is reasonable with the available database, do it.

Otherwise implement the architecture in a way that can grow cleanly into that format.

Important:

Do NOT silently substitute the old 12-club Champions Cup and call it UCL.

European competition should be a real multi-league competition.

Track:

qualification source
league-phase table
fixtures
points
goal difference
qualification status
knockout bracket
champion
top scorers
club coefficient / prestige impact

============================================================
4. EUROPA LEAGUE AND CONFERENCE LEAGUE
============================================================

After the UCL architecture is stable, add:

UEFA Europa League

UEFA Conference League

These do not need to be built through three completely separate engines.

Create reusable European competition infrastructure.

Example concept:

EuropeanCompetition

with configurable:

competition ID
name
participants
qualification rules
league-phase format
knockout rules
prestige value
financial rewards

Avoid copy-pasting three tournament engines.

============================================================
5. EUROPEAN QUALIFICATION
============================================================

Domestic results must affect next season.

Qualification should consider:

league placement
domestic cup winner
European title winners where supported

The exact slot distribution can be represented through configuration rather than hardcoded across unrelated files.

Create explicit qualification rules.

Examples:

Champions League places
Europa League places
Conference League places

Do not let the frontend decide qualification.

Backend season transition owns it.

============================================================
6. SEASON CALENDAR ENGINE
============================================================

The old simulation was designed around a single league calendar.

That assumption must be removed.

Build a calendar capable of holding:

domestic league
domestic cup
league cup
Champions League
Europa League
Conference League
international breaks if later added
transfer windows

A club may play:

League Saturday
Champions League Wednesday
League Sunday

or similar schedules.

The calendar must support multiple fixtures in the same simulation week.

Do NOT make "one week = exactly one match."

This is extremely important.

Players on clubs with deep cup/European runs should naturally reach:

45
50
55
60+

competitive appearances.

A club eliminated early should play fewer.

That is how realistic match volume should emerge.

Do NOT manufacture extra league matches to reach 60.

============================================================
7. FIX THE TRANSFER WINDOW STATE MACHINE
============================================================

There is a known serious bug.

The transfer window can sometimes:

stay at Week 1

OR

simulate beyond Week 12

Both behaviors are incorrect.

Treat the transfer window as a strict finite state machine.

Summer window:

Week 1
Week 2
...
Week 12

No Week 13.
No Week 14.
No repeated Week 1 reset.

After Week 12 has been PROCESSED:

close the summer window

then allow the next lifecycle stage.

Trace:

BeginOffSeasonWindow()
AdvanceOpenWindow()
CurrentWeek
CurrentDay
IsOffSeason
SeasonPhase
FinalizeSeasonTransition()
ResetForNewSeason()
macro simulation
save/load

Likely bug classes include:

BeginOffSeasonWindow being called repeatedly

CurrentWeek being reset to 1

CurrentWeek increment occurring after an invalid transition

UI using CurrentDay while backend uses CurrentWeek

season rollover checking the wrong boundary

Required invariant:

1 <= CurrentWeek <= 12

while the window is active.

Once Week 12 is completed:

IsWindowOpen = false

The engine must NEVER expose:

Week 13/12

or:

Week 14/12

or similar.

Sim Week:

advances exactly one transfer week

Sim Month:

advances at most four remaining transfer weeks

Sim Season:

processes ALL remaining transfer weeks and then transitions

Save at Week 7
→ load
→ still Week 7

Add regression tests for all boundary states.

============================================================
8. ADD A WINTER / JANUARY TRANSFER WINDOW
============================================================

Once the summer window state machine is correct, introduce a winter window.

The world should feel closer to real football.

Typical concept:

Summer:
large primary transfer window

Winter:
shorter January window

The exact internal number of simulation weeks can be configurable.

Do not hardcode transfer behavior around only "offseason."

Introduce explicit window types such as:

SUMMER
WINTER
CLOSED

Example:

TransferWindowState

type
start week/date
end week/date
current week
open
processed

This will make the system easier to expand later.

============================================================
9. PLAYER MORALE SYSTEM
============================================================

Introduce persistent player morale.

This should be meaningful but not annoyingly random.

Suggested range:

0–100

Example bands:

90–100
Excellent

75–89
Happy

55–74
Content

35–54
Unhappy

0–34
Very Unhappy

Morale should react to believable football events.

Positive examples:

regular starts
good performances
goals
assists
winning matches
winning trophies
promotion to important squad role
manager praise
successful development
playing in preferred position

Negative examples:

lack of playing time
repeated benching
being left out of matchday squad
poor club form
broken playing-time expectations
being played badly out of position
rejected transfer request
losing important matches
manager conflict

Do not apply huge morale swings every game.

Use gradual bounded changes.

============================================================
10. SQUAD ROLE / PLAYING-TIME EXPECTATIONS
============================================================

Give players a squad role.

Suggested roles:

Crucial
Important
Rotation
Squad
Prospect

This should matter.

A 90 OVR star marked Crucial should expect substantial playing time.

If he repeatedly gets 5-minute substitute appearances or remains benched:

morale should fall.

A 67 OVR Prospect should NOT demand the same number of starts.

Create expected playing-time logic based on:

role
OVR relative to squad
age
potential
recent form
competition importance

Track playing time over a rolling period rather than reacting to one match.

Example:

last 8–12 competitive matches.

Possible metrics:

starts
appearances
minutes
available matches

Then calculate:

expected_minutes_ratio
actual_minutes_ratio
playing_time_satisfaction

============================================================
11. PLAYING-TIME CONSEQUENCES
============================================================

Morale must have consequences without becoming arcade-like.

Possible effects:

small performance modifier
training effectiveness
development speed
transfer-request probability
contract willingness later
manager relationship

Keep gameplay effects bounded.

Example:

very happy:
small confidence benefit

content:
neutral

unhappy:
minor composure/form penalty

very unhappy:
greater chance to request move

Never give something absurd such as:

-20 OVR because morale is low.

============================================================
12. PLAYER TRANSFER REQUESTS
============================================================

Players can become unhappy enough to request a transfer.

Potential triggers:

very low morale
lack of expected playing time
club level below ability
repeated manager conflict
desire for stronger competition

Transfer request must:

persist
appear in inbox/news
increase seller willingness
NOT guarantee a transfer

The player may later withdraw the request if conditions improve.

============================================================
13. MATCH SHARPNESS / FITNESS / FATIGUE
============================================================

Consider adding a lightweight Career-Mode-style match-readiness model.

Possible separate values:

Fitness
Sharpness
Morale

Fitness:

short-term physical readiness

decreases with:
minutes played
fixture congestion
injury recovery

recovers with:
rest

Sharpness:

match readiness

increases with:
regular competitive minutes

decreases with:
long periods without playing

Morale:

psychological satisfaction

Do NOT combine all three into one stat.

This will make squad rotation meaningful.

============================================================
14. ROTATION AI
============================================================

Managers should understand fixture congestion.

If a club has:

league
Champions League
cup

within a short span,

AI should sometimes rotate.

Rotation choices should consider:

fitness
sharpness
morale
player role
OVR
competition importance
injury risk

A manager should not play the exact same XI 60 times unless squad depth truly forces it.

============================================================
15. PLAYER FORM
============================================================

Add recent form separate from OVR.

Form should derive from recent:

match ratings
goals
assists
clean sheets for relevant positions
mistakes
minutes

Use a rolling window.

Form should influence:

XI selection
award consideration
transfer interest
morale
media/inbox narratives

Do NOT permanently inflate OVR just because a player scored twice.

============================================================
16. COMPETITION IMPORTANCE
============================================================

Managers should understand that not all matches have equal importance.

Possible scale:

friendly
early domestic cup
league
European league phase
domestic cup semifinal
title-deciding league match
European knockout
European final

Use this when deciding:

lineup strength
rotation
risk tolerance
starters
substitutions

============================================================
17. TRANSFER AI EXPANSION
============================================================

With many leagues, transfer AI needs to become more intelligent.

Potential buying factors:

position need
player OVR
player potential
age
market value
club finances
club reputation
league prestige
player morale
player playing time
club ambition
manager tactical fit

Selling factors:

financial need
selling tendency
player importance
replacement availability
contract situation later
player transfer request
bid premium

Avoid random transfers with no sporting logic.

============================================================
18. TRANSFER DESTINATION LOGIC
============================================================

Players should evaluate moves.

Consider:

buyer reputation
expected playing time
league strength
European qualification
wage potential later
current morale
club ambition
competition for their position

A player should sometimes reject a move.

Example:

85 OVR starting striker at Milan

may reject:

mid-table Ligue 1 club offering rotation role

but consider:

Manchester United offering Important role

Do not make decisions entirely based on transfer fee.

============================================================
19. CLUB REPUTATION NOW MATTERS MUCH MORE
============================================================

The existing reputation system should be expanded into the multi-league world.

Reputation should affect:

transfer attraction
manager pressure
player expectations
European prestige
commercial/financial capacity
club expectations

HistoricalPrestige should remain relatively stable.

Reputation should be dynamic.

A historically large club can fall.

A smaller club can rise after sustained success.

But changes should take years, not one cup upset.

============================================================
20. CLUB SEASON EXPECTATIONS
============================================================

Create realistic board expectations.

Examples:

Real Madrid:
challenge for league
deep Champions League run

Manchester City:
title challenge
Champions League expectation

mid-table club:
top-half finish

relegation candidate:
survival

Board expectations should consider:

reputation
financial power
squad rating
previous season
European qualification

Manager stability should use these expectations plus BoardPatience.

============================================================
21. CLUB FINANCES
============================================================

Preserve existing club finance work and expand carefully.

Clubs should have:

Balance
TransferBudget

Possible future additions:

wage budget
competition prize money
European revenue
league finishing rewards
domestic cup rewards

Avoid making financial simulation excessively complicated immediately.

But competition success should have visible financial consequences.

============================================================
22. PLAYER DEVELOPMENT
============================================================

Preserve the existing realistic development guard.

Annual OVR growth target remains approximately:

+1 / +2 typical
+3 strong
+4 rare
+5 exceptional hard maximum

Playing time should influence development.

Young players who:

train well
play regularly
perform well

should develop more reliably.

Highly talented players who never play should still develop somewhat through training, but generally slower.

Do NOT let playing time stack with training to bypass the annual +5 ceiling.

============================================================
23. LOANS — HIGHLY RECOMMENDED
============================================================

Since playing time now matters, implement loans if the architecture can support them cleanly.

A young player who cannot get minutes at Arsenal might be loaned to another club.

Loan state should include:

parent club
temporary club
season/end date
optional buy clause later

At loan end:

player returns to parent club

This is DIFFERENT from permanent transfers.

Do not confuse:

OriginalClubID

with loan ownership.

Permanent ownership and temporary registration need separate fields.

============================================================
24. LEAGUE TABLES + COMPETITION HUB
============================================================

Redesign navigation as needed.

The frontend should make the expanded world understandable.

Consider a Competition Hub.

Possible navigation:

Home
Matches
Competitions
Squads
Transfers
Players
Inbox
History

Competitions page:

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

Each competition should expose:

table or bracket
fixtures
results
top scorers
current stage
holder
history

============================================================
25. WORLD DASHBOARD
============================================================

Consider adding a world overview.

Examples:

League leaders

Premier League
Arsenal

La Liga
Real Madrid

Bundesliga
Bayern

Serie A
Inter

Ligue 1
PSG

European qualification race

top scorers across Europe

big transfers

manager sackings

injuries

wonderkid watch

This would make the simulation feel alive beyond the currently selected club.

============================================================
26. PLAYER PROFILE IMPROVEMENT
============================================================

Expand player profile to show meaningful career information.

Potential fields:

age
nationality
position
OVR
potential
club
squad role
morale
form
fitness
sharpness
market value
season appearances
starts
minutes
goals
assists
competition breakdown
career history
transfer history
trophies
development timeline

Use real backend values.

No fake frontend-only state.

============================================================
27. SQUAD HUB IMPROVEMENT
============================================================

The squad screen should become closer to a real management overview.

Useful columns:

Position
Player
OVR
Age
Role
Morale
Form
Fitness
Sharpness
Apps
Starts
Minutes
Goals
Assists
Value
Availability

Allow sorting/filtering.

Keep the formation view.

Fix any existing formation overlap and slot-selection issues.

============================================================
28. PLAYER CONVERSATION / REQUEST SYSTEM — OPTIONAL CREATIVE FEATURE
============================================================

You have freedom to add lightweight player interactions if they integrate naturally.

Examples:

"I need more playing time."

"I've been in good form."

"I want to leave."

"I'm happy with my role."

Do not turn this into a giant dialogue game.

A simple event/inbox system is enough.

Choices may slightly affect morale.

============================================================
29. MANAGER AI PERSONALITY
============================================================

Existing manager styles should affect more than labels.

Examples:

Youth-focused manager:
more willing to start prospects

Conservative manager:
trusts veterans

High press manager:
prefers stamina/pace

Possession manager:
prefers technical midfielders

Aggressive transfer manager:
more market activity

Manager philosophy should influence:

XI
rotation
transfers
development opportunities

Keep behavior deterministic for the same universe seed.

============================================================
30. INJURY AND FIXTURE-CONGESTION REALISM
============================================================

Fixture congestion should slightly increase injury risk.

Risk factors:

low fitness
many recent minutes
high pressing
congested schedule

Do NOT create constant injuries.

The goal is squad-management pressure, not punishment.

============================================================
31. NEWS / INBOX EXPANSION
============================================================

A larger football world should generate stories such as:

league title race
cup upsets
European qualification
player transfer request
major transfer
manager sacking
wonderkid breakout
injury
award
record broken

Keep generated stories grounded in actual simulation state.

============================================================
32. AWARDS EXPANSION
============================================================

Current awards can remain.

Consider adding:

League Player of the Season
League Young Player of the Season
Golden Boot per league
European Golden Boot
Team of the Season per league
Champions League Player of the Season

Ballon d'Or remains global.

Avoid giving every award to the same player just because they scored the most goals.

============================================================
33. STATISTICS BY COMPETITION
============================================================

Do not only store one aggregate goal count.

Ideally track season statistics per competition.

Example:

Player X

Premier League:
30 apps
18 goals

FA Cup:
5 apps
4 goals

Champions League:
10 apps
6 goals

TOTAL:
45 apps
28 goals

This becomes especially important once there are many competitions.

Design this cleanly.

============================================================
34. SAVE FORMAT MIGRATION
============================================================

This expansion significantly changes world structure.

Do not silently destroy old saves.

Introduce an explicit save version if one does not already exist.

Migrate where reasonable.

If an old 12-club save cannot safely become a full European world, fail gracefully or start a clearly identified legacy universe.

Never panic on load.

============================================================
35. WORLD VALIDATION
============================================================

Expand ValidateWorldState.

Check:

unique clubs
correct league membership
no duplicate player ownership
fixture club references
competition IDs
league table integrity
transfer window bounds
club finances
morale bounds
fitness bounds
sharpness bounds
player role validity
European qualification consistency
no duplicate fixtures
no player scheduled for two clubs
loan ownership if implemented

============================================================
36. PERFORMANCE
============================================================

The world will now contain many more clubs and players.

Do not accidentally make simulation O(N^3) everywhere.

Profile obvious hot paths.

Avoid repeatedly scanning every player in Europe for simple lookups.

Introduce indexed structures where appropriate:

playerByID
clubByID
competitionByID

Do not sacrifice correctness for micro-optimization.

============================================================
37. DETERMINISM
============================================================

This remains extremely important.

Same:

universe seed
database
actions

should produce the same meaningful world.

Use subsystem RNG ownership.

Examples:

match RNG
transfer RNG
development RNG
manager RNG
competition draw RNG
injury RNG

Do not let Go map iteration affect:

draws
manager decisions
transfer targets
awards
lineups

Sort before RNG indexing.

============================================================
38. FRONTEND — FULL CREATIVE FREEDOM
============================================================

You are allowed to substantially improve the frontend where it improves the football-management experience.

You may:

reorganize navigation
improve information hierarchy
add competition dashboards
redesign squad tables
improve match presentation
improve responsive behavior
add richer charts
improve the transfer center
improve player profiles
improve inbox presentation
improve league tables
improve awards screens

Preserve the existing dark football aesthetic unless a better coherent evolution is appropriate.

Do NOT copy EA FC's exact UI.

Aim for:

professional
clean
dense but readable
football-focused
responsive
fast

Desktop remains primary.

============================================================
39. CREATIVE AUTHORITY
============================================================

You have permission to add additional systems that are not explicitly listed here IF they clearly contribute to the central goal:

"Make this feel like a living modern European football career simulation."

Before adding a feature, ask internally:

Does this improve:

football realism?
player/club decision-making?
world persistence?
career depth?
presentation?
long-term simulation?

If yes, and it integrates cleanly, you may implement it.

Examples of acceptable creative additions:

captaincy
club rivalries
homegrown status
squad registration
player promises
clean-sheet stats
manager tactical evolution
cup draws
derby morale impact
competition prize money
loan development
transfer deadline-day activity
season previews
club power rankings
European coefficients
dynamic club objectives

Avoid random gimmicks.

Every system should interact with the simulation meaningfully.

============================================================
40. IMPLEMENT IN PHASES
============================================================

Do not attempt 40 half-finished systems simultaneously.

Recommended development order:

PHASE A
World architecture
Top 5 domestic leagues
scheduler
calendar
domestic cups

PHASE B
Champions League
European qualification
Europa / Conference architecture

PHASE C
transfer-window correctness
winter transfer window
multi-league transfer AI

PHASE D
morale
roles
playing time
fitness
sharpness
form
rotation AI

PHASE E
frontend world/competition/squad overhaul

PHASE F
loans and deeper career systems

At the end of EACH phase:

run tests
run validator
run deterministic simulation
commit coherent changes

============================================================
41. LONG-RUN SIMULATION TEST
============================================================

The existing 10-season soak must evolve into a European-world soak.

At minimum test several seasons.

Eventually target:

10 complete seasons

Validate every year:

all domestic leagues finish
domestic cups finish
European competitions finish
qualification works
transfer windows close correctly
NO transfer week > maximum
finances stay valid
no duplicate players
morale stays 0–100
growth <= +5 annually
ownership remains valid
saves reload
next season initializes
competition history persists

Do not disable assertions simply because the larger world exposes bugs.

============================================================
42. REQUIRED TRANSFER WINDOW REGRESSION
============================================================

Explicitly assert:

CurrentWeek can never exceed TransferWindowWeeks while open.

Test:

Week 1
Week 2
...
Week 11
Week 12
CLOSED

Never:

Week 13
Week 14

Also test repeated calls to:

Sim Week
Sim Month
Sim Season

The state machine must remain valid.

============================================================
43. TESTING
============================================================

Final backend requirements:

go test ./...
go vet ./...
go test -race ./...

Frontend:

run the actual frontend test suite
npm run build

Add targeted tests for:

league schedules
cup draws
European qualification
transfer boundaries
morale
playing-time satisfaction
rotation
player transfer requests
long-run growth
save migration
multi-league persistence

============================================================
44. MANUAL SMOKE TEST
============================================================

Before declaring the phase stable, manually inspect:

Premier League table
La Liga table
Bundesliga table
Serie A table
Ligue 1 table

Domestic cup brackets

Champions League

world calendar

club squad

player profile

player morale after being benched

player morale after playing regularly

transfer center

summer transfer Week 12 boundary

winter transfer window

European qualification

season rollover

============================================================
45. FINAL REPORT
============================================================

When finished, provide a detailed engineering report containing:

architecture changes
database findings
leagues implemented
clubs loaded
domestic cups implemented
European competitions implemented
calendar model
qualification model
transfer-window bug root cause
transfer-window fix
winter-window behavior
morale system
playing-time model
fitness/sharpness/form implementation
rotation AI
transfer AI changes
loans if implemented
frontend improvements
save migration
determinism status
performance considerations
world validator status
multi-season soak results
Go tests
Go vet
race detector
frontend tests
frontend build
known limitations

Do not claim a system exists if it is merely stubbed.

============================================================
OVERALL PRODUCT DIRECTION
============================================================

The long-term target is:

A self-contained European football universe where the user can simulate years of football and watch clubs, managers, players, wonderkids, transfers, competitions, morale, form, development, and trophies evolve naturally.

Think less:

"12 clubs playing repeated games."

Think more:

"An interconnected football world."

The main priority for this phase is:

TOP FIVE LEAGUES + DOMESTIC CUPS + EUROPEAN COMPETITIONS + A REAL SHARED CALENDAR.

Once that foundation is correct, deepen:

players
morale
roles
rotation
transfers
frontend
career systems.

You have creative authority to improve the project beyond the exact wording of this prompt, provided every addition strengthens that central vision and is implemented coherently, deterministically, persistently, and with tests.