/**
 * English catalogue (F13 i18n). Keys are the stable contract; the goal is
 * decoupling copy from components, not translations — English ships first
 * and additional locales must cover every key here (enforced by
 * assertCatalogueComplete).
 */
export const en = {
  // Primary navigation tabs.
  'nav.home': 'Home',
  'nav.match': 'Match Centre',
  'nav.inbox': 'News',
  'nav.league': 'Tables',
  'nav.competitions': 'Competitions',
  'nav.history': 'History',
  'nav.squads': 'Clubs',
  'nav.players': 'Players',
  'nav.transfers': 'Transfers',
  'nav.lab': 'Wonderkids',
  'nav.stats': 'Statistics',

  // Sidebar section groups.
  'navgroup.season': 'Season',
  'navgroup.competitions': 'Competitions',
  'navgroup.market': 'Clubs & Market',
  'navgroup.future': 'The Future',

  // Simulation controls.
  'action.continue': 'Continue',
  'action.week': 'Week',
  'action.month': 'Month',
  'action.season': 'Season',
  'action.advancing': 'Advancing…',

  // Career actions.
  'action.newCareer': 'New career',
  'action.saveSlots': 'Save slots',
  'action.archiveNow': 'Archive now',
  'action.import': 'Import',
  'action.rename': 'Rename',
  'action.duplicate': 'Duplicate',
  'action.delete': 'Delete',
  'action.cancel': 'Cancel',
  'action.save': 'Save',

  // Data export.
  'action.exportCsv': 'Export CSV',
  'action.exportJson': 'Export JSON',

  // What-if sandbox.
  'action.runWhatIf': 'Run what-if',
  'action.rerollSeed': 'Reroll scratch seed',
  'whatif.title': 'What-if sandbox',
  'whatif.simulationOnly': 'Simulation only',

  // Common states.
  'common.loading': 'Loading…',
  'common.retry': 'Retry',
} as const;

export type MessageKey = keyof typeof en;
export type MessageParams = Record<string, string | number>;
