

## MiningBenefit分账记录Mock

### 先找一个常用的测试币种

mysql> select id,coin_type_id,name from app_coins where app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and name like '%usdt%';

```mysql
+--------------------------------------+--------------------------------------+------------+
| id                                   | coin_type_id                         | name       |
+--------------------------------------+--------------------------------------+------------+
| 4c8fb477-58da-49f5-aa1c-5e72120754c2 | 9ba0bfee-3dcf-4905-a95e-852436af748f | tusdttrc20 |
| 78e493d0-e78c-4efe-b537-32569987ee81 | 9c81fd06-ce50-466b-84e1-568018f00c8c | usdttrc20  |
| 80b846de-f89a-4543-9a37-5448465bbbb0 | 9ba0bfee-3dcf-4905-a95e-852436af748f | tusdttrc20 |
| d936fc39-c3ba-4d0f-aced-88c04bd8fb2e | aaf04c25-2c87-46e7-99d3-56814e40ec61 | tusdterc20 |
+--------------------------------------+--------------------------------------+------------+
```

这里选择tusdttrc20

### 查找绑定该币种的商品

查找goods中绑定9ba0bfee-3dcf-4905-a95e-852436af748f(`tusdttrc20`)的商品

```mysql
mysql> select id,title from good_manager.goods where coin_type_id like '9ba0bfee-3dcf-4905-a95e-852436af748f';
+--------------------------------------+--------------------------------------+
| id                                   | title                                |
+--------------------------------------+--------------------------------------+
| 04644193-6845-4dec-8187-032b311e0f5f | benifittest1108                      |
| 092c0673-4462-42fb-96a0-df709e3ff266 | goodtest20230301                     |
| 124049c1-83ea-4db4-a475-89b4dde2a328 | 4bb55874-f3aa-46e3-9855-3fe22c1e9c98 |
| 18d23b05-768b-497c-b446-47bfdd31e2e5 | test aleo 20221214                   |
| 1980a791-dbb4-4f11-88bc-1b05d129eec4 | goodtest20230301                     |
| 1e92dca0-648c-4cb0-8280-d9ef1fdcfc34 | goodtest20221028                     |
| 22bef208-22c7-4dec-b947-3a2555cdb003 | aleo                                 |
| 4719649d-caa5-44d0-821e-a753b710c166 | 2c216130-6277-48df-a25a-4508d67df64e |
| 4b2abb2e-e8b8-4c2d-942d-760120e25171 | benifittest1108                      |
| 5882e86e-90c7-4686-a0e2-1185a4203c79 | benifittest1108                      |
| 58881c11-5eab-415b-aa9a-78348c904136 | notiftest                            |
| 6133921c-28fe-4cf4-8b7e-643a683e47b2 | goodtest20221028                     |
| 764f9b54-2ecf-4e90-b598-f9802a69b857 | goodtest20230808                     |
| 7def8916-3d6d-405c-84ff-71f361722cb3 | inspireclonetest                     |
| 803abb2f-46af-4402-9dd7-ad0fce2b9274 | goodtest20230224                     |
| 82d13a99-059c-4797-b2fb-b6559083b651 | goodtest20221028                     |
| 88c38aab-0b54-4202-b4b0-011148167c69 | inspiretest0810                      |
| 8a92acde-020b-4c43-ab7b-da765903789f | test  erc20  coin                    |
| 8fb56c76-3e42-4c1a-b420-7dbb13d68969 | Aleo - ohash                         |
| a67b6ac1-11af-42b9-b6a4-9883ddfd234e | goodtest20230301                     |
| a6d30110-c0f8-4e72-92ee-551b07e586b1 | goodtest20221028                     |
| a9b44296-6df4-4dec-a057-989b9c1a58a8 | goodtest11111                        |
| b4971b6c-7483-4215-9fef-5efe594cfd61 | goodtest11111                        |
| cee01ed2-4e32-4619-bb68-62cce55ed150 | benefittest20230529                  |
| de420061-e878-4a8b-986a-805cadd59233 | Aleo 测试算力                    |
| ec7b60ee-2cff-4f67-aadc-86619e1b7f03 | benifittest1108                      |
| f5e013cc-a7ba-47e8-acd3-de839a438b63 | goodtest20230301                     |
| f644970b-1c8b-4626-bf60-2ab8b1587c1c | goodtest20221028                     |
+--------------------------------------+--------------------------------------+

# 这里选择aleo
# 22bef208-22c7-4dec-b947-3a2555cdb003 | aleo   
```

