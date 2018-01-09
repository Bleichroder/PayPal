package paypal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"zxPayPal/beelog"
	"zxPayPal/config"
	"zxPayPal/paypal/model"
)

func GetUserInfo(resp http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		resp.Write([]byte("get test ok!"))
		resp.WriteHeader(http.StatusOK)
		return
	case "POST":
	default:
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.ParseForm()
	code := req.Form.Get("code")
	beelog.Log.Debug("code:%s", code)

	client := &http.Client{}
	newreq, err := http.NewRequest("GET", fmt.Sprintf("%s%s?schema=openid", config.Config.PayPalBaseUrl, "/v1/identity/openidconnect/userinfo/"), nil)
	if err != nil {
		beelog.Log.Error("Failed to new http request! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	mu.RLock()
	t, err := GetUserAccessToken(code)
	if err != nil {
		beelog.Log.Error("Failed to get userAccesstoken! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	userAccessToken.Token = t.Token
	userAccessToken.ExpireTime = time.Now().Unix() + t.ExpiresIn - 10

	newreq.Header.Set("Content-Type", "application/json")
	newreq.Header.Set("Authorization", "Bearer "+userAccessToken.Token)

	newresp, err := client.Do(newreq)
	if err != nil {
		beelog.Log.Error("send request failed! error: %s", err.Error())
		return
	}

	defer newresp.Body.Close()
	mu.RUnlock()
	newrespData, err := ioutil.ReadAll(newresp.Body)
	if err != nil {
		beelog.Log.Error("read response failed! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	beelog.Log.Debug("get userinfo resp.statusCode:%d", newresp.StatusCode)
	beelog.Log.Debug("get userinfo resp:%s", string(newrespData))
	if newresp.StatusCode != 200 {
		beelog.Log.Error("resp 200!! get userinfo response%s", string(newrespData))
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	uinfo := new(model.PayPalUserInfo)
	err = json.Unmarshal(newrespData, uinfo)
	if err != nil {
		beelog.Log.Error("unmarshal response failed! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	uinfobyte, _ := json.Marshal(uinfo)
	resp.WriteHeader(http.StatusOK)
	resp.Write(uinfobyte)
	return
}
