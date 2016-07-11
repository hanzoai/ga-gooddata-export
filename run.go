package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flosch/pongo2"
	"github.com/hanzo-io/oauthful"
	"github.com/skratchdot/open-golang/open"
)

var proxyHTML = pongo2.Must(pongo2.FromFile("templates/proxy.html"))
var redirectHTML = pongo2.Must(pongo2.FromFile("templates/redirect.html"))

type GAFlow struct {
}

func (f GAFlow) Decode(req *http.Request) (*oauthful.AuthorizationResponse, error) {
	errStr := req.FormValue("error")
	if errStr != "" {
		return nil, errors.New(errStr + "\n" + req.FormValue("error_reason") + "\n" + req.FormValue("error_description"))
	}

	expiresIn, _ := strconv.Atoi(req.FormValue("expires_in"))
	res := &oauthful.AuthorizationResponse{
		AccessTokenResponse: oauthful.AccessTokenResponse{
			AccessToken: req.FormValue("access_token"),
			TokenType:   req.FormValue("token_type"),
			ExpiresIn:   int64(expiresIn),
		},
		State: req.FormValue("state"),
	}

	return res, nil
}

func (f GAFlow) Verify(res *oauthful.AuthorizationResponse) error {
	if res.State != Config.State {
		return errors.New("Authorization Error, State Does Not Match")
	}
	return nil
}

func (f GAFlow) AddParams(vals *url.Values) error {
	return nil
}

func FileExists(dir string) bool {
	if _, err := os.Stat(dir); err == nil {
		return true
	}
	return false
}

func MkDir(dir string) {
	if !FileExists(dir) {
		os.MkdirAll(dir, os.ModePerm)
	}
}

func main() {
	toks, ok := getOAuthTokens()
	if !ok {
		fmt.Println("Getting new tokens.")
		toks, ok = newOAuthTokens()
		if !ok {
			fmt.Println("Could not get new tokens.")
			return
		}
	}

	csvExport(toks)
}

// Export all CSVs since last date (and update last one just in case)
func csvExport(toks *oauthful.AccessTokenResponse) {
	date := Date{Config.FirstDate}

	f, err := os.Open(Config.DataPath + "/date")
	if err == nil {
		err = Decode(f, &date)
		if err != nil {
			date.Date = Config.FirstDate
		}
	}

	MkDir(Config.ExportPath)
	now := time.Now()

	for now.After(date.Date) {
		fmt.Printf("Querying from Date %v\n", date.Date)

		ioutil.WriteFile(Config.DataPath+"/date", EncodeBytes(date), os.ModePerm)

		if err := writeFile(toks, Config.TestQuery, "test", date.Date); err != nil {
			return
		}

		date.Date = date.Date.Add(time.Hour * 24)
	}

	mergeFiles("test")
}

// Merge the CSV for a specific dataset
func mergeFiles(prefix string) error {
	exportPath := Config.ExportPath

	fileInfos, err := ioutil.ReadDir(exportPath + "/" + prefix)
	if err != nil {
		return err
	}

	fmt.Printf("Merging %v.csv\n", prefix)
	filename := exportPath + "/" + prefix + ".csv"

	_, _ = os.OpenFile(filename, os.O_CREATE, os.ModePerm)
	fh, err := os.OpenFile(filename, os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(fh)

	first := true
	for _, fileInfo := range fileInfos {
		fn := fileInfo.Name()

		if fn == ".DS_Store" {
			continue
		}

		f, _ := os.Open(exportPath + "/" + prefix + "/" + fn)
		defer f.Close()

		// fmt.Printf("Merge %v\n", fn)

		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanLines)

		if !first {
			scanner.Scan()
		} else {
			first = false
		}

		for scanner.Scan() {
			line := scanner.Text()
			// fmt.Printf("Line %v\n", line)
			fmt.Fprintln(w, line)
		}
	}
	w.Flush() // Don't forget to flush!

	return nil
}

// Write the CSV for a specific query for a specific date
func writeFile(toks *oauthful.AccessTokenResponse, q, prefix string, date time.Time) error {
	MkDir(Config.ExportPath + "/" + prefix)

	filename := Config.ExportPath + "/" + prefix + "/" + prefix + date.Format("_2006-01-02") + ".csv"

	fmt.Printf("Using Query for %v\n", prefix)

	records := GAResponse{}

	err := queryForDate(q, toks, date, &records)
	if err != nil {
		fmt.Printf("Query Error for %v: %v\n", prefix, err)
		return err
	}

	headers := []string{}
	for _, header := range records.ColumnHeaders {
		headers = append(headers, header.Name)
	}

	csv := "\"" + strings.Join(headers, "\",\"") + "\",\"ga:date\"\n"
	dateStr := date.Format("2006-01-02")

	for _, row := range records.Rows {
		csv += "\"" + strings.Join(row, "\",\"") + "\",\"" + dateStr + "\"\n"
	}

	ioutil.WriteFile(filename, []byte(csv), os.ModePerm)

	return nil
}

