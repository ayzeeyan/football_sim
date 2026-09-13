import type { Club } from '../types';

// Club crest assets are loaded directly rather than fabricated from initials.
// Paths were verified against the football-logos source tree for the complete
// 2026–27 dataset; encode each segment so accented names and spaces are safe.
const G = 'https://raw.githubusercontent.com/luukhopman/football-logos/master/logos';
const crestUrl = (path: string) => `${G}/${path.split('/').map(encodeURIComponent).join('/')}`;

const CREST_PATHS: Record<string, string> = {
  // England — Premier League
  'EPL-ARS': 'England - Premier League/Arsenal FC.png',
  'EPL-AVL': 'England - Premier League/Aston Villa.png',
  'EPL-BOU': 'England - Premier League/AFC Bournemouth.png',
  'EPL-BRE': 'England - Premier League/Brentford FC.png',
  'EPL-BHA': 'England - Premier League/Brighton & Hove Albion.png',
  'EPL-CHE': 'England - Premier League/Chelsea FC.png',
  'EPL-COV': 'England - Premier League/Coventry City.png',
  'EPL-CRY': 'England - Premier League/Crystal Palace.png',
  'EPL-EVE': 'England - Premier League/Everton FC.png',
  'EPL-FUL': 'England - Premier League/Fulham FC.png',
  'EPL-HUL': 'England - Premier League/Hull City.png',
  'EPL-IPS': 'England - Premier League/Ipswich Town.png',
  'EPL-LEE': 'England - Premier League/Leeds United.png',
  'EPL-LIV': 'England - Premier League/Liverpool FC.png',
  'EPL-MCI': 'England - Premier League/Manchester City.png',
  'EPL-MUN': 'England - Premier League/Manchester United.png',
  'EPL-NEW': 'England - Premier League/Newcastle United.png',
  'EPL-NFO': 'England - Premier League/Nottingham Forest.png',
  'EPL-SUN': 'England - Premier League/Sunderland AFC.png',
  'EPL-TOT': 'England - Premier League/Tottenham Hotspur.png',

  // Spain — La Liga
  'LAL-ALA': 'Spain - LaLiga/Deportivo Alavés.png',
  'LAL-ATH': 'Spain - LaLiga/Athletic Bilbao.png',
  'LAL-ATM': 'Spain - LaLiga/Atlético de Madrid.png',
  'LAL-BAR': 'Spain - LaLiga/FC Barcelona.png',
  'LAL-CEL': 'Spain - LaLiga/Celta de Vigo.png',
  'LAL-DEP': 'Spain - LaLiga/Deportivo A Coruña.png',
  'LAL-ELC': 'Spain - LaLiga/Elche CF.png',
  'LAL-ESP': 'Spain - LaLiga/RCD Espanyol Barcelona.png',
  'LAL-GET': 'Spain - LaLiga/Getafe CF.png',
  'LAL-LEV': 'Spain - LaLiga/Levante UD.png',
  'LAL-MAL': 'Spain - LaLiga/Málaga CF.png',
  'LAL-OSA': 'Spain - LaLiga/CA Osasuna.png',
  'LAL-RAC': 'Spain - LaLiga/Racing Santander.png',
  'LAL-RAY': 'Spain - LaLiga/Rayo Vallecano.png',
  'LAL-RBB': 'Spain - LaLiga/Real Betis Balompié.png',
  'LAL-RMA': 'Spain - LaLiga/Real Madrid.png',
  'LAL-RSO': 'Spain - LaLiga/Real Sociedad.png',
  'LAL-SEV': 'Spain - LaLiga/Sevilla FC.png',
  'LAL-VAL': 'Spain - LaLiga/Valencia CF.png',
  'LAL-VIL': 'Spain - LaLiga/Villarreal CF.png',

  // Italy — Serie A
  'SEA-ATA': 'Italy - Serie A/Atalanta BC.png',
  'SEA-BOL': 'Italy - Serie A/Bologna FC 1909.png',
  'SEA-CAG': 'Italy - Serie A/Cagliari Calcio.png',
  'SEA-COM': 'Italy - Serie A/Como 1907.png',
  'SEA-FIO': 'Italy - Serie A/ACF Fiorentina.png',
  'SEA-FRO': 'Italy - Serie A/Frosinone Calcio.png',
  'SEA-GEN': 'Italy - Serie A/Genoa CFC.png',
  'SEA-INT': 'Italy - Serie A/Inter Milan.png',
  'SEA-JUV': 'Italy - Serie A/Juventus FC.png',
  'SEA-LAZ': 'Italy - Serie A/SS Lazio.png',
  'SEA-LEC': 'Italy - Serie A/US Lecce.png',
  'SEA-MIL': 'Italy - Serie A/AC Milan.png',
  'SEA-MON': 'Italy - Serie A/AC Monza.png',
  'SEA-NAP': 'Italy - Serie A/SSC Napoli.png',
  'SEA-PAR': 'Italy - Serie A/Parma Calcio 1913.png',
  'SEA-ROM': 'Italy - Serie A/AS Roma.png',
  'SEA-SAS': 'Italy - Serie A/US Sassuolo.png',
  'SEA-TOR': 'Italy - Serie A/Torino FC.png',
  'SEA-UDI': 'Italy - Serie A/Udinese Calcio.png',
  'SEA-VEN': 'Italy - Serie A/Venezia FC.png',

  // Germany — Bundesliga
  'BUN-AUG': 'Germany - Bundesliga/FC Augsburg.png',
  'BUN-BAY': 'Germany - Bundesliga/Bayern Munich.png',
  'BUN-BMG': 'Germany - Bundesliga/Borussia Mönchengladbach.png',
  'BUN-KOE': 'Germany - Bundesliga/1.FC Köln.png',
  'BUN-DOR': 'Germany - Bundesliga/Borussia Dortmund.png',
  'BUN-ELV': 'Germany - Bundesliga/SV 07 Elversberg.png',
  'BUN-SGE': 'Germany - Bundesliga/Eintracht Frankfurt.png',
  'BUN-SCF': 'Germany - Bundesliga/SC Freiburg.png',
  'BUN-HSV': 'Germany - Bundesliga/Hamburger SV.png',
  'BUN-TSG': 'Germany - Bundesliga/TSG 1899 Hoffenheim.png',
  'BUN-RBL': 'Germany - Bundesliga/RB Leipzig.png',
  'BUN-B04': 'Germany - Bundesliga/Bayer 04 Leverkusen.png',
  'BUN-M05': 'Germany - Bundesliga/1.FSV Mainz 05.png',
  'BUN-SCP': 'Germany - Bundesliga/SC Paderborn 07.png',
  'BUN-S04': 'Germany - Bundesliga/FC Schalke 04.png',
  'BUN-VFB': 'Germany - Bundesliga/VfB Stuttgart.png',
  'BUN-FCU': 'Germany - Bundesliga/1.FC Union Berlin.png',
  'BUN-SVW': 'Germany - Bundesliga/SV Werder Bremen.png',

  // France — Ligue 1
  'FL1-ANG': 'France - Ligue 1/Angers SCO.png',
  'FL1-AUX': 'France - Ligue 1/AJ Auxerre.png',
  'FL1-SBR': 'France - Ligue 1/Stade Brestois 29.png',
  'FL1-HAC': 'France - Ligue 1/Le Havre AC.png',
  'FL1-LEM': 'France - Ligue 1/Le Mans FC.png',
  'FL1-RCL': 'France - Ligue 1/RC Lens.png',
  'FL1-LIL': 'France - Ligue 1/LOSC Lille.png',
  'FL1-FCL': 'France - Ligue 1/FC Lorient.png',
  'FL1-OL': 'France - Ligue 1/Olympique Lyon.png',
  'FL1-OM': 'France - Ligue 1/Olympique Marseille.png',
  'FL1-ASM': 'France - Ligue 1/AS Monaco.png',
  'FL1-NIC': 'France - Ligue 1/OGC Nice.png',
  'FL1-PFC': 'France - Ligue 1/Paris FC.png',
  'FL1-PSG': 'France - Ligue 1/Paris Saint-Germain.png',
  'FL1-REN': 'France - Ligue 1/Stade Rennais FC.png',
  'FL1-RCS': 'France - Ligue 1/RC Strasbourg Alsace.png',
  'FL1-TFC': 'France - Ligue 1/FC Toulouse.png',
  'FL1-EST': 'France - Ligue 1/ESTAC Troyes.png',
};

