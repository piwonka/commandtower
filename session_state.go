package main

import (
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"os"
	"strings"
)

type SessionState struct {
	commanderCount          int
	backSteps               int
	prevCommanderNames      []string
	prevCommanderImages     []fyne.Resource
	prevCommanderDecklists  []string
	prevCommanderDeckPrices []float64
}

func GetPreviousCommanderData(state *SessionState) fyne.Resource {
	if state.commanderCount > 0 && state.backSteps < state.commanderCount {
		state.backSteps += 1
		if len(state.prevCommanderImages) > 0 {
			resource := state.prevCommanderImages[state.commanderCount-state.backSteps]
			return resource
		} else {
			return nil
		}
	} else {
		return nil
	}
}

func AddNewCommanderDataToCache(state *SessionState, name string, image fyne.Resource) {
	state.commanderCount += 1
	state.prevCommanderNames = append(state.prevCommanderNames, name)
	state.prevCommanderImages = append(state.prevCommanderImages, image)
	state.prevCommanderDecklists = append(state.prevCommanderDecklists, "")    // add placeholder values for now
	state.prevCommanderDeckPrices = append(state.prevCommanderDeckPrices, 0.0) // add placeholder values for now
}

func GetNextCommanderData(state *SessionState, selected []string, queryEntry string) fyne.Resource {
	if state.backSteps == 0 {
		name, imageUri := GetCommanderFromScryfall(selected, queryEntry) // get any first commander (nothing selected)
		fmt.Println(name + " : " + imageUri)
		image := GetImageResource(imageUri)
		AddNewCommanderDataToCache(state, name, image)
		return image
	} else {
		state.backSteps -= 1
		resource := state.prevCommanderImages[state.commanderCount-state.backSteps]
		return resource
	}
}
func GetCurrentDeckList(state *SessionState) string {
	if len(state.prevCommanderImages) > 0 {
		deckList := state.prevCommanderDecklists[state.commanderCount-state.backSteps]
		if deckList != "" {
			return deckList
		} else {
			deckList, err := GetEDHRecAvgDecklist(state.prevCommanderNames[state.commanderCount-state.backSteps])
			if err == nil {
				state.prevCommanderDecklists[state.commanderCount-state.backSteps] = deckList
				return deckList
			}
		}
	}
	return ""
}
func GetCurrentDeckPrice(state *SessionState) float64 {
	if len(state.prevCommanderImages) > 0 {
		price := state.prevCommanderDeckPrices[state.commanderCount-state.backSteps]
		if price != 0.0 {
			return price
		} else {
			deckList := GetCurrentDeckList(state)
			if deckList == "" {
				return 0.0
			} else {
				price := GetScryfallPricingData(strings.Split(deckList, "\n"), 2)
				state.prevCommanderDeckPrices[state.commanderCount-state.backSteps] = price
				return price
			}
		}
	}
	return 0.0
}

func PersistCompleteDataSets(state *SessionState) {
	cacheDir, err := os.UserCacheDir()
	if err == nil {
		// Check if cache directory exists and if not create it
		if _, err = os.Stat(cacheDir + string(os.PathSeparator) + "CommandTower"); errors.Is(err, os.ErrNotExist) {
			os.Mkdir(cacheDir+string(os.PathSeparator)+"CommandTower", os.ModePerm)
		}
		// Check if cache exists and if not create it
		if _, err = os.Stat(cacheDir + string(os.PathSeparator) + "CommandTower" + string(os.PathListSeparator) + "commander_data.json"); errors.Is(err, os.ErrNotExist) {
			os.Create(cacheDir + string(os.PathSeparator) + "CommandTower" + string(os.PathListSeparator) + "commander_data.json")
		}
		for i := range state.commanderCount {
			if state.prevCommanderImages[i] != nil && state.prevCommanderDecklists != nil {
				// TODO: ADD MARSHALLING
			}
		}
	}
}
