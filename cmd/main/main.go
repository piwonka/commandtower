package main

import (
	"C"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/tidwall/gjson"
	"golang.design/x/clipboard"
)
import "strconv"

var PLACEHOLDER_IMAGE string = "https://static.wikia.nocookie.net/mtgsalvation_gamepedia/images/f/f8/Magic_card_back.jpg/revision/latest?cb=20140813141013"
var EDHREC_BASE_URL string = "https://edhrec.com"

// Selects a random commander depending on the input constraints and fetches an image and a decklist for said commander
// Params: An Array of strings containing all currently selected color checkboxes (and the "Exact" checkbox) from the UI
// Returns: A Tuple of strings, the first of which being the image URI of the commander and the second being the decklist for the commander
func GetCommanderImageAndDecklist(selected_colors []string) (string, string) {
	fmt.Println("Building Query...")
	var query string = BuildScryfallCommanderQuery(selected_colors)
	fmt.Println("Retrieving Commander with Query: " + query)
	commanderData, err := GetScryfallCommanderData(query)
	fmt.Println(commanderData)
	if err != nil {
		return PLACEHOLDER_IMAGE, "" // return a placeholder image and no decklist
	} else {
		card_name, image_uri := ParseScryfallData(commanderData)
		fmt.Println(card_name + " : " + image_uri)
		deck_list, err := GetEDHRecAvgDecklist(card_name)
		if err != nil {
			return PLACEHOLDER_IMAGE, "" // return a placeholder image and no decklist
		} else {
			return image_uri, deck_list
		}
	}
}

// Retrieves the average decklist for a given commander name from EDHRec.com
// Params: The name of the commander the decklist shall be retrieved for
// Returns: a Tuple containing a string, representing the average decklist for the commander and an error that is nil unless the retrieval was unsuccessful
func GetEDHRecAvgDecklist(commander string) (string, error) {
	avg_deck_endpoint := EDHREC_BASE_URL + "/_next/data/" + GetBuildId() + "/average-decks/" + commander + ".json?commanderName=" + commander // 7-TtnLfoAX_AgebfCokAf
	fmt.Println("Retrieving Deck from: " + avg_deck_endpoint)
	page_resp, err := http.Get(avg_deck_endpoint)
	if err != nil {
		return "", err
	} else {
		page_bytes, err := io.ReadAll(page_resp.Body)
		defer page_resp.Body.Close()
		if err != nil {
			return "", err
		} else {
			page_json := string(page_bytes)
			num_decks_value := gjson.Get(page_json, "pageProps.data.num_decks").String() // if num_decks does not exist
			if num_decks_value == "" {
				fmt.Println("REST")
				deck_json := gjson.Get(page_json, "pageProps.data.deck").String()
				fmt.Println(deck_json)
				deck_json = strings.ReplaceAll(deck_json, "\",\"", "\n")
				if len(deck_json) > 4 {
					return deck_json[2 : len(deck_json)-2], nil
				} else {
					return "", errors.New("The deck inside the response was empty")
				}
			}
			_, err := strconv.Atoi(num_decks_value)
			if err != nil {
				fmt.Println("ERROR" + err.Error())
				return "", err
			} else {
				fmt.Println("NO DECKS")
				return "", errors.New("no decks found for commander: " + commander)
			}
		}
	}
}

// Builds a Scryfall.com query using the selected constraints from the UI
// Params: An Array of strings containing all currently selected color checkboxes (and the "Exact" checkbox) from the UI
// Returns :  The complete scryfall api request with the query as a string
func BuildScryfallCommanderQuery(selected_colors []string) string {
	var query string = "https://api.scryfall.com/cards/random?q="
	query += url.QueryEscape("is:Commander (game:paper) legal:commander (type:creature OR type:planeswalker)") // we only query for commanders
	if len(selected_colors) == 0 || len(selected_colors) == 1 && selected_colors[0] == "Exact" {               // if nothing is selected or only the exact box is selected we dont add colors to the query
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

// Parses the JSON data retrieved from a Scryfall card query and returns the pre-formatted cardname for EDHRec queries and the URI for a normal sized image of the card
// Params: The json_data from the Scryfall API response as a string
// Return: A tuple of strings containing the formatted card name and the image URI
func ParseScryfallData(json_data string) (string, string) {
	// get the cards name
	var card_name string = gjson.Get(json_data, "name").String()
	// format the card name to EDHREC URL format
	fmt.Println("Retrieved Commander: " + card_name)
	var replacer strings.Replacer = *strings.NewReplacer(
		" ", "-",
		",", "",
		"'", "",
	)

	var formatted_card_name string = strings.ToLower(replacer.Replace(card_name))
	first_card_name, _, found := strings.Cut(formatted_card_name, "-//")
	image_uri := gjson.Get(json_data, "image_uris.normal").String()
	if found { // if the card is double faced we get only the first card image
		formatted_card_name = first_card_name
		image_uri = gjson.Get(json_data, "card_faces.0.image_uris.normal").String()
	}

	return formatted_card_name, image_uri
}

// Sends a HTTP GET request to scryfall using the given query and returns the response
// Params: The url + query for the Scryfall API as a string
// Returns a tuple containing the entire json response and an error if the retrieval failed. (+ empty string)
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

// Transforms an image URI into a fyne Resource for usage inside the UI
// Params: the image URI as a string
// Returns: a fyne.Resource representing the Image
func GetImageResource(image_uri string) fyne.Resource {
	r, err := fyne.LoadResourceFromURLString(image_uri) // load image
	if err != nil {                                     // if image can not be loaded attempt to load a placeholder
		r, err = fyne.LoadResourceFromURLString(PLACEHOLDER_IMAGE)
		if err != nil { // if placeholder cant be loaded either, end the execution with error TODO: Download placeholder image, have it inside the program, do not retrieve it from online to secure failure case
			panic(err)
		}
		return r
	}
	return r
}

// Creates a valid buildId needed for EDHRec queries
// Param: None
// Return: A valid buildId as a String
func GetBuildId() string {
	response, err := http.Get(EDHREC_BASE_URL)
	if err != nil {
		return "7-TtnLfoAX_AgebfCokAf"
	} else {
		body, err := io.ReadAll(response.Body) // read the data to a byte array
		defer response.Body.Close()            // close the Reader once this function returns
		if err != nil {                        // if we cant read the response, return the error
			return "7-TtnLfoAX_AgebfCokAf"
		} else { // if we read the response successfully
			script_block_regex := regexp.MustCompile("<script id=\"__NEXT_DATA__\" type=\"application/json\">(.*)</script>")
			match := script_block_regex.Find(body)
			props_data := string(match)
			id := gjson.Get(props_data, "buildId").String()
			fmt.Println("ID EQUALS: " + id)
			return id
		}
	}
}

// The main function that is excecuted
// Params: None
// Returns: Nothing
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
	resource := GetImageResource(image_uri)
	img := canvas.NewImageFromResource(resource)
	img.FillMode = canvas.ImageFillOriginal

	// Buttons
	new_commander := widget.NewButton("New Commander", func() {
		image_uri, deck_list := GetCommanderImageAndDecklist(checkGroup.Selected)
		clipboard.Write(clipboard.FmtText, []byte(deck_list))
		resource := GetImageResource(image_uri)
		img.Resource = resource
		img.Refresh()
	})

	vBox := container.NewVBox(img, checkBoxes, new_commander)
	w.SetContent(vBox)
	w.ShowAndRun()
}
