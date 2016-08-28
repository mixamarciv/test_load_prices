package main

import (
	//"bytes"
	"fmt"
	"strconv"
	//"strings"
	//"text/template"
	//"time"

	"github.com/PuerkitoBio/goquery"
	xmlx "github.com/jteeuwen/go-pkg-xmlx"
	mf "github.com/mixamarciv/gofncstd3000"
	"github.com/parnurzeal/gorequest"

	s "strings"
)

var Fmts = fmt.Sprintf
var Itoa = strconv.Itoa

func main() {
	Initdb()

	p1 := make(chan int)
	p2 := make(chan int)
	p3 := make(chan int)

	load_from := 2800
	load_to := 1000 * 1000 * 1000

	go startload(p1, load_from)
	go startload(p2, load_from+1)
	go startload(p3, load_from+2)

	for i := load_from + 3; i < load_to; i++ {
		select {
		case <-p1:
			go startload(p1, i)
		case <-p2:
			go startload(p2, i)
		case <-p3:
			go startload(p3, i)
		}
	}

}

func startload(p chan<- int, iditem int) {
	loaditem2(iditem)
	p <- 1
}

func loaditem(id int) {
	sid := Itoa(id)
	//LogPrint("load " + sid)

	url := "http://api.eve-central.com/api/quicklook?typeid=" + sid
	req := gorequest.New()
	_, bodyresult, errs := req.Post(url).End()
	if len(errs) > 0 {
		LogPrint(Fmts("%#v", errs))
		LogPrintAndExit("request send error: \n url: " + url + "\n\n")
	}
	//Fmts("%#v", bodyresult)
	if s.Index(bodyresult, "<?xml ") != 0 { //если это не xml
		LogPrint("skip " + sid + ": not xml")
		return
	}

	doc := xmlx.New()
	err := doc.LoadString(bodyresult, nil)
	LogPrintErrAndExit("xmlx.LoadString error: \n"+bodyresult+"\n\n", err)
	node := doc.SelectNode("*", "quicklook")

	name := node.S("*", "itemname")

	node_s := doc.SelectNode("*", "sell_orders").SelectNodes("*", "order")
	node_b := doc.SelectNode("*", "buy_orders").SelectNodes("*", "order")

	//LogPrint("sell orders " + Itoa(len(node_s)) + "  buy orders " + Itoa(len(node_b)))

	if node_s == nil || len(node_s) == 0 || node_b == nil || len(node_b) == 0 {
		LogPrint("skip " + sid + ": " + name + ";  " + Itoa(len(node_s)) + " / " + Itoa(len(node_b)))
		return
	}

	query := `DELETE FROM itemtype WHERE id = ` + sid
	_, err = db.Exec(query)
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

	name = s.Replace(name, "'", "''", -1)
	query = `INSERT INTO itemtype(id,name) VALUES(` + sid + `,'` + name + `')`
	_, err = db.Exec(query)
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

	//загружаем ордера:
	query = `DELETE FROM sell_order WHERE id = ` + sid
	_, err = db.Exec(query)
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
	for _, n := range node_s {

		station := n.S("*", "station_name")
		station = s.Replace(station, "'", "''", -1)

		price := n.S("*", "price")
		cnt := n.S("*", "vol_remain")
		expires := n.S("*", "expires")

		query = `INSERT INTO sell_order(id,station,price,cnt,expires) 
		         VALUES(` + sid + `,'` + station + `',` + price + `*100,` + cnt + `,'` + expires + `')`
		_, err = db.Exec(query)
		LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
	}

	query = `DELETE FROM buy_order WHERE id = ` + sid
	_, err = db.Exec(query)
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
	for _, n := range node_s {

		station := n.S("*", "station_name")
		station = s.Replace(station, "'", "''", -1)

		price := n.S("*", "price")
		cnt := n.S("*", "vol_remain")
		expires := n.S("*", "expires")

		query = `INSERT INTO buy_order(id,station,price,cnt,expires) 
		         VALUES(` + sid + `,'` + station + `',` + price + `*100,` + cnt + `,'` + expires + `')`
		_, err = db.Exec(query)
		LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
	}

	query = `commit`
	_, err = db.Exec(query)
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

	LogPrint("load " + sid + ": " + name + ";  " + Itoa(len(node_s)) + " / " + Itoa(len(node_b)))
}

func Trim(str string) string {
	return s.Trim(str, " \n\r\t")
}

func loaditem2(id int) {
	sid := Itoa(id)
	//LogPrint("load " + sid)

	url := "https://eve-central.com/home/quicklook.html?typeid=" + sid
	doc, err := goquery.NewDocument(url)
	if err != nil {
		LogPrint(Fmts("%#v", err))
		LogPrintAndExit("request send error: \n url: " + url + "\n\n")
	}

	sel := doc.Find("h1")
	if len(sel.Nodes) == 0 {
		LogPrint("skip " + sid + ": not found h1")
		return
	}
	name := Trim(sel.Text())
	name = s.Replace(name, " - Market Browser", "", 1)

	sels := doc.Find("#sell_orders tr")
	if len(sels.Nodes) < 2 {
		LogPrint("skip " + sid + ": not found sell_orders")
		return
	}

	selb := doc.Find("#buy_orders tr")
	if len(selb.Nodes) < 2 {
		LogPrint("skip " + sid + ": not found buy_orders")
		return
	}
	//sel := doc.Find("h1")

	query := `DELETE FROM itemtype WHERE id = ` + sid
	_, err = db.Exec(query)
	//res.Close()
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

	name = s.Replace(name, "'", "''", -1)
	query = `INSERT INTO itemtype(id,name) VALUES(` + sid + `,'` + name + `')`
	_, err = db.Exec(query)
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

	//загружаем ордера:
	loadprices(sels, "sell_order", sid)
	loadprices(selb, "buy_order", sid)

	query = `commit`
	_, err = db.Exec(query)
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

	LogPrint("load " + sid + ": " + name + ";  " + Itoa(len(sels.Nodes)) + " / " + Itoa(len(selb.Nodes)))
}

