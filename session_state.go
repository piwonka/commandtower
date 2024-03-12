package main

import (
	"fyne.io/fyne/v2"
)

type SessionState struct {
	commanderCount         int
	backSteps              int
	prevCommanderImages    []fyne.Resource
	prevCommanderDecklists []string
}

func GetCurrentCommanderData(state *SessionState) (fyne.Resource, string) {
	if len(state.prevCommanderImages) > 0 {
		deckList := state.prevCommanderDecklists[state.commanderCount-state.backSteps]
		resource := state.prevCommanderImages[state.commanderCount-state.backSteps]
		return resource, deckList
	} else {
		return nil, ""
	}
}

func GetPreviousCommanderData(state *SessionState) (fyne.Resource, string) {
	if state.commanderCount > 0 && state.backSteps < state.commanderCount {
		state.backSteps += 1
		resource, deckList := GetCurrentCommanderData(state)
		return resource, deckList
	} else {
		return nil, ""
	}
}

func AddNewCommanderDataToCache(state *SessionState, image fyne.Resource, deckList string) {
	state.commanderCount += 1
	state.prevCommanderImages = append(state.prevCommanderImages, image)
	state.prevCommanderDecklists = append(state.prevCommanderDecklists, deckList)
}

func GetNextCommanderData(state *SessionState, selected []string, queryEntry string) (fyne.Resource, string) {
	if state.backSteps == 0 {
		imageUri, deckList := GetCommanderImageAndDecklist(selected, queryEntry) // get any first commander (nothing selected)
		image := GetImageResource(imageUri)
		AddNewCommanderDataToCache(state, image, deckList)
		return image, deckList
	} else {
		state.backSteps -= 1
		deckList := state.prevCommanderDecklists[state.commanderCount-state.backSteps]
		resource := state.prevCommanderImages[state.commanderCount-state.backSteps]
		return resource, deckList
	}
}
