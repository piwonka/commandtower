package main

import (
	"C"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/tidwall/gjson"
	"golang.design/x/clipboard"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)
import "strconv"

var PlaceholderImage string = "https://static.wikia.nocookie.net/mtgsalvation_gamepedia/images/f/f8/Magic_card_back.jpg/revision/latest?cb=20140813141013"
var EdhrecBaseUrl string = "https://edhrec.com"

// GetCommanderImageAndDecklist
// Selects a random commander depending on the input constraints and fetches an image and a decklist for said commander
// Params: An Array of strings containing all currently selected color checkboxes (and the "Exact" checkbox) from the UI
// Returns: A Tuple of strings, the first of which being the image URI of the commander and the second being the decklist for the commander
func GetCommanderImageAndDecklist(selectedColors []string) (string, string) {
	fmt.Println("Building Query...")
	var query string = BuildScryfallCommanderQuery(selectedColors)
	fmt.Println("Retrieving Commander with Query: " + query)
	commanderData, err := GetScryfallCommanderData(query)
	fmt.Println(commanderData)
	if err != nil {
		return PlaceholderImage, "" // return a placeholder image and no decklist
	} else {
		cardName, imageUri := ParseScryfallData(commanderData)
		fmt.Println(cardName + " : " + imageUri)
		deckList, err := GetEDHRecAvgDecklist(cardName)
		if err != nil {
			return PlaceholderImage, "" // return a placeholder image and no decklist
		} else {
			return imageUri, deckList
		}
	}
}

