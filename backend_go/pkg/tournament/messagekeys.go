package tournament

// Message keys (F13 i18n): the stable contract for backend-generated prose.
// The English text stays with its generator; these constants name every
// inbox category the world emits so clients and future catalogues key on
// symbols instead of string literals. Values are the persisted wire
// contract — never change them, only add new ones.
const (
	MsgCategoryMatch     = "match"
	MsgCategoryTransfer  = "transfer"
	MsgCategoryWonderkid = "wonderkid"
	MsgCategoryHonour    = "honour"
	MsgCategoryRace      = "race"
	MsgCategoryCup       = "cup"
	MsgCategorySystem    = "system"
	MsgCategoryInjury    = "injury"
	MsgCategoryDugout    = "dugout"
	MsgCategoryYouth     = "youth"
	MsgCategoryNXGN      = "nxgn"
	MsgCategoryMilestone = "milestone"
	MsgCategoryManager   = "manager"
	MsgCategoryWatch     = "watch"
	MsgCategoryClub      = "club"
)

// InboxCategories lists every category constant in a deterministic order.
// The frontend InboxItem.category union and the i18n catalogue mirror this
// list; TestInboxCategoriesAreTheWireContract keeps them honest.
var InboxCategories = []string{
	MsgCategoryMatch,
	MsgCategoryTransfer,
	MsgCategoryWonderkid,
	MsgCategoryHonour,
	MsgCategoryRace,
	MsgCategoryCup,
	MsgCategorySystem,
	MsgCategoryInjury,
	MsgCategoryDugout,
	MsgCategoryYouth,
	MsgCategoryNXGN,
	MsgCategoryMilestone,
	MsgCategoryManager,
	MsgCategoryWatch,
	MsgCategoryClub,
}

// Achievement IDs are message keys too: the milestone ledger's stable
// identifiers (see achievementDefinitions) double as the catalogue keys for
// achievement prose, so the frontend can resolve titles and descriptions
// from a locale catalogue without new backend fields.
