package paypal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
	"zxPayPal/beelog"
	"zxPayPal/config"

	"github.com/garyburd/redigo/redis"
)

// TokenResponse is for API response for the /oauth2/token endpoint
type TokenResponse struct {
	Token     string `json:"access_token"`
	Type      string `json:"token_type"`
	ExpiresIn int64  `json:"expires_in"`
}

type Token struct {
	Token        string
	RefreshToken string
	ExpireTime   int64
}

var (
	mu              sync.RWMutex
	accessToken     Token
	userAccessToken Token
)

func GetAccessToken() (*TokenResponse, error) {
	client := new(http.Client)
	buf := bytes.NewBuffer([]byte("grant_type=client_credentials"))
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", config.Config.PayPalBaseUrl, "/v1/oauth2/token"), buf)
	if err != nil {
		return &TokenResponse{}, err
	}

	req.SetBasicAuth(config.Config.ClientID, config.Config.Secret)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	t := new(TokenResponse)
	// Set default headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en_US")

	resp, err := client.Do(req)
	if err != nil {
		beelog.Log.Error("send request failed! error: %s", err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		beelog.Log.Error("read response failed! error: %s", err.Error())
		return nil, err
	}

	if resp.StatusCode != 200 {
		beelog.Log.Error("resp 200!! get access token response%s", string(respData))
		return nil, fmt.Errorf("get token resp code 200")
	}

	err = json.Unmarshal(respData, t)
	if err != nil {
		beelog.Log.Error("unmarshal response failed! error: %s", err.Error())
		return nil, err
	}

	beelog.Log.Debug("%s", string(respData))

	return t, err
}

func GetUserAccessToken(code string) (*TokenResponse, error) {
	client := new(http.Client)
	buf := bytes.NewBuffer([]byte("grant_type=authorization_code&response_type=token&redirect_uri=urn:ietf:wg:oauth:2.0:oob&code=" + code))
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", config.Config.PayPalBaseUrl, "/v1/oauth2/token"), buf)
	if err != nil {
		return &TokenResponse{}, err
	}

	req.SetBasicAuth(config.Config.ClientID, config.Config.Secret)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	t := new(TokenResponse)
	// Set default headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en_US")

	resp, err := client.Do(req)
	if err != nil {
		beelog.Log.Error("send request failed! error: %s", err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		beelog.Log.Error("read response failed! error: %s", err.Error())
		return nil, err
	}

	if resp.StatusCode != 200 {
		beelog.Log.Error("resp 200!! get access token response%s", string(respData))
		return nil, fmt.Errorf("get token resp code 200")
	}

	err = json.Unmarshal(respData, t)
	if err != nil {
		beelog.Log.Error("unmarshal response failed! error: %s", err.Error())
		return nil, err
	}

	beelog.Log.Debug("%s", string(respData))

	return t, err
}

var redisPool *redis.Pool

func RedisInit() {
	redisPool = GetRedisPool(config.Config.RedisHost, config.Config.RedisPassword, 10)
}

func GetRedisPool(server, password string, maxConn int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     100,
		MaxActive:   maxConn,
		Wait:        true,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}