// GetEDHRecAvgDecklist
// Retrieves the average decklist for a given commander name from EDHRec.com
// Params: The name of the commander the decklist shall be retrieved for
// Returns: a Tuple containing a string, representing the average decklist for the commander and an error that is nil unless the retrieval was unsuccessful
func GetEDHRecAvgDecklist(commander string) (string, error) {
	avgDeckEndpoint := EdhrecBaseUrl + "/_next/data/" + GetBuildId() + "/average-decks/" + commander + ".json?commanderName=" + commander // 7-TtnLfoAX_AgebfCokAf
	fmt.Println("Retrieving Deck from: " + avgDeckEndpoint)
	pageResp, err := http.Get(avgDeckEndpoint)
	if err != nil {
		return "", err
	} else {
		pageBytes, err := io.ReadAll(pageResp.Body)
		defer pageResp.Body.Close()
		if err != nil {
			return "", err
		} else {
			pageJson := string(pageBytes)
			numDecksValue := gjson.Get(pageJson, "pageProps.data.num_decks").String()
			if numDecksValue == "" { // if num_decks does not exist
				deckJson := gjson.Get(pageJson, "pageProps.data.deck").String()
				fmt.Println(deckJson)
				deckJson = strings.ReplaceAll(deckJson, "\",\"", "\n")
				if len(deckJson) > 4 {
					return deckJson[2 : len(deckJson)-2], nil
				} else {
					return "", errors.New("the deck inside the response was empty")
				}
			}
			_, err := strconv.Atoi(numDecksValue)
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

// BuildScryfallCommanderQuery
// Builds a Scryfall.com query using the selected constraints from the UI
// Params: An Array of strings containing all currently selected color checkboxes (and the "Exact" checkbox) from the UI
// Returns :  The complete scryfall api request with the query as a string
func BuildScryfallCommanderQuery(selectedColors []string) string {
	var query string = "https://api.scryfall.com/cards/random?q="
	query += url.QueryEscape("is:Commander (game:paper) legal:commander (type:creature OR type:planeswalker)") // we only query for commanders
	if len(selectedColors) == 0 || len(selectedColors) == 1 && selectedColors[0] == "Exact" {                  // if nothing is selected or only the exact box is selected we dont add colors to the query
		return query
	} else { // if colors are selected
		var colors string = "<="           // assume non-exact matches
		for _, c := range selectedColors { // add each selected color to the query
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

// ParseScryfallData
// Parses the JSON data retrieved from a Scryfall card query and returns the pre-formatted cardname for EDHRec queries and the URI for a normal sized image of the card
// Params: The json_data from the Scryfall API response as a string
// Return: A tuple of strings containing the formatted card name and the image URI
func ParseScryfallData(jsonData string) (string, string) {
	// get the cards name
	var cardName string = gjson.Get(jsonData, "name").String()
	// format the card name to EDHREC URL format
	fmt.Println("Retrieved Commander: " + cardName)
	var replacer strings.Replacer = *strings.NewReplacer(
		" ", "-",
		",", "",
		"'", "",
		"&", "")

	var formattedCardName string = strings.ToLower(replacer.Replace(cardName))
	transformer := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, err := transform.String(transformer, formattedCardName)
	fmt.Println(res)
	if err == nil {
		formattedCardName = res
	}
	firstCardName, _, found := strings.Cut(formattedCardName, "-//")
	imageUri := gjson.Get(jsonData, "image_uris.normal").String()
	if found { // if the card is double faced we get only the first card image
		formattedCardName = firstCardName
		if imageUri == "" {
			imageUri = gjson.Get(jsonData, "card_faces.0.image_uris.normal").String()

		}
	}

	return formattedCardName, imageUri
}

// GetScryfallCommanderData
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

// GetImageResource
// Transforms an image URI into a fyne Resource for usage inside the UI
// Params: the image URI as a string
// Returns: a fyne.Resource representing the Image
func GetImageResource(imageUri string) fyne.Resource {
	r, err := fyne.LoadResourceFromURLString(imageUri) // load image
	if err != nil {                                    // if image can not be loaded attempt to load a placeholder
		r, err = fyne.LoadResourceFromURLString(PlaceholderImage)
		if err != nil { // if placeholder cant be loaded either, end the execution with error TODO: Download placeholder image, have it inside the program, do not retrieve it from online to secure failure case
			panic(err)
		}
		return r
	}
	return r
}

// GetBuildId
// Creates a valid buildId needed for EDHRec queries
// Param: None
// Return: A valid buildId as a String
func GetBuildId() string {
	response, err := http.Get(EdhrecBaseUrl)
	if err != nil {
		return "7-TtnLfoAX_AgebfCokAf"
	} else {
		body, err := io.ReadAll(response.Body) // read the data to a byte array
		defer response.Body.Close()            // close the Reader once this function returns
		if err != nil {                        // if we cant read the response, return the error
			return "7-TtnLfoAX_AgebfCokAf"
		} else { // if we read the response successfully
			scriptBlockRegex := regexp.MustCompile("<script id=\"__NEXT_DATA__\" type=\"application/json\">(.*)</script>")
			match := scriptBlockRegex.Find(body)
			propsData := string(match)
			id := gjson.Get(propsData, "buildId").String()
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
	w := myApp.NewWindow("Command Tower")

	// init checkBoxes´ß
	colors := []string{"White", "Black", "Blue", "Red", "Green", "Exact"}
	checkGroup := widget.NewCheckGroup(colors, nil)
	checkBoxes := container.NewHBox(container.NewCenter(checkGroup))

	// pull any first commander image
	imageUri, deckList := GetCommanderImageAndDecklist(checkGroup.Selected) // get any first commander (nothing selected)
	// load deck into clipboard
	clipboard.Write(clipboard.FmtText, []byte(deckList))
	resource := GetImageResource(imageUri)
	img := canvas.NewImageFromResource(resource)
	img.FillMode = canvas.ImageFillOriginal

	// Buttons
	newCommander := widget.NewButton("New Commander", func() {
		imageUri, deckList := GetCommanderImageAndDecklist(checkGroup.Selected)
		clipboard.Write(clipboard.FmtText, []byte(deckList))
		resource := GetImageResource(imageUri)
		img.Resource = resource
		img.Refresh()
	})

	vBox := container.NewVBox(img, checkBoxes, newCommander)
	w.SetContent(vBox)
	w.ShowAndRun()
}