### 查询AppGood

查找appgoods表中GoodID为22bef208-22c7-4dec-b947-3a2555cdb003的商品

```shell
mysql> select id,good_id,app_id,online,visible,good_name,price from good_manager.app_goods where good_id='22bef208-22c7-4dec-b947-3a2555cdb003' and app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769';
+--------------------------------------+--------------------------------------+--------------------------------------+--------+---------+-----------+------------------------+
| id                                   | good_id                              | app_id                               | online | visible | good_name | price                  |
+--------------------------------------+--------------------------------------+--------------------------------------+--------+---------+-----------+------------------------+
| 11aca2f3-c47f-438c-bfc1-764d7038c2c1 | 22bef208-22c7-4dec-b947-3a2555cdb003 | ff2c5d50-be56-413e-aba5-9c7ad888a769 |      1 |       1 | goodname  | 120.000000000000000000 |
+--------------------------------------+--------------------------------------+--------------------------------------+--------+---------+-----------+------------------------+
1 row in set (0.00 sec)
```

找个有钱人

```mysql
mysql> select user_id,spendable from ledger_manager.generals where coin_type_id='9ba0bfee-3dcf-4905-a95e-852436af748f' and app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and deleted_at=0;
+--------------------------------------+-----------------------------------+
| user_id                              | spendable                         |
+--------------------------------------+-----------------------------------+
| fba0bd90-99b2-44e1-88e8-5fdfad2dc9f0 |              0.000000000000000000 |
| 15cf1283-634a-4008-9913-c9a9235316a9 | 20000000000350.294000000000000000 |
| c48cf817-0b54-476f-9962-6379203a562a |              0.000000000000000000 |
| 628db1e7-2fd9-4468-a785-a434ba5849bc |          49995.000000000000000000 |
| b36df48a-3581-442b-b5ad-83ecf6effcdd |          99519.000000000000000000 |
| 06094f12-0c0c-43d9-ae0a-f34064ce1234 |            280.000000000000000000 |
| 297d7c7c-ea54-4843-8502-2c3b925f2749 |            107.888000000000000000 |
| 8c14fb2f-14f9-4656-84c3-e7ef104e9d58 |              5.000000000000000000 |
| 9bfb1441-7090-4cff-9451-7ada311bf736 |            223.126000000000000000 |
| f04837ad-1e19-4577-a54b-2425ba666620 |          49988.000000000000000000 |
| fec3494c-e7af-4e3f-b656-eb1174550c9e |              0.000000000000000000 |
| 8fe34f56-30dc-4507-a9dc-89b23e705794 |              2.325000000000000000 |
| d522efd6-7914-4ecd-afa7-a12d8cb4e59d |            184.000000000000000000 |
| 45dddb96-6f7d-410d-9c14-88744feedd30 |          49891.750000000000000000 |
| 06094f12-0c0c-43d9-ae0a-f34064ce8411 |              9.000000000000000000 |
| 1ecd7818-5ab0-442f-8a8b-209aa364a14b |          50000.000000000000000000 |
| 2a193c40-5a49-4212-88c3-82cc6c8793f4 |              0.000000000000000000 |
| 67e81b97-0eb3-4991-9857-7a7f49c9d296 |          49960.000000000000000000 |
| 7c9100a2-8e74-4c58-ade1-0c6efcef9b98 |              7.000000000000000000 |
| d0bbd699-65aa-4a98-9aed-d0753387f0e9 |              3.000000000000000000 |
| 720a7a27-0443-4f39-b960-824d23d19f69 |             12.000000000000000000 |
| 8358d943-00aa-49b1-acb9-e2d19ca4a5ae |          50034.000000000000000000 |
| 0f8f9873-41ae-4ee0-a1d9-522fff41e28d |             12.000000000000000000 |
| f4ba15ee-d76a-45d3-a0e5-d2d9a2c256bb |          50000.000000000000000000 |
| 090ce4b7-86c7-4faf-9d69-3a2e44e33555 |          49983.425000000000000000 |
| 06094f12-0c0c-43d9-ae0a-a34064ce84ca |              1.000000000000000000 |
| d214b1d3-1a42-4311-9ab6-3df0a9baa89e |             30.000000000000000000 |
| 8f65ff9b-9cf5-4e90-a49b-fd7c23caac3e |          39481.000000000000000000 |
| d75d746c-f586-4198-8e11-03a33cfbff1c |          49789.749999350000000000 |
| 6dfeafe9-b3d3-4f76-ad6b-6e4e1b66b111 |              3.000000000000000000 |
| 91a2da97-1ad7-4fe8-86fd-3fcba9636dfe |             15.000000000000000000 |
| 06094f12-0c0c-43d9-ae0a-f34064ce84c5 |              1.000000000000000000 |
| 9891b64d-1971-4f9e-b2e4-31b4fd5b8048 |           8520.000000000000000000 |
| b5b553af-616e-404d-9afc-aea039d29081 |              0.000000000000000000 |
| 06094f12-0c0c-43d9-ae0a-f34064ce84c1 |              7.000000000000000000 |
| 6c71e779-398a-4d3c-bb38-427f0f704bb3 |            615.000000000000000000 |
| f5a2c920-c757-4c4e-a865-e31fa89a8438 |             28.875000000000000000 |
| ec98ece0-e6b4-4cc9-96b8-7a072f9ed473 |            117.750000000000000000 |
| 06094f12-0c0c-43d9-ae0a-f34064ce2222 |             20.000000000000000000 |
| adbe61a0-b72d-42fb-b2f0-741d7bd068e5 |          49922.916666050000000000 |
| df2edeeb-fc6d-46c0-ae2a-8d86eaaaca9a |              0.000000000000000000 |
| 5c73d9ae-171d-427f-b10c-b358e4e70f95 |              7.800000000000000000 |
| 567edef8-8933-4bd2-a0ee-cf2edab995b1 |          49805.000000000000000000 |
| a4749ef8-5136-42e6-8570-0a2e047c98e6 |           1010.000000000000000000 |
| 1c3b1576-e6d3-46f7-bc1b-6f8bf6fd5645 |            400.000000000000000000 |
| 4775ff56-c3b9-4728-9ac6-b5df1ee9fbab |          99633.609000000000000000 |
| 2c3b1576-e6d3-46f7-bc1b-6f8bf6fd5645 |           1102.246466000000000000 |
| 073d39ba-7c75-4a26-b5ec-318492cb385b |          48913.375333000000000000 |
| 3700d2f4-a6d8-4a86-be5d-79ce15034abd |          50000.000000000000000000 |
| 2b258abf-a115-4d9c-be89-67285fef0270 |              1.500000000000000000 |
| ba9767ed-5fb2-473a-8ab0-34329b06d3c6 |          50000.000000000000000000 |
| 8ef814ae-320a-4485-b4f6-a01d07b44916 |              3.250000000000000000 |
| 4775ff56-c3b9-4728-9ac6-b5df1ee9fba1 |             20.000000000000000000 |
| 06094f12-0c0c-43d9-ae0a-f34064ce84ca |              3.000000000000000000 |
| d7ff8af6-d056-4377-87db-e9ee3d1885a1 |           1000.000000000000000000 |
```

