package main

import (
	"C"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/tidwall/gjson"
	"golang.design/x/clipboard"
)

var PLACEHOLDER_IMAGE string = "https://static.wikia.nocookie.net/mtgsalvation_gamepedia/images/f/f8/Magic_card_back.jpg/revision/latest?cb=20140813141013"

func GetCommanderImageAndDecklist(selected_colors []string) (string, string) {
	fmt.Println("Building Query...")
	var query string = BuildScryfallCommanderQuery(selected_colors)
	fmt.Println("Retrieving Commander with Query: " + query)
	commanderData, err := GetScryfallCommanderData(query)
	if err != nil {
		return PLACEHOLDER_IMAGE, "" // return a placeholder image and no decklist
	} else {
		card_name, image_uri := ParseScryfallData(commanderData)
		deck_list, err := GetEDHRecAvgDecklist(card_name)
		if err != nil {
			return PLACEHOLDER_IMAGE, "" // return a placeholder image and no decklist
		} else {
			return image_uri, deck_list
		}
	}
}

func GetEDHRecAvgDecklist(commander string) (string, error) {
	// TODO: add computation of session token instead of hardcoded one ( this will work for about 24 hours after writing )
	avg_deck_endpoint := "https://edhrec.com/_next/data/7-TtnLfoAX_AgebfCokAf/average-decks/" + commander + ".json?commanderName=" + commander
	fmt.Println("Retrieving Deck from: " + avg_deck_endpoint)
	page_resp, err := http.Get(avg_deck_endpoint) // TODO: add error handling
	if err != nil {
		return "", err
	} else {
		page_bytes, err := io.ReadAll(page_resp.Body) // TODO: add error handling
		defer page_resp.Body.Close()
		if err != nil {
			return "", err
		} else {
			page_json := string(page_bytes)
			fmt.Println(page_json)
			deck_json := gjson.Get(page_json, "pageProps.data.deck").String()
			fmt.Println(deck_json)
			deck_json = strings.ReplaceAll(deck_json, "\",\"", "\n")
			return deck_json[2 : len(deck_json)-2], nil
		}
	}
}

func BuildScryfallCommanderQuery(selected_colors []string) string {
	var query string = "https://api.scryfall.com/cards/random?q="
	query += url.QueryEscape("is:Commander")                                                     // we only query for commanders
	if len(selected_colors) == 0 || len(selected_colors) == 1 && selected_colors[0] == "Exact" { // if nothing is selected or only the exact box is selected we dont add colors to the query
		return query
	} else { // if colors are selected
		var colors string = "<="            // assume non-exact matches
		for _, c := range selected_colors { // add each selected color to the query
			fmt.Println(c + " is selected.")
			switch c {
			case "White":
				colors += "W"
			case "Black":
				colors += "B"
			case "Blue":
				colors += "U"
			case "Red":
				colors += "R"
			case "Green":
				colors += "G"
			case "Exact": // if exact matches are wanted, remove the '<' from the colors string
				colors = colors[1:]
			}
		}
		return query + url.QueryEscape(" color"+colors)
	}
}

func ParseScryfallData(json_data string) (string, string) {
	// get the cards name
	var card_name string = gjson.Get(json_data, "name").String()
	// format the card name to EDHREC URL format
	fmt.Println("Retrieved Commander: " + card_name)
	var replacer strings.Replacer = *strings.NewReplacer(
		" ", "-",
		",", "",
		"'", "")

	var formatted_card_name string = strings.ToLower(replacer.Replace(card_name))
	// get uri of card image
	image_uri := gjson.Get(json_data, "image_uris.normal").String()

	return formatted_card_name, image_uri
}

func GetScryfallCommanderData(query string) (string, error) {
	response, err := http.Get(query) // request json data from the scryfall rest api
	if err != nil {                  // if this fails return the error
		return "", err
	} else { // if we got json data
		body, err := io.ReadAll(response.Body) // read the data to a byte array
		defer response.Body.Close()            // close the Reader once this function returns
		if err != nil {                        // if we cant read the response, return the error
			return "", err
		} else { // if we read the response successfully
			return string(body), nil // return the json string
		}
	}
}

func getImageResource(image_uri string) fyne.Resource {
	r, err := fyne.LoadResourceFromURLString(image_uri) // load image
	if err != nil {                                     // if image can not be loaded attempt to load a placeholder
		r, err = fyne.LoadResourceFromURLString(PLACEHOLDER_IMAGE)
		if err != nil { // if placeholder cant be loaded either, end the execution with error
			//panic(err)
		}
		return r
	}
	return r
}

func main() {
	// init clipboard access
	err := clipboard.Init()
	if err != nil { // end execution with error if clipboard access is blocked
		//panic(err)
	}

	// VIEW
	// init window
	myApp := app.New()
	w := myApp.NewWindow("Commander")

	// init checkBoxes´ß
	colors := []string{"White", "Black", "Blue", "Red", "Green", "Exact"}
	checkGroup := widget.NewCheckGroup(colors, nil)
	checkGroup.Horizontal = true
	checkBoxes := container.NewHBox(container.NewCenter(checkGroup))

	// pull any first commander image
	image_uri, deck_list := GetCommanderImageAndDecklist(checkGroup.Selected) // get any first commander (nothing selected)
	// load deck into clipboard
	clipboard.Write(clipboard.FmtText, []byte(deck_list))
	resource := getImageResource(image_uri)
	img := canvas.NewImageFromResource(resource)
	img.FillMode = canvas.ImageFillOriginal

	// Buttons
	new_commander := widget.NewButton("New Commander", func() {
		image_uri, deck_list := GetCommanderImageAndDecklist(checkGroup.Selected)
		clipboard.Write(clipboard.FmtText, []byte(deck_list))
		resource := getImageResource(image_uri)
		img.Resource = resource
		img.Refresh()
	})

	vBox := container.NewVBox(img, checkBoxes, new_commander)
	w.SetContent(vBox)
	w.ShowAndRun()
}
