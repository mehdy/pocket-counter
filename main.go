package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
)

var (
	port        int
	consumerKey string
)

type oauthRequestResponse struct {
	Code string `json:"code"`
}

type authorizeResponse struct {
	AccessToken string `json:"access_token"`
}

type getArticlesResponse struct {
	Complete int                    `json:"complete"`
	List     map[string]interface{} `json:"list"`
}

func init() {
	flag.IntVar(&port, "port", 5000, "local port to listen on")
	flag.StringVar(&consumerKey, "consumer-key", "", "the consumer_key you get from getpocket.com for your app")
}

func getCode(redirectURI string) (string, error) {
	buf := bytes.NewBuffer([]byte{})
	json.NewEncoder(buf).Encode(
		map[string]string{"consumer_key": consumerKey, "redirect_uri": redirectURI},
	)

	req, err := http.NewRequest("POST", "https://getpocket.com/v3/oauth/request", buf)
	if err != nil {
		return "", fmt.Errorf("Create new request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF8")
	req.Header.Set("X-Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unexpected response code: %d, X-Error: %s", resp.StatusCode, resp.Header.Get("X-Error"))
	}

	result := &oauthRequestResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Response decode failed: %w", err)
	}

	return result.Code, nil
}

func getAccessToken(code string) (string, error) {
	buf := bytes.NewBuffer([]byte{})
	json.NewEncoder(buf).Encode(
		map[string]string{"consumer_key": consumerKey, "code": code},
	)

	req, err := http.NewRequest("POST", "https://getpocket.com/v3/oauth/authorize", buf)
	if err != nil {
		return "", fmt.Errorf("Create new request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF8")
	req.Header.Set("X-Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unexpected response code: %d, X-Error: %s", resp.StatusCode, resp.Header.Get("X-Error"))
	}

	result := &authorizeResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Response decode failed: %w", err)
	}

	return result.AccessToken, nil
}

func getUnreadArticlesCount(accessToken string) (int, error) {
	buf := bytes.NewBuffer([]byte{})
	json.NewEncoder(buf).Encode(
		map[string]string{"consumer_key": consumerKey, "access_token": accessToken},
	)

	req, err := http.NewRequest("POST", "https://getpocket.com/v3/get", buf)
	if err != nil {
		return 0, fmt.Errorf("Create new request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF8")
	req.Header.Set("X-Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("Request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Unexpected response code: %d, X-Error: %s", resp.StatusCode, resp.Header.Get("X-Error"))
	}

	result := &getArticlesResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("Response decode failed: %w", err)
	}

	return len(result.List), nil

}

func main() {
	flag.Parse()

	if consumerKey == "" {
		fmt.Println("You must provide consumer-key. Visit https://getpocket.com/developer/apps/ to get one")
		return
	}

	if port < 1024 {
		fmt.Println("port number must be larger than 1024")
		return
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/", port)

	fmt.Println("Getting code")
	code, err := getCode(redirectURI)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Open https://getpocket.com/auth/authorize?request_token=%s&redirect_uri=%s\n", code, redirectURI)

	server := http.Server{Addr: fmt.Sprintf(":%d", port)}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Getting access token")

		accessToken, err := getAccessToken(code)
		if err != nil {
			fmt.Println(err)
			return
		}

		count, err := getUnreadArticlesCount(accessToken)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Total number of unread articles: %d\n", count)

		server.Shutdown(context.Background())
	})

	server.ListenAndServe()
}