#### 找到有钱人的email_address

```mysql
mysql> select id,email_address from appuser_manager.app_users where id in (select user_id from ledger_manager.generals where coin_type_id='9ba0bfee-3dcf-4905-a95e-852436af748f' and app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and deleted_at=0);

+--------------------------------------+---------------------------+
| id                                   | email_address             |
+--------------------------------------+---------------------------+
| fba0bd90-99b2-44e1-88e8-5fdfad2dc9f0 | tang20230810H@npool.com   |
| 15cf1283-634a-4008-9913-c9a9235316a9 |                           |
| c48cf817-0b54-476f-9962-6379203a562a | tang20230810D@npool.com   |
| 628db1e7-2fd9-4468-a785-a434ba5849bc | 11223345@npool.cc         |
| b36df48a-3581-442b-b5ad-83ecf6effcdd | chenmiao@npool.cc         |
| 297d7c7c-ea54-4843-8502-2c3b925f2749 | tanghong@npool.cc         |
| 8c14fb2f-14f9-4656-84c3-e7ef104e9d58 |                           |
| 9bfb1441-7090-4cff-9451-7ada311bf736 | tang20220927@npool.cc     |
| f04837ad-1e19-4577-a54b-2425ba666620 |                           |
| fec3494c-e7af-4e3f-b656-eb1174550c9e | tang20230810F@npool.com   |
| 8fe34f56-30dc-4507-a9dc-89b23e705794 | tang20230810B@npool.com   |
| d522efd6-7914-4ecd-afa7-a12d8cb4e59d | 1111111@npool.com         |
| 45dddb96-6f7d-410d-9c14-88744feedd30 | tang20221209@npool.cc     |
| 1ecd7818-5ab0-442f-8a8b-209aa364a14b | th13564039482@163.com     |
| 2a193c40-5a49-4212-88c3-82cc6c8793f4 | ttttang222226986@163.com  |
| 67e81b97-0eb3-4991-9857-7a7f49c9d296 | dqliprocyondev2@npool.com |
| 7c9100a2-8e74-4c58-ade1-0c6efcef9b98 | tanghong2222@npool.com    |
| d0bbd699-65aa-4a98-9aed-d0753387f0e9 | 452280221@qq.com          |
| 720a7a27-0443-4f39-b960-824d23d19f69 |                           |
| 8358d943-00aa-49b1-acb9-e2d19ca4a5ae | 531759321@qq.com          |
| 0f8f9873-41ae-4ee0-a1d9-522fff41e28d | tang20221211@npool.cc     |
| f4ba15ee-d76a-45d3-a0e5-d2d9a2c256bb |                           |
| 090ce4b7-86c7-4faf-9d69-3a2e44e33555 | 531759320@qq.com          |
| d214b1d3-1a42-4311-9ab6-3df0a9baa89e | tang20230811@npool.com    |
| 8f65ff9b-9cf5-4e90-a49b-fd7c23caac3e | tang20230605@npool.com    |
| d75d746c-f586-4198-8e11-03a33cfbff1c | tang20221213@npool.cc     |
| 6dfeafe9-b3d3-4f76-ad6b-6e4e1b66b111 | tang20220929@npool.cc     |
| 91a2da97-1ad7-4fe8-86fd-3fcba9636dfe | tang20230811D@npool.com   |
| 9891b64d-1971-4f9e-b2e4-31b4fd5b8048 | tang20230605b@npool.cc    |
| b5b553af-616e-404d-9afc-aea039d29081 | tang20230810I@npool.com   |
| 6c71e779-398a-4d3c-bb38-427f0f704bb3 | chenmiao2@npool.cc        |
| f5a2c920-c757-4c4e-a865-e31fa89a8438 | tang20230810A@npool.com   |
| ec98ece0-e6b4-4cc9-96b8-7a072f9ed473 | tang20221208@npool.cc     |
| adbe61a0-b72d-42fb-b2f0-741d7bd068e5 | tang20221214@npool.cc     |
| df2edeeb-fc6d-46c0-ae2a-8d86eaaaca9a | tang20230810G@npool.com   |
| 5c73d9ae-171d-427f-b10c-b358e4e70f95 | tang20230810C@npool.com   |
| 567edef8-8933-4bd2-a0ee-cf2edab995b1 | 111112@npool.com          |
| a4749ef8-5136-42e6-8570-0a2e047c98e6 | dqliprocyondev1@npool.com |
| 4775ff56-c3b9-4728-9ac6-b5df1ee9fbab | chenmiao@npool.com        |
| 2c3b1576-e6d3-46f7-bc1b-6f8bf6fd5645 | tang20230613@163.com      |
| 073d39ba-7c75-4a26-b5ec-318492cb385b | dqli161@163.com           |
| 3700d2f4-a6d8-4a86-be5d-79ce15034abd | 1234@npool.cc             |
| 2b258abf-a115-4d9c-be89-67285fef0270 | tang20230810E@npool.com   |
| ba9767ed-5fb2-473a-8ab0-34329b06d3c6 | 123@npool.cc              |
| 8ef814ae-320a-4485-b4f6-a01d07b44916 | 531759321@qq1.com         |
| 06094f12-0c0c-43d9-ae0a-f34064ce84ca | tanghong@npool.com        |
| d7ff8af6-d056-4377-87db-e9ee3d1885a1 | daiki0926@npool.cc        |
+--------------------------------------+---------------------------+
47 rows in set (0.01 sec)
```

