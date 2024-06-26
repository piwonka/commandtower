package main

import (
	"C"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.design/x/clipboard"
	"strconv"
)

// GetImageResource
// Transforms an image URI into a fyne Resource for usage inside the UI
// Params: the image URI as a string
// Returns: a fyne.Resource representing the Image
func GetImageResource(imageUri string) fyne.Resource {
	r, err := fyne.LoadResourceFromURLString(imageUri) // load image
	if err != nil {                                    // if image can not be loaded attempt to load a placeholder
		return resourcePlaceholderPng
	}
	return r
}

func NewCheckboxWithIcon(resource *fyne.StaticResource) (*widget.Check, *fyne.Container) { // TODO: create a custom widget for this

	checkBox := widget.NewCheck("", nil)
	img := canvas.NewImageFromResource(resource)
	img.FillMode = canvas.ImageFillOriginal
	return checkBox, container.NewVBox(checkBox, img)
}

func GetSelectedChoices(choiceColorMap map[*widget.Check]string) []string {
	result := make([]string, 0)
	for check, color := range choiceColorMap {
		if check.Checked {
			result = append(result, color)
		}
	}
	return result
}

// The main function that is excecuted
// Params: None
// Returns: Nothing
func main() {
	// initialize session State
	state := SessionState{
		commanderCount:          -1, // -1 == we don't have any commanders; 0 == we have a commander and its index in the cache is 0; ...
		backSteps:               0,
		currentCardFace:         0,
		prevCommanderNames:      make([]string, 0),
		prevCommanderImages:     make([]fyne.Resource, 0),
		prevCommanderDecklists:  make([]string, 0),
		prevCommanderDeckPrices: make([]float64, 0),
	}

	// init clipboard access
	err := clipboard.Init()

	if err != nil { // end execution with error if clipboard access is blocked
		//panic(err)
	}

	// Build Main View Objects

	// init window
	myApp := app.New()
	w := myApp.NewWindow("Command Tower")

	// init search field
	searchQuery := widget.NewEntry()
	searchQuery.PlaceHolder = "Scryfall Search Query"

	// init checkBoxes
	choices := container.NewHBox()
	choiceColorMap := make(map[*widget.Check]string)
	resources := []*fyne.StaticResource{resourceWSvg, resourceBSvg, resourceUSvg, resourceRSvg, resourceGSvg, resourceCSvg, resourceESvg} // TODO: fix this hack, there must be a way to reference these resources by their StaticName
	for i, color := range []string{"w", "b", "u", "r", "g", "c", "e"} {
		checkBox, checkContainer := NewCheckboxWithIcon(resources[i])
		choiceColorMap[checkBox] = color
		choices.Add(checkContainer)
	}

	// Image
	img := canvas.NewImageFromResource(nil)
	img.Resize(fyne.NewSize(480, 680))
	img.FillMode = canvas.ImageFillOriginal
	clickableImage := NewClickableImage(img, func() {
		res := GetOtherCardFaceForCurrentCard(&state)
		if res != nil {
			img.Resource = res
			img.Refresh()
		}
	})
	// Price Checking
	priceContainer := container.NewCenter()
	// price label
	price := binding.NewString()
	priceLabel := widget.NewLabel("")
	priceLabel.Bind(price)

	// price Button
	priceCheck := widget.NewButton("Check Price", func() {
		p := GetCurrentDeckPrice(&state)
		price.Set(strconv.FormatFloat(p, 'f', 2, 64) + "€")
		priceContainer.RemoveAll()
		priceContainer.Add(priceLabel)
	})
	priceContainer.Add(priceCheck)

	// Buttons
	// Previous
	previous := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		image := GetPreviousCommanderData(&state)
		if image != nil { // if there is a previous commander
			clickableImage.image.Resource = image
			clickableImage.image.Refresh()
			priceContainer.RemoveAll()
			priceContainer.Add(priceCheck)
		}

	})
	// Get Decklist
	get := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		deckList := GetCurrentDeckList(&state)
		clipboard.Write(clipboard.FmtText, []byte(deckList))
	})
	//Next
	next := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		image := GetNextCommanderData(&state, GetSelectedChoices(choiceColorMap), searchQuery.Text)
		clickableImage.image.Resource = image
		clickableImage.image.Refresh()
		priceContainer.RemoveAll()
		priceContainer.Add(priceCheck)
	})

	buttons := container.NewCenter(container.NewHBox(previous, get, next))
	vBox := container.NewVBox(searchQuery, clickableImage, container.NewCenter(choices), buttons, priceContainer)
	w.SetContent(vBox)
	w.SetIcon(resourceIconPng)

	// Load initial state
	// pull any first commander image
	image := GetNextCommanderData(&state, GetSelectedChoices(choiceColorMap), searchQuery.Text) // get any first commander (nothing selected)
	// Set the Image inside the View and Refresh
	clickableImage.image.Resource = image
	clickableImage.image.Refresh()

	w.ShowAndRun()
}
