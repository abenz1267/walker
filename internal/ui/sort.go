package ui

import (
	"slices"
	"strings"

	"github.com/abenz1267/walker/internal/util"
)

func sortEntries(entries []util.Entry) {
	slices.SortFunc(entries, func(a, b util.Entry) int {
		if tahAcceptedIdentifier != "" {
			if a.Identifier() == tahAcceptedIdentifier {
				return -1
			}

			if b.Identifier() == tahAcceptedIdentifier {
				return 1
			}
		}

		text := elements.input.Text()

		if text == "" {
			if a.Matching == util.AlwaysTopOnEmptySearch && b.Matching != util.AlwaysTopOnEmptySearch {
				return -1
			}

			if b.Matching == util.AlwaysTopOnEmptySearch && a.Matching != util.AlwaysTopOnEmptySearch {
				return 1
			}
		}

		if a.Matching == util.AlwaysTop && b.Matching != util.AlwaysTop {
			return -1
		}

		if b.Matching == util.AlwaysTop && a.Matching != util.AlwaysTop {
			return 1
		}

		if a.Matching == util.AlwaysBottom && b.Matching != util.AlwaysBottom {
			return 1
		}

		if b.Matching == util.AlwaysBottom && a.Matching != util.AlwaysBottom {
			return -1
		}

		if !a.LastUsed.IsZero() && !b.LastUsed.IsZero() {
			if a.OpenWindows > b.OpenWindows {
				return 1
			}

			if a.OpenWindows < b.OpenWindows {
				return -1
			}
		}

		if text != "" {
			min := a.ScoreFinal - 50
			max := a.ScoreFinal + 50

			if min < b.ScoreFinal && b.ScoreFinal < max {
				if a.Module != b.Module {
					if a.Weight > b.Weight {
						return -1
					}

					if a.Weight < b.Weight {
						return 1
					}
				}

				if a.File == b.File {
					if a.Prefer && !b.Prefer {
						return -1
					}

					if !a.Prefer && b.Prefer {
						return 1
					}
				}
			}
		}

		if a.ScoreFinal == b.ScoreFinal {
			if !a.LastUsed.IsZero() && !b.LastUsed.IsZero() {
				return b.LastUsed.Compare(a.LastUsed)
			}

			return strings.Compare(a.Label, b.Label)
		}

		if a.ScoreFinal > b.ScoreFinal {
			return -1
		}

		if a.ScoreFinal < b.ScoreFinal {
			return 1
		}

		return 0
	})
}