### 创建订单：

- GoodValueUSD为商品价格*购买的份数*
- *GoodValue为商品价格*购买的份数*汇率

- PaymentCoinTypeID购买商品用的啥币种
- CoinTypeID商品的产出是啥币种

- LiveCoinUSDCurrency
- CoinUSDCurrency

```shell
grpcurl -d '{
    "Info": {
        "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
        "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6",
        "GoodID": "22bef208-22c7-4dec-b947-3a2555cdb003",
        "AppGoodID": "11aca2f3-c47f-438c-bfc1-764d7038c2c1",
        "PaymentCoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
        "InvestmentType": 10,
        "Units": "3",
        "GoodValue": "359.6796",
        "GoodValueUSD": "360",
        "DurationDays": 365,
        "OrderType": 10,
        "PaymentType": 10,
        "CoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
        "CoinUSDCurrency": "0.99911",
        "LiveCoinUSDCurrency": "0.99911",
        "StartAt": 1694016000,
        "EndAt": 1725552000
    }
}' --plaintext localhost:50441 order.middleware.order1.v1.Middleware.CreateOrder

# result
{
  "Info": {
    "ID": "0776467a-d25c-4606-97c4-364412d94d47",
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6",
    "GoodID": "22bef208-22c7-4dec-b947-3a2555cdb003",
    "AppGoodID": "11aca2f3-c47f-438c-bfc1-764d7038c2c1j",
    "PaymentID": "00000000-0000-0000-0000-000000000000",
    "ParentOrderID": "00000000-0000-0000-0000-000000000000",
    "Units": "3.000000000000000000",
    "GoodValue": "359.679600000000000000",
    "GoodValueUSD": "360.000000000000000000",
    "PaymentAmount": "0.000000000000000000",
    "DiscountAmount": "0.000000000000000000",
    "PromotionID": "00000000-0000-0000-0000-000000000000",
    "DurationDays": 365,
    "OrderTypeStr": "Normal",
    "OrderType": "Normal",
    "InvestmentTypeStr": "UnionMining",
    "InvestmentType": "UnionMining",
    "CouponIDsStr": "[]",
    "PaymentTypeStr": "PayWithBalanceOnly",
    "PaymentType": "PayWithBalanceOnly",
    "CoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
    "PaymentCoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
    "TransferAmount": "0.000000000000000000",
    "BalanceAmount": "0.000000000000000000",
    "CoinUSDCurrency": "0.999110000000000000",
    "LocalCoinUSDCurrency": "0.000000000000000000",
    "LiveCoinUSDCurrency": "0.999110000000000000",
    "OrderStateStr": "OrderStateCreated",
    "OrderState": "OrderStateCreated",
    "StartModeStr": "OrderStartConfirmed",
    "StartMode": "OrderStartConfirmed",
    "StartAt": 1694016000,
    "EndAt": 1725552000,
    "BenefitStateStr": "BenefitWait",
    "BenefitState": "BenefitWait",
    "PaymentFinishAmount": "0.000000000000000000",
    "PaymentStateStr": "PaymentStateWait",
    "PaymentState": "PaymentStateWait",
    "CancelStateStr": "DefaultOrderState",
    "CreatedAt": 1693972088,
    "UpdatedAt": 1693972088
  }
}
```

