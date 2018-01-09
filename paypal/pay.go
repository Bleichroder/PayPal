package paypal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"zxPayPal/beelog"
	"zxPayPal/db"
	"zxPayPal/paypal/model"
)

func Pay(resp http.ResponseWriter, req *http.Request) {
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

	payreq := new(model.PayReq)
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, req.Body)
	if err != nil {
		beelog.Log.Error("copy req.body error:%s", err.Error())
		return
	}
	beelog.Log.Debug("buf:%s", buf.Bytes())
	err = json.Unmarshal(buf.Bytes(), payreq)
	if err != nil {
		beelog.Log.Error("unmarshal request failed! error: %s", err.Error())
		return
	}

	beelog.Log.Debug("Pay request:%+v", payreq)

	pay := new(model.PayPalList)
	pay.Id = payreq.OrdelId
	pay.Fee = payreq.Fee
	pay.PayType = payreq.PayType
	pay.ProductName = payreq.ProductName
	pay.ProductDesc = payreq.ProductDesc
	pay.Status = 1
	pay.Created = time.Now().Format("2006-01-02 15:04:05")
	pay.Updated = time.Now().Format("2006-01-02 15:04:05")

	payresp := new(model.PayResp)
	result, err := db.Exec("insert into paypal_pay (`id`, `fee`, `pay_type`, `product_name`, `product_desc`, `status`, `created`, `updated`) values (?, ?, ?, ?, ?, ?, ?, ?)", pay.Id, pay.Fee, pay.PayType, pay.ProductName, pay.ProductDesc, pay.Status, pay.Created, pay.Updated)
	if err != nil {
		beelog.Log.Error("database error:%s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		payresp.Code = -1
		payresp.Msg = fmt.Sprintf("database error:%s", err.Error())
		return
	}
	ra, err := result.RowsAffected()
	if err != nil {
		beelog.Log.Error("database error:%s", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		payresp.Code = -1
		payresp.Msg = fmt.Sprintf("database error:%s", err.Error())
		return
	}
	if ra == 0 {
		beelog.Log.Error("affected rows is 0")
		resp.WriteHeader(http.StatusInternalServerError)
		payresp.Code = -1
		payresp.Msg = fmt.Sprintf("affected rows is 0")
		return
	}

	payresp.Code = 1
	payresp.Msg = "insert paypal_pay success"
	presp, _ := json.Marshal(payresp)

	resp.WriteHeader(http.StatusOK)
	resp.Write(presp)
}