export const CLUB_CREST_URLS: Record<string, string> = Object.fromEntries(
  Object.entries(CREST_PATHS).map(([clubId, path]) => [clubId, crestUrl(path)]),
);

export function getClubCrestUrl(club: Pick<Club, 'club_id'> | null | undefined): string | null {
  if (!club) return null;
  return CLUB_CREST_URLS[club.club_id] ?? null;
}

/** Short-name lookup for awards, histories, and transfer payloads. */
const CLUB_ID_BY_SHORT: Record<string, string> = {
  ARS: 'EPL-ARS', AVL: 'EPL-AVL', BOU: 'EPL-BOU', BRE: 'EPL-BRE', BHA: 'EPL-BHA', CHE: 'EPL-CHE', COV: 'EPL-COV', CRY: 'EPL-CRY', EVE: 'EPL-EVE', FUL: 'EPL-FUL', HUL: 'EPL-HUL', IPS: 'EPL-IPS', LEE: 'EPL-LEE', LIV: 'EPL-LIV', MCI: 'EPL-MCI', MUN: 'EPL-MUN', NEW: 'EPL-NEW', NFO: 'EPL-NFO', SUN: 'EPL-SUN', TOT: 'EPL-TOT',
  ALA: 'LAL-ALA', ATH: 'LAL-ATH', ATM: 'LAL-ATM', BAR: 'LAL-BAR', CEL: 'LAL-CEL', DEP: 'LAL-DEP', ELC: 'LAL-ELC', ESP: 'LAL-ESP', GET: 'LAL-GET', LEV: 'LAL-LEV', MAL: 'LAL-MAL', OSA: 'LAL-OSA', RAC: 'LAL-RAC', RAY: 'LAL-RAY', BET: 'LAL-RBB', RMA: 'LAL-RMA', RSO: 'LAL-RSO', SEV: 'LAL-SEV', VAL: 'LAL-VAL', VIL: 'LAL-VIL',
  ATA: 'SEA-ATA', BOL: 'SEA-BOL', CAG: 'SEA-CAG', COM: 'SEA-COM', FIO: 'SEA-FIO', FRO: 'SEA-FRO', GEN: 'SEA-GEN', INT: 'SEA-INT', JUV: 'SEA-JUV', LAZ: 'SEA-LAZ', LEC: 'SEA-LEC', MIL: 'SEA-MIL', MON: 'SEA-MON', NAP: 'SEA-NAP', PAR: 'SEA-PAR', ROM: 'SEA-ROM', SAS: 'SEA-SAS', TOR: 'SEA-TOR', UDI: 'SEA-UDI', VEN: 'SEA-VEN',
  AUG: 'BUN-AUG', BAY: 'BUN-BAY', BMG: 'BUN-BMG', KOE: 'BUN-KOE', BVB: 'BUN-DOR', ELV: 'BUN-ELV', SGE: 'BUN-SGE', SCF: 'BUN-SCF', HSV: 'BUN-HSV', TSG: 'BUN-TSG', RBL: 'BUN-RBL', B04: 'BUN-B04', M05: 'BUN-M05', SCP: 'BUN-SCP', S04: 'BUN-S04', VFB: 'BUN-VFB', FCU: 'BUN-FCU', SVW: 'BUN-SVW',
  ANG: 'FL1-ANG', AUX: 'FL1-AUX', SBR: 'FL1-SBR', HAC: 'FL1-HAC', LEM: 'FL1-LEM', RCL: 'FL1-RCL', LIL: 'FL1-LIL', FCL: 'FL1-FCL', OL: 'FL1-OL', OM: 'FL1-OM', ASM: 'FL1-ASM', NIC: 'FL1-NIC', PFC: 'FL1-PFC', PSG: 'FL1-PSG', REN: 'FL1-REN', RCS: 'FL1-RCS', TFC: 'FL1-TFC', EST: 'FL1-EST',
};

export function getClubCrestUrlByShort(shortName: string | null | undefined): string | null {
  if (!shortName) return null;
  return CLUB_CREST_URLS[CLUB_ID_BY_SHORT[shortName.toUpperCase()]] ?? null;
}
