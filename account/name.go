package account

import "strings"

type AccountName string

func (an *AccountName) IsValid() bool {
	return true
}
func (an *AccountName) IsChild(other AccountName) bool {
	if !other.IsValid() {
		return false
	}

	return strings.HasSuffix(string(other), string(*an))
}