// Query GA API for a specific date
func queryForDate(q string, toks *oauthful.AccessTokenResponse, date time.Time, records *GAResponse) error {
	since := date.Add(time.Hour * -24)
	until := date

	url := Config.ReportingUrl + fmt.Sprintf(q, Config.AppId, toks.AccessToken, since.Format("2006-01-02"), until.Format("2006-01-02"))

	// Start querying data out
	response := GAResponse{NextLink: url}

	for response.NextLink != "" {
		nextUrl := response.NextLink
		fmt.Printf("Url: %v\n", nextUrl)

		res, err := http.Get(nextUrl)
		if err != nil {
			return err
		}

		response = GAResponse{}

		err = Decode(res.Body, &response)
		if err != nil {
			return err
		}

		// 	// Retry if out of api, otherwise its super annoying
		// 	if wrapper.Error.Message != "" {
		// 		if strings.Contains(wrapper.Error.Message, "#17") {
		// 			wrapper.Paging.Next = nextUrl
		// 			fmt.Println("Out of API, waiting 10 minutes before retrying...")
		// 			time.Sleep(time.Second * 600)
		// 			continue
		// 		}
		// 		return errors.New(wrapper.Error.Message)
		// 	}

		for _, row := range response.Rows {
			records.Rows = append(records.Rows, row)
		}

		if response.NextLink != "" {
			fmt.Println("Loading Next Page")
			response.NextLink += "&access_token=" + toks.AccessToken
		} else {
			fmt.Println("All Pages Loaded")
		}
	}

	records.ColumnHeaders = response.ColumnHeaders

	return nil
}

// Flatten the JSON so it doesn't screw up the Good Data Parser
func flattenJson(str string) string {
	return strings.Replace(strings.Replace(strings.Replace(str, "\"", "", -1), " ", "", -1), "\n", "", -1)
}

// Load OAuth Tokens from File if possible
func getOAuthTokens() (*oauthful.AccessTokenResponse, bool) {
	f, err := os.Open(Config.TokensPath)
	if err != nil {
		return nil, false
	}

	fmt.Println("Loading Tokens")
	toks := &oauthful.AccessTokenResponse{}

	err = Decode(f, toks)
	if err != nil {
		return nil, false
	}

	fmt.Println("Decoding Tokens")
	return toks, true
}

// Issue an OAuth2.0 Authorization Request
func newOAuthTokens() (*oauthful.AccessTokenResponse, bool) {
	go server()

	qs := []string{
		"client_id=" + Config.ClientId,
		"state=" + Config.State,
		"scope=" + Config.Scope,
		"redirect_uri=" + Config.RedirectUri,
		"response_type=token",
	}

	url := Config.AuthorizeUrl + "?" + strings.Join(qs, "&")

	open.Run(url)

	time.Sleep(time.Second * 10)

	return getOAuthTokens()
}

func oauthProxyRedirectHandler(w http.ResponseWriter, r *http.Request) {
	pctx := pongo2.Context{}

	err := proxyHTML.ExecuteWriter(pctx, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// OAuth2.0 Redirect Handler
func oauthRedirectHandler(w http.ResponseWriter, r *http.Request) {
	hc := &http.Client{}
	client := oauthful.New(hc, Config.TokenUrl, GAFlow{})

	pctx := pongo2.Context{
		"success": true,
	}

	if res, err := client.Handle(r); err != nil {
		pctx["success"] = false
		pctx["error"] = err.Error()
	} else {
		pctx["response"] = res
	}

	err := redirectHTML.ExecuteWriter(pctx, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if pctx["success"] == false {
		return
	}

	MkDir(Config.DataPath)
	ioutil.WriteFile(Config.TokensPath, EncodeBytes(pctx["response"]), os.ModePerm)
}

// Run the server that waits to execute the OAuth2.0 Redirect Handler
func server() {
	http.HandleFunc("/redirect", oauthProxyRedirectHandler)
	http.HandleFunc("/realredirect", oauthRedirectHandler)
	http.ListenAndServe(":8080", nil)
}
