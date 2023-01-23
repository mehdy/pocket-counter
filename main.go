package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path"
)

var (
	port        int
	consumerKey string
	configPath  string
)

type config struct {
	ConsumerKey string `json:"consumer_key"`
	AccessToken string `json:"access_token"`
}

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
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	defaultConfigPath := path.Join(u.HomeDir, ".config", "pocket", "config.json")

	flag.IntVar(&port, "port", 5000, "local port to listen on")
	flag.StringVar(&consumerKey, "consumer-key", "", "the consumer_key you get from getpocket.com for your app")
	flag.StringVar(&configPath, "config", defaultConfigPath, "path to config file")
}

func getCode(consumerKey, redirectURI string) (string, error) {
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

func getAccessToken(consumerKey, code string) (string, error) {
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

func getUnreadArticlesCount(cfg *config) (int, error) {
	buf := bytes.NewBuffer([]byte{})
	json.NewEncoder(buf).Encode(
		map[string]string{"consumer_key": cfg.ConsumerKey, "access_token": cfg.AccessToken},
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

func loadConfig() (*config, error) {
	flag.Parse()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Dir(configPath), 0755); err != nil {
			return nil, fmt.Errorf("Create config dir failed: %w", err)
		}
	}

	cfg := &config{}
	f, err := os.Open(configPath)
	if err == nil {
		if err := json.NewDecoder(f).Decode(cfg); err != nil {
			return nil, fmt.Errorf("Config decode failed: %w", err)
		}
	}

	if consumerKey != "" {
		cfg.ConsumerKey = consumerKey
	}

	if cfg.ConsumerKey == "" {
		return nil, fmt.Errorf("You must provide consumer-key. Visit https://getpocket.com/developer/apps/ to get one")
	}

	if port < 1024 {
		return nil, fmt.Errorf("port number must be larger than 1024")
	}

	return cfg, nil
}

func updateConfig(cfg *config) error {
	f, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Open config file failed: %w", err)
	}

	if err := json.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("Config encode failed: %w", err)
	}

	return nil
}

func showResult(cfg *config, ch chan struct{}) {
	<-ch

	count, err := getUnreadArticlesCount(cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Total number of unread articles: %d\n", count)
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	ch := make(chan struct{})
	if cfg.AccessToken == "" {
		go func() {
			redirectURI := fmt.Sprintf("http://localhost:%d/", port)

			fmt.Println("Getting code")
			code, err := getCode(cfg.ConsumerKey, redirectURI)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Printf("Open https://getpocket.com/auth/authorize?request_token=%s&redirect_uri=%s\n", code, redirectURI)

			server := http.Server{Addr: fmt.Sprintf(":%d", port)}

			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				fmt.Println("Getting access token")

				cfg.AccessToken, err = getAccessToken(cfg.ConsumerKey, code)
				if err != nil {
					fmt.Println(err)
					return
				}

				ch <- struct{}{}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintf(w, "<html><body>You may close the window</body></html>")
			})

			if err := server.ListenAndServe(); err != nil {
				fmt.Println(err)
			}
		}()
	} else {
		go func() {
			ch <- struct{}{}
		}()
	}

	showResult(cfg, ch)

	err = updateConfig(cfg)
	if err != nil {
		fmt.Println(err)
	}
}
