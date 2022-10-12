package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mmcdole/gofeed"
)

var (
	rssUrl              = os.Getenv("RSS_URL")
	webflowApiKey       = os.Getenv("WEBFLOW_API_KEY")
	webflowCollectionID = os.Getenv("WEBFLOW_COLLECTION_ID")

	webflowAPIEndpoint         = "https://api.webflow.com/"
	webflowAPICollectionList   = fmt.Sprintf("%s/collections/%s/items", webflowAPIEndpoint, webflowCollectionID)
	webflowAPICollectionCreate = fmt.Sprintf("%s/collections/%s/items?live=true", webflowAPIEndpoint, webflowCollectionID)

	webflowHeaders = map[string]string{
		"Accept-Version": "1.0.0",
		"Authorization":  fmt.Sprintf("Bearer %s", webflowApiKey),
		"content-type":   "application/json",
	}
)

func main() {
	if webflowApiKey == "" {
		panic(fmt.Errorf("missing env var WEBFLOW_API_KEY"))
	}
	if webflowCollectionID == "" {
		panic(fmt.Errorf("missing env var WEBFLOW_COLLECTION_ID"))
	}
	if rssUrl == "" {
		panic(fmt.Errorf("missing env var RSS_URL"))
	}

	items, err := getExistingItems()
	if err != nil {
		panic(err)
	}
	slugs := map[string]bool{}
	for _, item := range items {
		slugs[item["slug"].(string)] = true
	}

	rss := gofeed.NewParser()
	feed, err := rss.ParseURL(rssUrl)
	if err != nil {
		panic(err)
	}
	for _, ri := range feed.Items {
		slug := guidToSlug(ri.GUID)
		image := ""
		created := ri.PublishedParsed.Format("2006-01-02T15:04:05-0700") // This is the only format Webflow seems to expect
		for _, e := range ri.Enclosures {
			if e.Type == "image/jpeg" {
				image = e.URL
			}
		}
		if image == "" {
			fmt.Printf("[debug] skipping %s because it doesn't have an image\n", ri.Link)
			continue
		}
		if _, ok := slugs[slug]; ok {
			fmt.Printf("[debug] skipping %s because it is already published\n", ri.Link)
			continue
		}

		fmt.Printf("CREATING ITEM: %v %+v %v\n", ri.Link, ri.Title, slug)

		cid, err := createItem(map[string]interface{}{
			"link":              ri.Link,
			"name":              ri.Title,
			"slug":              slug,
			"description":       ri.Description,
			"preview-image-url": image,
			// Not documented, but passing a url in the iamge object will upload it
			"preview-image": map[string]interface{}{
				"url": image,
			},
			// AFAICT, it's easier to create our own date instead of overwriting the built in ones
			"time": created,
		})
		if err != nil {
			fmt.Printf("[error] Create Item Failed: %v\n", err)
		} else {
			fmt.Printf("   -> %s\n", cid)
		}
		time.Sleep(time.Second * 15) // Generally just stay under any rate limiting
	}
}

func getExistingItems() (items []map[string]interface{}, err error) {
	req, err := http.NewRequest("GET", webflowAPICollectionList, nil)
	if err != nil {
		return items, err
	}
	for k, v := range webflowHeaders {
		req.Header.Set(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return items, fmt.Errorf("invalid status_code: %d", res.StatusCode)
	}

	var resp struct {
		Items []map[string]interface{} `json:"items"`
	}
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return items, err
	}
	return resp.Items, nil
}

func createItem(fields map[string]interface{}) (string, error) {
	fields["_archived"] = false
	fields["_draft"] = false
	body, _ := json.Marshal(map[string]interface{}{
		"fields": fields,
	})

	req, err := http.NewRequest("POST", webflowAPICollectionCreate, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	for k, v := range webflowHeaders {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		b, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("invalid status_code: %d (Body: %v)", res.StatusCode, string(b))
	}

	var resp map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return "", err
	}
	return resp["_cid"].(string), nil

}

func guidToSlug(g string) string {
	s := sha1.New()
	s.Write([]byte(g))
	bs := s.Sum(nil)
	return fmt.Sprintf("%x", bs)
}
