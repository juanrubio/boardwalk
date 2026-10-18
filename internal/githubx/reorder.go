package githubx

// PositionAfterID computes the afterId argument for moving itemID one
// step within its grouping column: dir -1 = up (swap with the item
// above), +1 = down. groupName "" treats the whole list as one
// column. ok=false at the column ends (or unknown item); a returned
// afterID of "" with ok=true means "move to the very top".
func PositionAfterID(items []Item, groupName, itemID string, dir int) (afterID string, ok bool) {
	if dir == 0 {
		return "", false
	}
	targetIdx := -1
	for i := range items {
		if items[i].ID == itemID {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return "", false
	}

	optionOf := func(it Item) string {
		v, _ := it.Value(groupName)
		return v.OptionName
	}
	targetOption := ""
	if groupName != "" {
		targetOption = optionOf(items[targetIdx])
	}

	// The target's column, in the given (global) order.
	var column []Item
	colIdx := -1
	for _, it := range items {
		if groupName != "" && optionOf(it) != targetOption {
			continue
		}
		if it.ID == itemID {
			colIdx = len(column)
		}
		column = append(column, it)
	}

	neighbourIdx := colIdx + dir
	if neighbourIdx < 0 || neighbourIdx >= len(column) {
		return "", false
	}
	neighbour := column[neighbourIdx]

	if dir > 0 {
		return neighbour.ID, true
	}
	// Up: sit immediately before the neighbour in the global order,
	// i.e. after whatever precedes it there (any column), or at the
	// very top when the neighbour is first overall.
	for i := range items {
		if items[i].ID == neighbour.ID {
			if i == 0 {
				return "", true
			}
			return items[i-1].ID, true
		}
	}
	return "", false
}

// ApplyReorder returns items with itemID removed and re-inserted
// after afterID (index 0 when afterID==""). Unknown IDs: unchanged copy.
func ApplyReorder(items []Item, itemID, afterID string) []Item {
	out := make([]Item, len(items))
	copy(out, items)

	itemIdx := -1
	for i := range items {
		if items[i].ID == itemID {
			itemIdx = i
			break
		}
	}
	if itemIdx < 0 || afterID == itemID {
		return out
	}
	if afterID != "" {
		found := false
		for i := range items {
			if items[i].ID == afterID {
				found = true
				break
			}
		}
		if !found {
			return out
		}
	}

	moved := items[itemIdx]
	rest := make([]Item, 0, len(items)-1)
	rest = append(rest, items[:itemIdx]...)
	rest = append(rest, items[itemIdx+1:]...)

	insertAt := 0
	if afterID != "" {
		for i := range rest {
			if rest[i].ID == afterID {
				insertAt = i + 1
				break
			}
		}
	}
	result := make([]Item, 0, len(items))
	result = append(result, rest[:insertAt]...)
	result = append(result, moved)
	result = append(result, rest[insertAt:]...)
	return result
}
