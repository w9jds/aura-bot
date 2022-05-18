package main

import (
	"fmt"

	esi "github.com/w9jds/go.esi"
)

type CharacterRef struct {
	CharacterID uint
	AllianceID  uint
	CorpID      uint

	Display string
}

func (ref CharacterRef) getAffiliation() uint {
	var affiliation uint

	if ref.CorpID != 0 {
		affiliation = ref.CorpID
	}

	if ref.AllianceID != 0 {
		affiliation = ref.AllianceID
	}

	return affiliation
}

func (ref *CharacterRef) getLinks(names map[uint]esi.NameRef) {
	var affiliation string
	characterLink := fmt.Sprintf("[%s](https://zkillboard.com/character/%d)", names[ref.CharacterID].Name, ref.CharacterID)

	if ref.CorpID != 0 {
		affiliation = fmt.Sprintf("[%s](https://zkillboard.com/corporation/%d)", names[ref.CorpID].Name, ref.CorpID)
	}

	if ref.AllianceID != 0 {
		affiliation = fmt.Sprintf("[%s](https://zkillboard.com/alliance/%d)", names[ref.AllianceID].Name, ref.AllianceID)
	}

	ref.Display = fmt.Sprintf("%s\n%s", characterLink, affiliation)
}
