package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/SimonBackx/lantern-crawler/queries"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type ApiController struct {
	url    string
	client *http.Client
}

func NewApiController() *ApiController {
	tr := &http.Transport{
		ResponseHeaderTimeout: 10 * time.Second,
	}

	client := &http.Client{
		Timeout:   40 * time.Second,
		Transport: tr,
	}

	return &ApiController{url: "http://localhost:8080/api", client: client}
}

func (a *ApiController) SaveResult(result *queries.Result) error {
	jsonString, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = a.newRequest("POST", "/result", bytes.NewReader(jsonString))
	return err
}

func (a *ApiController) GetQueries() ([]queries.Query, error) {
	body, err := a.newRequest("GET", "/queries", nil)
	if err != nil {
		return nil, err
	}

	var queries []queries.Query
	err = json.Unmarshal(body, &queries)
	if err != nil {
		return nil, err
	}
	return queries, nil
}

func (a *ApiController) newRequest(method, url string, reader io.Reader) ([]byte, error) {
	fmt.Printf("%s %s\n", method, url)

	key := "secret"
	user := "secret"

	if request, err := http.NewRequest(method, a.url+url, reader); err == nil {
		request.Header.Add("X-API-USER", user)
		request.Header.Add("X-API-KEY", key)

		if response, err := a.client.Do(request); err == nil {
			defer response.Body.Close()

			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return nil, err
			}

			fmt.Printf("Response %s %s %s\n", method, url, response.Status)

			// Yay! Response :D
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return body, nil
			}
			return body, fmt.Errorf("Request was not successfull (status %v)", response.StatusCode)

		} else {
			if response != nil && response.Body != nil {
				response.Body.Close()
			}
			return nil, err
		}
	} else {
		return nil, err
	}
}