订单创建完成后，将对应的订单状态(Order_States表)修改为OrderStatePaid即可

### 创建MiningBenefit的Statement记录

```shell
grpcurl -d '{
    "Info":{
        "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
        "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6",
        "CoinTypeID":"9ba0bfee-3dcf-4905-a95e-852436af748f",
        "IOType":"Incoming",
        "IOSubType":"MiningBenefit",
        "Amount": "25",
        "IOExtra":"{\"GoodID\": \"11aca2f3-c47f-438c-bfc1-764d7038c2c1j\", \"OrderID\": \"a87772e5-2c4f-4460-98cb-dfb3ff728423\"}"
    }
}'  --plaintext localhost:50421 ledger.middleware.ledger.statement.v2.Middleware.CreateStatement

grpcurl -d '{
    "Info":{
        "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
        "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6",
        "CoinTypeID":"9ba0bfee-3dcf-4905-a95e-852436af748f",
        "IOType":"Incoming",
        "IOSubType":"MiningBenefit",
        "Amount": "15",
        "IOExtra":"{\"GoodID\": \"11aca2f3-c47f-438c-bfc1-764d7038c2c1j\", \"OrderID\": \"0776467a-d25c-4606-97c4-364412d94d47\"}"
    }
}'  --plaintext localhost:50421 ledger.middleware.ledger.statement.v2.Middleware.CreateStatement

grpcurl -d '{
    "Info":{
        "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
        "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6",
        "CoinTypeID":"9ba0bfee-3dcf-4905-a95e-852436af748f",
        "IOType":"Incoming",
        "IOSubType":"MiningBenefit",
        "Amount": "15",
        "IOExtra":"{\"GoodID\": \"11aca2f3-c47f-438c-bfc1-764d7038c2c1j\", \"OrderID\": \"aa37127c-741a-4a3b-a06f-37169a35f2c0\"}"
    }
}'  --plaintext localhost:50421 ledger.middleware.ledger.statement.v2.Middleware.CreateStatement
```

### 查询

