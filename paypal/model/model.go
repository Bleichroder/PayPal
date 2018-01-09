package model

type PayPalSyncReq struct {
	Client       PayPalClient    `json:"client"`
	Response     PayPalExtraResp `json:"response"`
	ResponseType string          `json:"response_type"`
}

type PayPalClient struct {
	Environment      string `json:"environment"`
	PayPalSDKVersion string `json:"paypal_sdk_version"`
	Platform         string `json:"platform"`
	ProductName      string `json:"product_name"`
}

type PayPalExtraResp struct {
	CreateTime string `json:"create_time"`
	Id         string `json:"id"`
	Intent     string `json:"intent"`
	State      string `json:"state"`
}

type PayPalSyncResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type PayPalVerifyResp struct {
	Id    string `json:"id"`
	State string `json:"state"`
	Payer struct {
		Status string `json:"status"`
	} `json:"payer"`
	Transactions []struct {
		Custom           string `json:"custom"`
		Description      string `json:"description"`
		RelatedResources []struct {
			Sale struct {
				Id     string `json:"id"`
				State  string `json:"state"`
				Amount struct {
					Total    string `json:"total"`
					Currency string `json:"currency"`
				} `json:"amount"`
				TransactionFee struct {
					Value    string `json:"value"`
					Currency string `json:"currency"`
				} `json:"transaction_fee"`
				CreateTime string `json:"create_time"`
			} `json:"sale"`
		} `json:"related_resources"`
	} `json:"transactions"`
}

type PayReq struct {
	OrdelId     int64  `json:"order_id"`     // Pay表id
	PayType     int    `json:"pay_type"`     // 支付type，paypal为300
	Fee         int    `json:"fee"`          // 支付金额
	ProductName string `json:"product_name"` // 商品名称
	ProductDesc string `json:"product_desc"` // 商品描述
}

type PayResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type PayPalList struct {
	Id             int64  // Pay表id
	OrderId        string // Paypal的Pay id
	Fee            int    // 金额（分）
	FeeCurrency    string // 金额币种
	TaxFee         int    // 手续费金额（分）
	TaxFeeCurrency string //手续费币种
	ProductName    string // 商品名称
	ProductDesc    string // 商品描述
	PayType        int    // 支付type，paypal为900
	Status         int    // 1--发起支付（按下购买按钮）  2--Paypal付款成功  3--香蕉充值成功  -1--交易失败
	State          string // 付款返回状态
	PaymentDate    string // 付款时间
	TransactionId  string // Paypal的transaction id
	Created        string
	Updated        string
}

type PayPalUserInfo struct {
	PayerId         string `json:"payer_id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	EmailVerified   bool   `json:"email_verified,string"`
	Phone           string `json:"phone_number"`
	VerifiedAccount bool   `json:"verified,string"`
}

type AppPayoutReq struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	Currency string `json:"currency"`
	Receiver string `json:"receiver"`
}

type AppPayoutResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type PayoutReq struct {
	SendBatchHeader struct {
		SenderBatchId string `json:"sender_batch_id"`
		EmailSubject  string `json:"email_subject"`
	} `json:"sender_batch_header"`
	Items []Item `json:"items"`
}

type Item struct {
	RecipientType string `json:"recipient_type"`
	Amount        struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	} `json:"amount"`
	Note         string `json:"note"`
	SenderItemId string `json:"sender_item_id"`
	Receiver     string `json:"receiver"`
}

type PayoutResp struct {
	BatchHeader struct {
		SenderBatchHeader struct {
			SenderBatchId string `json:"sender_batch_id"`
			EmailSubject  string `json:"email_subject"`
		} `json:"sender_batch_header"`
		PayoutBatchId string `json:"payout_batch_id"`
		BatchStatus   string `json:"batch_status"`
	} `json:"batch_header"`
	Items []Item `json:"items"`
}
