package main

import (
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var PlaceholderImage string = "https://static.wikia.nocookie.net/mtgsalvation_gamepedia/images/f/f8/Magic_card_back.jpg/revision/latest?cb=20140813141013"
var EdhrecBaseUrl string = "https://edhrec.com"

// GetCommanderImageAndDecklist
// Selects a random commander depending on the input constraints and fetches an image and a decklist for said commander
// Params: An Array of strings containing all currently selected color checkboxes (and the "Exact" checkbox) from the UI
// Returns: A Tuple of strings, the first of which being the image URI of the commander and the second being the decklist for the commander
func GetCommanderImageAndDecklist(selectedColors []string, searchQuery string) (string, string, float64) {
	var query string = BuildScryfallCommanderQuery(selectedColors, searchQuery)
	fmt.Println("Retrieving Commander with Query: " + query)
	commanderData, err := GetScryfallCommanderData(query)
	if err != nil {
		return PlaceholderImage, "", 0.0 // return a placeholder image and no decklist
	} else {
		cardName, imageUri := ParseScryfallData(commanderData)
		fmt.Println(cardName + " : " + imageUri)
		deckList, err := GetEDHRecAvgDecklist(cardName)
		if err != nil {
			return PlaceholderImage, "", 0.0 // return a placeholder image and no decklist
		} else {
			price := GetScryfallPricingData(strings.Split(deckList, "\n"))
			return imageUri, deckList, price
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
func BuildScryfallCommanderQuery(selectedColors []string, searchQuery string) string {
	var query string = "https://api.scryfall.com/cards/random?q="
	query += url.QueryEscape("is:Commander (game:paper) legal:commander (type:creature OR type:planeswalker) " + searchQuery + " ") // we only query for commanders
	if len(selectedColors) == 0 || len(selectedColors) == 1 && selectedColors[0] == "Exact" {                                       // if nothing is selected or only the exact box is selected we dont add colors to the query
		return query
	} else { // if colors are selected
		var colors string = "<="           // assume non-exact matches
		for _, c := range selectedColors { // add each selected color to the query
			switch c {
			case "e": // if exact matches are wanted, remove the '<' from the colors string
				colors = colors[1:]
			default:
				colors += c
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
		"&", "",
		".", "")

	var formattedCardName string = strings.ToLower(replacer.Replace(cardName))
	transformer := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, err := transform.String(transformer, formattedCardName)
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

// GetScryfallPricingData
// Build a json object of card identifiers for a decklist and retrieve pricing information for the entire deck
// params: a decklist as an [] string with every entry being formatted as "1 <CardName>"
// returns the sum of prices of the cheapest, most recent printings of all cards in the request in euro
func GetScryfallPricingData(deck []string) float64 {
	sum := 0.0
	part1, part2 := deck[:len(deck)/2], deck[len(deck)/2:]
	for i := range 2 {
		// build json array
		jsonArray := "{\"identifiers\":["
		localDeck := part1
		if i != 0 {
			localDeck = part2
		}

		for _, card := range localDeck {
			name := strings.SplitN(card, " ", 2)[1]
			jsonArray += "{\"name\":\"" + name + "\"},"
		}

		jsonArray = jsonArray[:len(jsonArray)-1] + "]}"

		resp, err := http.Post("https://api.scryfall.com/cards/collection/", "application/json", strings.NewReader(jsonArray))
		if err != nil {
			return 0.0
		} else {
			body, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				return 0.0
			} else {
				data := string(body)
				prices := gjson.Get(data, "data.#.prices.eur")
				for _, p := range prices.Array() {
					sum += p.Float()
				}
			}
		}
	}
	fmt.Println("Price:" + strconv.FormatFloat(sum, 'f', 2, 64))
	return sum
}
