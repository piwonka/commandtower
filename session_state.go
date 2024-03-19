package main

import (
	"fyne.io/fyne/v2"
)

type SessionState struct {
	commanderCount          int
	backSteps               int
	prevCommanderImages     []fyne.Resource
	prevCommanderDecklists  []string
	prevCommanderDeckPrices []float64
}

func GetCurrentCommanderData(state *SessionState) (fyne.Resource, string, float64) {
	if len(state.prevCommanderImages) > 0 {
		deckList := state.prevCommanderDecklists[state.commanderCount-state.backSteps]
		resource := state.prevCommanderImages[state.commanderCount-state.backSteps]
		price := state.prevCommanderDeckPrices[state.commanderCount-state.backSteps]
		return resource, deckList, price
	} else {
		return nil, "", 0.0
	}
}

func GetPreviousCommanderData(state *SessionState) (fyne.Resource, string, float64) {
	if state.commanderCount > 0 && state.backSteps < state.commanderCount {
		state.backSteps += 1
		resource, deckList, price := GetCurrentCommanderData(state)
		return resource, deckList, price
	} else {
		return nil, "", 0.0
	}
}

func AddNewCommanderDataToCache(state *SessionState, image fyne.Resource, deckList string, price float64) {
	state.commanderCount += 1
	state.prevCommanderImages = append(state.prevCommanderImages, image)
	state.prevCommanderDecklists = append(state.prevCommanderDecklists, deckList)
	state.prevCommanderDeckPrices = append(state.prevCommanderDeckPrices, price)
}

func GetNextCommanderData(state *SessionState, selected []string, queryEntry string) (fyne.Resource, string, float64) {
	if state.backSteps == 0 {
		imageUri, deckList, price := GetCommanderImageAndDecklist(selected, queryEntry) // get any first commander (nothing selected)
		image := GetImageResource(imageUri)
		AddNewCommanderDataToCache(state, image, deckList, price)
		return image, deckList, price
	} else {
		state.backSteps -= 1
		deckList := state.prevCommanderDecklists[state.commanderCount-state.backSteps]
		resource := state.prevCommanderImages[state.commanderCount-state.backSteps]
		price := state.prevCommanderDeckPrices[state.commanderCount-state.backSteps]
		return resource, deckList, price
	}
}
