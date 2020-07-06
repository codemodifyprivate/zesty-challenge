package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	httpServer "github.com/codemodify/systemkit-appserver-http"
	crashproof "github.com/codemodify/systemkit-crashproof"
)

var data = map[string]int{}
var dataLoadError error

func main() {
	const listenOn = ":9000"

	crashproof.ConcurrentCodeCrashCatcher = reportCrash
	crashproof.RunAppAndCatchCrashes(func() {

		// 1. lazy load the data
		crashproof.Go(loadData)

		// 2. wait for queries
		fmt.Println("running-on:", listenOn)
		httpServer.NewHTTPServer([]httpServer.HTTPHandler{
			{
				Route:   "/autocomplete",
				Verb:    "GET",
				Handler: autocompleteRequestHandler,
			},
		}).Run(listenOn, true)
	})
}

func reportCrash(err interface{}, packageName string, callStack []crashproof.StackFrame) {
	fmt.Fprintf(os.Stderr, "\n\nCRASH: %v\n\npackage %s\n\nstack: %v\n\n", err, packageName, callStack)
}

func autocompleteRequestHandler(rw http.ResponseWriter, r *http.Request) {
	termValue := r.URL.Query().Get("term")
	topNValue := r.URL.Query().Get("top")
	topN := 25

	// if top N was specified - use that
	if len(strings.TrimSpace(topNValue)) > 0 {
		parsedTopN, err := strconv.Atoi(topNValue)
		if err == nil {
			topN = parsedTopN
		}
	}

	// find all words starting with `term`
	possibleWords := KeyValArray{}
	for key, val := range data {
		if strings.Index(key, termValue) == 0 {
			possibleWords = append(possibleWords, KeyVal{key, val})
		}
	}

	// sort by value
	sort.Sort(sort.Reverse(possibleWords))

	topNToReturn := topN
	if len(possibleWords) < topN {
		topNToReturn = len(possibleWords)
	}

	sb := strings.Builder{}
	for i := 0; i < topNToReturn; i++ {
		sb.WriteString(fmt.Sprintf("%s\n", possibleWords[i].Key))
	}

	rw.Write([]byte(sb.String()))
}

func loadData() {
	dataAsBytes, err := ioutil.ReadFile("shakespeare-complete.txt")
	if err != nil {
		dataLoadError = err
		return
	}

	var isLetters = regexp.MustCompile(`^[a-zA-Z]+$`).MatchString

	lines := strings.Split(string(dataAsBytes), "\n")
	for _, line := range lines {
		words := strings.Split(line, " ")
		for _, word := range words {
			if isLetters(word) {
				if _, ok := data[word]; ok { // if key exists
					data[word]++
				} else {
					data[word] = 0
				}
			}
		}
	}
}

type KeyVal struct {
	Key string
	Val int
}
type KeyValArray []KeyVal

func (thisRef KeyValArray) Len() int {
	return len(thisRef)
}
func (thisRef KeyValArray) Less(i, j int) bool {
	return thisRef[i].Val < thisRef[j].Val
}
func (thisRef KeyValArray) Swap(i, j int) {
	tmp := thisRef[i]
	thisRef[j] = thisRef[j]
	thisRef[i] = tmp
}
