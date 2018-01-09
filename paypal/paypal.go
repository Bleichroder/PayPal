package paypal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"zxPayPal/beelog"
	"zxPayPal/config"
	"zxPayPal/db"
	"zxPayPal/paynsq"
	"zxPayPal/paypal/model"

	"github.com/garyburd/redigo/redis"
)

// PayPal IPN回调
func PayPalIPN(resp http.ResponseWriter, req *http.Request) {
	beelog.Log.Debug("\nPayPal IPN")
	switch req.Method {
	case "GET":
		resp.Write([]byte("get test ok!"))
		resp.WriteHeader(http.StatusOK)
		return
	case "POST":
		resp.WriteHeader(http.StatusOK)
	default:
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, req.Body)
	if err != nil {
		beelog.Log.Error("copy req.body error:%s", err.Error())
		return
	}
	beelog.Log.Debug("buf.string():%s", buf.String())
	// 解析IPN传回的订单数据
	values, err := url.ParseQuery(buf.String())
	if err != nil {
		beelog.Log.Error("parsequery IPN resp failed! error:%s", err.Error())
		return
	}

	go reqRespPayPal(buf.String(), values)
}

func reqRespPayPal(r string, v url.Values) {
	client := &http.Client{}
	r = "cmd=_notify-validate&" + r

	// 讲IPN回传的数据加上cmd返回，验证合法性
	req, err := http.NewRequest("POST", config.Config.PayPalIPNUrl, strings.NewReader(r))
	if err != nil {
		beelog.Log.Error("Failed to new http request! error: %s", err.Error())
		return
	}

	req.Header.Set("User-Agent", "GO-IPN-VerificationScript")
	resp, err := client.Do(req)
	if err != nil {
		beelog.Log.Error("send request failed! error: %s", err.Error())
		return
	}

	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		beelog.Log.Error("read response failed! error: %s", err.Error())
		return
	}

	if resp.StatusCode != 200 {
		beelog.Log.Error("resp not 200!! IPN response%s", string(respData))
		return
	}

	beelog.Log.Debug("IPN resp:%s", string(respData))

	// IPN消息为和发消息
	if string(respData) == "VERIFIED" && strings.ToLower(v.Get("txn_type")) == "express_checkout" {
		paylist := new(model.PayPalList)
		err = db.QueryRow("select id, fee, fee_currency, tax_fee, tax_fee_currency, order_id, pay_type, payment_date, product_name, product_desc, state, status, transaction_id, created, updated from paypal_pay where id = ?", v.Get("custom")).Scan(&paylist.Id, &paylist.Fee, &paylist.FeeCurrency, &paylist.TaxFee, &paylist.TaxFeeCurrency, &paylist.OrderId, &paylist.PayType, &paylist.PaymentDate, &paylist.ProductName, &paylist.ProductDesc, &paylist.State, &paylist.Status, &paylist.TransactionId, &paylist.Created, &paylist.Updated)
		if err != nil {
			beelog.Log.Error("select from database failed! err:%s", err.Error())
		}
		if strings.ToLower(v.Get("payment_status")) == "completed" || strings.ToLower(v.Get("payment_status")) == "pending" {
			if paylist.State == "completed" && paylist.TransactionId == v.Get("txn_id") {
				// 数据库中已经存有本次订单
				if strings.ToLower(v.Get("payment_status")) == "completed" {
					beelog.Log.Debug("同步已经完成")
					// 把订单信息传入nsq
					paylist.State = strings.ToLower(v.Get("payment_status"))
					jsonBytes, _ := json.Marshal(paylist)
					beelog.Log.Debug("payTable:%+v", paylist)
					err = paynsq.Publish(fmt.Sprintf("update_banana"), jsonBytes)
					if err != nil {
						beelog.Log.Error("nsq publish send fail err:%s", err)
						return
					}

					// 更新数据库
					result, err := db.Exec("update paypal_pay set payment_date = ?, updated = ? where id = ?", v.Get("payment_date"), time.Now().Format("2006-01-02 15:04:05"), v.Get("custom"))
					if err != nil {
						beelog.Log.Error("database err:%s", err.Error())
						return
					}
					if ra, _ := result.RowsAffected(); ra == 0 {
						beelog.Log.Error("affected rows is 0")
					}
				}
				return
			} else if paylist.State == "pending" && paylist.TransactionId == v.Get("txn_id") {
				// 数据库中存在的是待处理的订单
				if strings.ToLower(v.Get("payment_status")) == "completed" {
					beelog.Log.Debug("pending已成功")
					// 把订单信息传入nsq
					paylist.State = strings.ToLower(v.Get("payment_status"))
					jsonBytes, _ := json.Marshal(paylist)
					beelog.Log.Debug("payTable:%+v", paylist)
					err = paynsq.Publish(fmt.Sprintf("update_banana"), jsonBytes)
					if err != nil {
						beelog.Log.Error("nsq publish send fail err:%s", err)
						return
					}

					// 更新数据库
					result, err := db.Exec("update paypal_pay set state = ?, payment_date = ?, updated = ? where id = ? and status != 3", strings.ToLower(v.Get("payment_status")), v.Get("payment_date"), time.Now().Format("2006-01-02 15:04:05"), v.Get("custom"))
					if err != nil {
						beelog.Log.Error("database err:%s", err.Error())
						return
					}
					if ra, _ := result.RowsAffected(); ra == 0 {
						beelog.Log.Error("affected rows is 0")
					}
				} else {
					beelog.Log.Debug("pending")
				}
				return
			} else if paylist.TransactionId == "" {
				// 数据库中不存在本次订单

				f, err := strconv.ParseFloat(v.Get("mc_gross"), 64)
				if err != nil {
					beelog.Log.Error("mc_gross is not float, err:%s", err.Error())
				}
				paylist.Fee = int(f * 100)
				paylist.FeeCurrency = v.Get("mc_currency")
				fe, err := strconv.ParseFloat(v.Get("mc_fee"), 64)
				if err != nil {
					beelog.Log.Error("mc_fee is not float, err:%s", err.Error())
				}
				paylist.TaxFee = int(fe * 100)
				paylist.TaxFeeCurrency = v.Get("mc_currency")
				paylist.TransactionId = v.Get("txn_id")
				paylist.State = strings.ToLower(v.Get("payment_status"))
				paylist.Status = 2
				paylist.PaymentDate = v.Get("payment_date")
				paylist.Updated = time.Now().Format("2006-01-02 15:04:05")
				result, err := db.Exec("update paypal_pay set fee = ?, fee_currency = ?, tax_fee = ?, tax_fee_currency = ?, transaction_id = ?, state = ?, status = ?, payment_date = ?, updated = ? where id = ? and status != 3", paylist.Fee, paylist.FeeCurrency, paylist.TaxFee, paylist.TaxFeeCurrency, paylist.TransactionId, paylist.State, paylist.Status, paylist.PaymentDate, paylist.Updated, v.Get("custom"))
				if err != nil {
					beelog.Log.Error("database err:%s", err.Error())
					return
				}
				if ra, _ := result.RowsAffected(); ra == 0 {
					beelog.Log.Error("affected rows is 0")
				}

				// 把订单信息传入nsq
				jsonBytes, _ := json.Marshal(paylist)
				beelog.Log.Debug("payTable:%+v", paylist)
				err = paynsq.Publish(fmt.Sprintf("update_banana"), jsonBytes)
				if err != nil {
					beelog.Log.Error("nsq publish send fail err:%s", err)
					return
				}
				return
			}
		} else {
			// IPN返回一次失败订单消息
			beelog.Log.Debug("paypal失败")
			if paylist.TransactionId == v.Get("txn_id") {
				// 把订单信息传入nsq
				paylist.Status = -1
				paylist.State = strings.ToLower(v.Get("payment_status"))
				jsonBytes, _ := json.Marshal(paylist)
				beelog.Log.Debug("payTable:%+v", paylist)
				err = paynsq.Publish(fmt.Sprintf("update_banana"), jsonBytes)
				if err != nil {
					beelog.Log.Error("nsq publish send fail err:%s", err)
					return
				}

				beelog.Log.Debug("primary state is %s", paylist.State)
				result, err := db.Exec("update paypal_pay set status = ?, state = ?, payment_date = ?, updated = ? where id = ? and status != 3", -1, strings.ToLower(v.Get("payment_status")), v.Get("payment_date"), time.Now().Format("2006-01-02 15:04:05"), v.Get("custom"))
				if err != nil {
					beelog.Log.Error("database err:%s", err.Error())
					return
				}
				if ra, _ := result.RowsAffected(); ra == 0 {
					beelog.Log.Error("affected rows is 0")
				}
			}
		}
	} else if string(respData) == "INVALID" {
		beelog.Log.Error("IPN Verify failed:INVALID")
	}
}