```shell
grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6"
}' --plaintext localhost:50411 ledger.gateway.ledger.profit.v1.Gateway.GetProfits

grpcurl -d '{
   "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6"
}' --plaintext localhost:50411 ledger.gateway.ledger.profit.v1.Gateway.GetMiningRewards

grpcurl -d '{
   "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6"
}' --plaintext localhost:50411 ledger.gateway.ledger.profit.v1.Gateway.GetIntervalProfits

grpcurl -d '{
   "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6"
}' --plaintext localhost:50411 ledger.gateway.ledger.profit.v1.Gateway.GetGoodProfits

```



## 创建提现

```shell
#生成验证码
grpcurl -d '{
    "Prefix": "PrefixUserCode",
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "Account": "lidaqiangdev@npool.com",
    "AccountType": 20,
    "UsedFor": 60
}' --plaintext basal-middleware:50631 basal.middleware.usercode.v1.Middleware.CreateUserCode

#创建提现
grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "732d2a46-3c71-448f-8f92-067ff11634e1",
    "CoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
    "AccountID": "137b2b27-b511-4a43-bc9d-fe2263a05549",
    "Amount": "10",
    "Account": "lidaqiangdev@npool.com",
    "AccountType": 20,
    "VerificationCode": "577876"
}' --plaintext localhost:50411 ledger.gateway.withdraw.v1.Gateway.CreateWithdraw
```

### Get方法测试

```shell
grpcurl -d '{
    "TargetAppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "Offset": 0,
    "Limit": 10
}' --plaintext localhost:50411 ledger.gateway.ledger.statement.v1.Gateway.GetAppStatements
# select * from details where app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and deleted_at=0;

grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "2b258abf-a115-4d9c-be89-67285fef0270",
    "Offset": 0,
    "Limit": 1
}' --plaintext localhost:50411 ledger.gateway.ledger.statement.v1.Gateway.GetStatements
# select * from details where app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and user_id='2b258abf-a115-4d9c-be89-67285fef0270' and deleted_at=0;

grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "2b258abf-a115-4d9c-be89-67285fef0270"
}' --plaintext localhost:50411 ledger.gateway.ledger.v1.Gateway.GetLedgers
# select * from generals where coin_type_id='9ba0bfee-3dcf-4905-a95e-852436af748f' and app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and user_id='2b258abf-a115-4d9c-be89-67285fef0270' and deleted_at=0;

grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6",
}' --plaintext localhost:50411 ledger.gateway.ledger.profit.v1.Gateway.GetProfits
#  select * from profits where app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and user_id='2b258abf-a115-4d9c-be89-67285fef0270' and deleted_at=0;

grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "2b258abf-a115-4d9c-be89-67285fef0270"
}' --plaintext localhost:50411 ledger.gateway.ledger.profit.v1.Gateway.GetMiningRewards

grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "2b258abf-a115-4d9c-be89-67285fef0270"
}' --plaintext localhost:50411 ledger.gateway.ledger.profit.v1.Gateway.GetIntervalProfits

grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "d75d746c-f586-4198-8e11-03a33cfbff1c"
}' --plaintext localhost:50411  ledger.gateway.withdraw.v1.Gateway.GetWithdraws
# select * from withdraws where app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and user_id='d75d746c-f586-4198-8e11-03a33cfbff1c' and deleted_at=0;

grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769"
}' --plaintext localhost:50411  ledger.gateway.withdraw.v1.Gateway.GetAppWithdraws
# select * from withdraws where app_id='ff2c5d50-be56-413e-aba5-9c7ad888a769' and deleted_at=0;

grpcurl -d '{
    "AppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "UserID": "d75d746c-f586-4198-8e11-03a33cfbff1c",
    "TargetAppID": "ff2c5d50-be56-413e-aba5-9c7ad888a769",
    "TargetUserID": "a4749ef8-5136-42e6-8570-0a2e047c98e6",
    "CoinTypeID": "9ba0bfee-3dcf-4905-a95e-852436af748f",
    "Amount": "10000"
}' --plaintext localhost:50411 ledger.gateway.ledger.deposit.v1.Gateway.CreateAppUserDeposit

grpcurl -d '{
    "AppID": "ab4d1208-7da9-11ec-a6ea-fb41bda845cd",
    "UserID": "1a0c2966-8cbf-4fd3-a0ee-a2edfb6f99fb"
}' --plaintext localhost:50411 ledger.gateway.ledger.v1.Gateway.GetLedgers
```

