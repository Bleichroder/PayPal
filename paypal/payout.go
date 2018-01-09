package paypal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"zxPayPal/beelog"
	"zxPayPal/config"
	"zxPayPal/paypal/model"

	"github.com/garyburd/redigo/redis"
)

func Payout(resp http.ResponseWriter, req *http.Request) {
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
	b := new(bytes.Buffer)
	_, err := io.Copy(b, req.Body)
	if err != nil {
		beelog.Log.Error("copy request body err: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	payout := new(model.AppPayoutReq)
	err = json.Unmarshal(b.Bytes(), payout)
	if err != nil {
		beelog.Log.Error("unmarshal payout req err: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	preq := new(model.PayoutReq)
	preq.SendBatchHeader.SenderBatchId = payout.Id
	preq.SendBatchHeader.EmailSubject = "获得提现"
	preq.Items = make([]model.Item, 1)
	preq.Items[0].Amount.Value = payout.Value
	preq.Items[0].Amount.Currency = payout.Currency
	preq.Items[0].RecipientType = payout.Type
	preq.Items[0].SenderItemId = payout.Id + "001"
	preq.Items[0].Note = "感谢使用猩猩话题圈"
	preq.Items[0].Receiver = payout.Receiver
	preqdata, _ := json.Marshal(preq)
	newreq, err := http.NewRequest("POST", fmt.Sprintf("%s%s", config.Config.PayPalBaseUrl, "/v1/payments/payouts?sync_mode=false"), strings.NewReader(string(preqdata)))
	if err != nil {
		beelog.Log.Error("Failed to new http request! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	mu.RLock()
	c := redisPool.Get()
	defer c.Close()
	actoken, err := c.Do("GET", "accessToken")
	if err != nil {
		beelog.Log.Error("Failed to get accesstoken from redis! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	beelog.Log.Debug("redis_accessToken: %v, accessToken: %+v", actoken, accessToken)
	if actoken == nil {
		t, err := GetAccessToken()
		if err != nil {
			beelog.Log.Error("Failed to get accesstoken! error: %s", err.Error())
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		accessToken.Token = t.Token
		accessToken.ExpireTime = time.Now().Unix() + t.ExpiresIn - 10
		token, _ := json.Marshal(accessToken)
		c.Do("SET", "accessToken", string(token), "EX", t.ExpiresIn-10)
		mu.RUnlock()
	} else {
		rtoken, _ := redis.String(actoken, err)
		json.Unmarshal([]byte(rtoken), accessToken)
		if accessToken.ExpireTime < time.Now().Unix() {
			t, err := GetAccessToken()
			if err != nil {
				beelog.Log.Error("Failed to get accesstoken! error: %s", err.Error())
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}
			accessToken.Token = t.Token
			accessToken.ExpireTime = time.Now().Unix() + t.ExpiresIn - 10
			token, _ := json.Marshal(accessToken)
			c.Do("SET", "accessToken", string(token), "EX", t.ExpiresIn-10)
			mu.RUnlock()
		} else {
			mu.RUnlock()
		}
	}

	newreq.Header.Set("Content-Type", "application/json")
	newreq.Header.Set("Authorization", "Bearer "+accessToken.Token)

	newresp, err := client.Do(newreq)
	if err != nil {
		beelog.Log.Error("send request failed! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer newresp.Body.Close()
	newrespData, err := ioutil.ReadAll(newresp.Body)
	if err != nil {
		beelog.Log.Error("read response failed! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	beelog.Log.Debug("get userinfo resp.statusCode:%d", newresp.StatusCode)
	beelog.Log.Debug("get userinfo resp:%s", string(newrespData))
	if newresp.StatusCode != 201 {
		beelog.Log.Error("resp not 201!! payout code response%s", string(newrespData))
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	pResp := new(model.PayoutResp)
	err = json.Unmarshal(newrespData, pResp)
	if err != nil {
		beelog.Log.Error("unmarshal response failed! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	beelog.Log.Debug("payout resp:%+v", pResp)

	payoutResp := new(model.AppPayoutResp)
	beelog.Log.Debug("batch_status:%+v", pResp.BatchHeader.BatchStatus)
	if pResp.BatchHeader.BatchStatus == "SUCCESS" || pResp.BatchHeader.BatchStatus == "PENDING" || pResp.BatchHeader.BatchStatus == "NEW" {
		payoutResp.Code = 1
		payoutResp.Msg = pResp.BatchHeader.BatchStatus
	} else {
		payoutResp.Code = -1
		payoutResp.Msg = pResp.BatchHeader.BatchStatus
	}
	prespdata, _ := json.Marshal(payoutResp)
	beelog.Log.Debug("payoutResp:%+v", payoutResp)
	resp.WriteHeader(http.StatusOK)
	resp.Write(prespdata)
}