// PayPal App回调
func PayPalVerify(resp http.ResponseWriter, req *http.Request) {
	beelog.Log.Debug("\nPayPal Verify")
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
	err := req.ParseForm()
	if err != nil {
		beelog.Log.Error("parseform failed! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	// bbb, err := httputil.DumpRequest(req, true)
	// if err != nil {
	// 	beelog.Log.Error("DumpRequest:%s", err.Error())
	// }
	// beelog.Log.Debug("DumpRequest body:", string(bbb))
	// beelog.Log.Debug("content-type:", req.Header.Get("Content-Type"))
	r := req.Form.Get("resp")
	beelog.Log.Debug("resp:%s", r)
	paypalSyncReq := new(model.PayPalSyncReq)
	err = json.Unmarshal([]byte(r), paypalSyncReq)
	if err != nil {
		beelog.Log.Error("unmarshal response failed! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	newreq, err := http.NewRequest("GET", fmt.Sprintf("%s%s", config.Config.PayPalBaseUrl, "/v1/payments/payment/")+paypalSyncReq.Response.Id, nil)
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
		_, err = c.Do("SET", "accessToken", string(token), "EX", t.ExpiresIn-10)
		if err != nil {
			beelog.Log.Error("Failed to set accesstoken to redis! error: %s", err.Error())
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
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
			_, err = c.Do("SET", "accessToken", string(token), "EX", t.ExpiresIn-10)
			if err != nil {
				beelog.Log.Error("Failed to refresh accesstoken to redis! error: %s", err.Error())
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}
			mu.RUnlock()
		} else {
			mu.RUnlock()
		}
	}

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

	if newresp.StatusCode != 200 {
		beelog.Log.Error("resp not 200!! verify paypal response%s", string(newrespData))
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	beelog.Log.Debug("verify paypal resp:%s", string(newrespData))
	payVerifyResp := new(model.PayPalVerifyResp)
	err = json.Unmarshal(newrespData, payVerifyResp)
	if err != nil {
		beelog.Log.Error("unmarshal response failed! error: %s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	if payVerifyResp.Payer.Status == "VERIFIED" {
		if len(payVerifyResp.Transactions) > 0 {
			if len(payVerifyResp.Transactions[0].RelatedResources) > 0 {
				paylist := new(model.PayPalList)
				err = db.QueryRow("select id, fee, fee_currency, tax_fee, tax_fee_currency, order_id, pay_type, payment_date, product_name, product_desc, state, status, transaction_id, created, updated from paypal_pay where id = ?", payVerifyResp.Transactions[0].Custom).Scan(&paylist.Id, &paylist.Fee, &paylist.FeeCurrency, &paylist.TaxFee, &paylist.TaxFeeCurrency, &paylist.OrderId, &paylist.PayType, &paylist.PaymentDate, &paylist.ProductName, &paylist.ProductDesc, &paylist.State, &paylist.Status, &paylist.TransactionId, &paylist.Created, &paylist.Updated)
				if err != nil {
					beelog.Log.Error("select from database failed! err:%s", err.Error())
				}
				pSyncResp := new(model.PayPalSyncResp)

				if strings.ToLower(payVerifyResp.Transactions[0].RelatedResources[0].Sale.State) == "completed" || strings.ToLower(payVerifyResp.Transactions[0].RelatedResources[0].Sale.State) == "pending" {
					resp.WriteHeader(http.StatusOK)
					if paylist.State == "completed" && paylist.TransactionId == payVerifyResp.Transactions[0].RelatedResources[0].Sale.Id {
						beelog.Log.Debug("异步已经完成")
						// 把订单信息传入nsq
						jsonBytes, _ := json.Marshal(paylist)
						beelog.Log.Debug("payTable:%+v", paylist)
						err = paynsq.Publish(fmt.Sprintf("update_banana"), jsonBytes)
						if err != nil {
							beelog.Log.Error("nsq publish send fail err:%s", err)
							pSyncResp.Code = -1
							pSyncResp.Msg = "nsq publish fail"
							b, _ := json.Marshal(pSyncResp)
							resp.WriteHeader(http.StatusInternalServerError)
							resp.Write(b)
							return
						}

						//更新数据库
						result, err := db.Exec("update paypal_pay set updated = ?, order_id = ? where id = ?", time.Now().Format("2006-01-02 15:04:05"), payVerifyResp.Id, payVerifyResp.Transactions[0].Custom)
						if err != nil {
							beelog.Log.Error("database err:%s", err.Error())
							return
						}
						if ra, _ := result.RowsAffected(); ra == 0 {
							beelog.Log.Error("affected rows is 0")
						}
						pSyncResp.Code = 1
						pSyncResp.Msg = "PayPal支付成功"
						b, _ := json.Marshal(pSyncResp)
						resp.WriteHeader(http.StatusOK)
						resp.Write(b)
						return
					} else if paylist.State == "pending" && paylist.TransactionId == payVerifyResp.Transactions[0].RelatedResources[0].Sale.Id {
						if strings.ToLower(payVerifyResp.Transactions[0].RelatedResources[0].Sale.State) == "completed" {
							beelog.Log.Debug("pending已成功")
							// 把订单信息传入nsq
							paylist.State = payVerifyResp.Transactions[0].RelatedResources[0].Sale.State
							paylist.OrderId = payVerifyResp.Id
							jsonBytes, _ := json.Marshal(paylist)
							beelog.Log.Debug("payTable:%+v", paylist)
							err = paynsq.Publish(fmt.Sprintf("update_banana"), jsonBytes)
							if err != nil {
								beelog.Log.Error("nsq publish send fail err:%s", err)
								pSyncResp.Code = -1
								pSyncResp.Msg = "nsq publish fail, 无法更新香蕉"
								b, _ := json.Marshal(pSyncResp)
								resp.WriteHeader(http.StatusInternalServerError)
								resp.Write(b)
								return
							}

							// 更新数据库
							result, err := db.Exec("update paypal_pay set state = ?, updated = ?, order_id = ? where id = ? and status != 3", payVerifyResp.Transactions[0].RelatedResources[0].Sale.State, time.Now().Format("2006-01-02 15:04:05"), payVerifyResp.Id, payVerifyResp.Transactions[0].Custom)
							if err != nil {
								beelog.Log.Error("database err:%s", err.Error())
								return
							}
							if ra, _ := result.RowsAffected(); ra == 0 {
								beelog.Log.Error("affected rows is 0")
							}
							pSyncResp.Code = 1
							pSyncResp.Msg = "PayPal支付成功"
							b, err := json.Marshal(pSyncResp)
							resp.WriteHeader(http.StatusOK)
							resp.Write(b)
						} else {
							pSyncResp.Code = 2
							pSyncResp.Msg = "PayPal支付待处理"
							b, _ := json.Marshal(pSyncResp)
							resp.WriteHeader(http.StatusOK)
							resp.Write(b)
						}
						return
					} else if paylist.TransactionId == "" {
						// 更新数据库
						f, err := strconv.ParseFloat(payVerifyResp.Transactions[0].RelatedResources[0].Sale.Amount.Total, 64)
						if err != nil {
							beelog.Log.Error("mc_gross is not float, err:%s", err.Error())
						}
						paylist.Fee = int(f * 100)
						paylist.FeeCurrency = payVerifyResp.Transactions[0].RelatedResources[0].Sale.Amount.Currency
						fe, err := strconv.ParseFloat(payVerifyResp.Transactions[0].RelatedResources[0].Sale.TransactionFee.Value, 64)
						if err != nil {
							beelog.Log.Error("mc_fee is not float, err:%s", err.Error())
						}
						paylist.TaxFee = int(fe * 100)
						paylist.TaxFeeCurrency = payVerifyResp.Transactions[0].RelatedResources[0].Sale.TransactionFee.Currency
						paylist.TransactionId = payVerifyResp.Transactions[0].RelatedResources[0].Sale.Id
						paylist.State = payVerifyResp.Transactions[0].RelatedResources[0].Sale.State
						paylist.Status = 2
						paylist.PaymentDate = payVerifyResp.Transactions[0].RelatedResources[0].Sale.CreateTime
						paylist.OrderId = payVerifyResp.Id
						paylist.Updated = time.Now().Format("2006-01-02 15:04:05")
						result, err := db.Exec("update paypal_pay set fee = ?, fee_currency = ?, tax_fee = ?, tax_fee_currency = ?, transaction_id = ?, state = ?, status = ?, payment_date = ?, updated = ?, order_id = ? where id = ? and status != 3", paylist.Fee, paylist.FeeCurrency, paylist.TaxFee, paylist.TaxFeeCurrency, paylist.TransactionId, paylist.State, paylist.Status, paylist.PaymentDate, paylist.Updated, paylist.OrderId, payVerifyResp.Transactions[0].Custom)
						if err != nil {
							beelog.Log.Error("database err:%s", err.Error())
							return
						}
						if ra, _ := result.RowsAffected(); ra == 0 {
							beelog.Log.Error("affected rows is 0")
						}
						if strings.ToLower(payVerifyResp.Transactions[0].RelatedResources[0].Sale.State) == "completed" {
							pSyncResp.Code = 1
							pSyncResp.Msg = "PayPal支付成功"
							b, _ := json.Marshal(pSyncResp)
							resp.WriteHeader(http.StatusOK)
							resp.Write(b)
						} else {
							pSyncResp.Code = 2
							pSyncResp.Msg = "PayPal支付待处理"
							b, _ := json.Marshal(pSyncResp)
							resp.WriteHeader(http.StatusOK)
							resp.Write(b)
						}

						// 把订单信息传入nsq
						jsonBytes, _ := json.Marshal(paylist)
						beelog.Log.Debug("payTable:%+v", paylist)
						err = paynsq.Publish(fmt.Sprintf("update_banana"), jsonBytes)
						if err != nil {
							beelog.Log.Error("nsq publish send fail err:%s", err)
							pSyncResp.Code = -1
							pSyncResp.Msg = "nsq publish fail, 更新香蕉失败"
							b, _ := json.Marshal(pSyncResp)
							resp.WriteHeader(http.StatusInternalServerError)
							resp.Write(b)
							return
						}
						return
					}
				} else {
					pSyncResp.Code = -1
					pSyncResp.Msg = "PayPal支付失败"
					b, _ := json.Marshal(pSyncResp)
					resp.WriteHeader(http.StatusInternalServerError)
					resp.Write(b)
				}
			}
		}
	}
}
