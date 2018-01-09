CREATE TABLE `wx_pay` (
  `id` varchar(128) NOT NULL,
  `fee` int(11) NOT NULL,
  `product_desc` varchar(128) DEFAULT NULL,
  `pay_type` int(11) NOT NULL,
  `time_start` varchar(128) NOT NULL,
  `app_id` varchar(128) NOT NULL,
  `app_order_id` varchar(128) NOT NULL,
  `app_name` varchar(128) NOT NULL,
  `mch_key` varchar(128) NOT NULL,
  `message` varchar(128) NOT NULL,
  `transaction_id` varchar(128) DEFAULT NULL,
  `time_end` varchar(128) NOT NULL,
  `status` tinyint(4) NOT NULL,
  `success` tinyint(4) NOT NULL,
  `created` varchar(128) NOT NULL,
  `updated` varchar(128) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='微信支付记录表'

CREATE TABLE `ali_pay` (
  `id` varchar(128) NOT NULL,
  `service_app_name` varchar(128) NOT NULL,
  `service_order_id` varchar(128) NOT NULL,
  `fee` int(11) NOT NULL,
  `product_name` varchar(128) DEFAULT NULL,
  `product_desc` varchar(128) DEFAULT NULL,
  `pay_type` int(11) NOT NULL,
  `status` tinyint(4) NOT NULL,
  `app_id` varchar(128) NOT NULL,
  `trade_status` varchar(128) DEFAULT NULL,
  `trade_status_msg` varchar(128) DEFAULT NULL,
  `transaction_id` varchar(128) DEFAULT NULL,
  `created` varchar(128) NOT NULL,
  `updated` varchar(128) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='支付宝支付记录表'

CREATE TABLE `paypal_pay` (
    `id` bigint(20) NOT NULL,
    `order_id` varchar(128) DEFAULT '',
    `fee` int(11) NOT NULL,
    `fee_currency` varchar(10) DEFAULT '',
    `tax_fee` int(11) DEFAULT 0,
    `tax_fee_currency` varchar(10) DEFAULT '',
    `product_name` varchar(128) DEFAULT '',
    `product_desc` varchar(128) DEFAULT '',
    `pay_type` int(11) NOT NULL,
    `status` tinyint(4) NOT NULL,
    `state` varchar(20) DEFAULT '',
    `payment_date` varchar(40) DEFAULT '',
    `transaction_id` varchar(128) DEFAULT '',
    `created` varchar(128) NOT NULL,
    `updated` varchar(128) NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='PayPal支付记录表'