func loadprices(sel *goquery.Selection, tablename string, sid string) {
	query := `DELETE FROM ` + tablename + ` WHERE id = ` + sid
	_, err := db.Exec(query)
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
	for i, _ := range sel.Nodes {
		t := sel.Eq(i).Find("td")
		if len(t.Nodes) < 6 {
			continue
		}

		station := Trim(t.Eq(0).Text()) + " > " + s.Replace(Trim(t.Eq(1).Text()), "[-]", "", 1)

		price := s.Replace(Trim(t.Eq(2).Text()), ",", "", -1)
		cnt := s.Replace(Trim(t.Eq(3).Text()), ",", "", -1)
		cnt = mf.StrRegexpReplace(cnt, "\\(Min: [\\d,]+\\)", "")
		expires := Trim(t.Eq(4).Text())

		query = `INSERT INTO ` + tablename + `(id,station,price,cnt,expires) 
		         VALUES(` + sid + `,'` + station + `',` + price + `*100,` + cnt + `,'` + expires + `')`
		_, err = db.Exec(query)
		LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
	}
}

/************************
SELECT
  (a.sell_price1 * MINVALUE(a.sell_cnt1,a.buy_cnt2)) AS sell1,
  (a.buy_price2 * MINVALUE(a.sell_cnt1,a.buy_cnt2)) AS buy2,
  (a.buy_price2 * MINVALUE(a.sell_cnt1,a.buy_cnt2)) - (a.sell_price1 * MINVALUE(a.sell_cnt1,a.buy_cnt2)) AS prof,
  a.*
FROM
(
SELECT
  a.id,
  a.name,

  (SELECT FIRST 1 t.price/100 AS price FROM sell_order t
   WHERE t.id = a.id ORDER BY t.price ) AS sell_price1,
  (SELECT FIRST 1 t.cnt FROM sell_order t
   WHERE t.id = a.id ORDER BY t.price ) AS sell_cnt1,
  (SELECT FIRST 1 t.station FROM sell_order t
   WHERE t.id = a.id ORDER BY t.price ) AS sell_station1,

  (SELECT FIRST 1 t.price/100 AS price FROM buy_order t
   WHERE t.id = a.id ORDER BY t.price DESC) AS buy_price1,
  (SELECT FIRST 1 t.cnt FROM buy_order t
   WHERE t.id = a.id ORDER BY t.price DESC) AS buy_cnt1,
  (SELECT FIRST 1 t.station FROM buy_order t
   WHERE t.id = a.id ORDER BY t.price DESC) AS buy_station1,


  (SELECT FIRST 1 SKIP 1 t.price/100 AS price FROM sell_order t
   WHERE t.id = a.id ORDER BY t.price ) AS sell_price2,
  (SELECT FIRST 1 SKIP 1 t.cnt FROM sell_order t
   WHERE t.id = a.id ORDER BY t.price ) AS sell_cnt2,
  (SELECT FIRST 1 SKIP 1 t.station FROM sell_order t
   WHERE t.id = a.id ORDER BY t.price ) AS sell_station2,

  (SELECT FIRST 1 SKIP 1 t.price/100 AS price FROM buy_order t
   WHERE t.id = a.id ORDER BY t.price DESC) AS buy_price2,
  (SELECT FIRST 1 SKIP 1 t.cnt FROM buy_order t
   WHERE t.id = a.id ORDER BY t.price DESC) AS buy_cnt2,
  (SELECT FIRST 1 SKIP 1 t.station FROM buy_order t
   WHERE t.id = a.id ORDER BY t.price DESC) AS buy_station2,

  '-' AS tmp
FROM itemtype a
WHERE a.id > 0
) a
WHERE a.sell_price2 < a.buy_price2
  --AND a.sell_price2 * MINVALUE(a.sell_cnt2,a.buy_cnt2) < a.buy_price2 * MINVALUE(a.sell_cnt2,a.buy_cnt2)
  --AND a.sell_price2 * MINVALUE(a.sell_cnt2,a.buy_cnt2) < a.buy_price2 * MINVALUE(a.sell_cnt2,a.buy_cnt2) - 1000000*100
  AND a.sell_price1 * MINVALUE(a.sell_cnt1,a.buy_cnt1) < a.buy_price1 * MINVALUE(a.sell_cnt1,a.buy_cnt1) - 1*1000*1000
  AND a.sell_price1 * MINVALUE(a.sell_cnt1,a.buy_cnt2) < a.buy_price2 * MINVALUE(a.sell_cnt1,a.buy_cnt2) - 1*1000*1000


***********************************/
