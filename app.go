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

	flags "github.com/jessevdk/go-flags"

	"errors"
	"os"
	s "strings"
)

var Fmts = fmt.Sprintf
var Print = fmt.Print
var Itoa = strconv.Itoa

func main() {
	Initdb()

	var opts struct {
		Load_from   int `long:"load_from" description:"start load id from"`
		Load_count  int `long:"load_count" description:"count id load"`
		Update_only int `long:"update_only" description:"update only" default:"0"`
	}
	_, err := flags.ParseArgs(&opts, os.Args)
	LogPrintErrAndExit("ошибка разбора параметров", err)

	p1 := make(chan int)
	p2 := make(chan int)
	p3 := make(chan int)
	go func() {
		p3 <- 0
		p2 <- 0
		p1 <- 0
	}()

	load_from := opts.Load_from                 //10
	load_to := opts.Load_from + opts.Load_count //1000 * 1000 * 1000

	if opts.Update_only == 0 {
		LogPrint("Загрузка новых записей в бд")
		loadtype := 0
		//go startload(p1, load_from)
		//go startload(p2, load_from+1)
		//go startload(p3, load_from+2)

		for i := load_from + 3; i < load_to; i++ {
			select {
			case <-p1:
				go startload(p1, i, loadtype)
			case <-p2:
				go startload(p2, i, loadtype)
			case <-p3:
				go startload(p3, i, loadtype)
			}
		}
	} else {
		LogPrint("Обновление существующих записей в бд")
		loadtype := 1
		query := `SELECT FIRST ` + Itoa(opts.Load_count) + ` SKIP ` + Itoa(opts.Load_from) +
			` id FROM itemtype ORDER BY id`
		rows, err := db.Query(query)
		LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
		var i int
		for rows.Next() {
			err = rows.Scan(&i)
			LogPrintErrAndExit("rows.Scan error: \n"+query+"\n\n", err)
			select {
			case <-p1:
				go startload(p1, i, loadtype)
			case <-p2:
				go startload(p2, i, loadtype)
			case <-p3:
				go startload(p3, i, loadtype)
			}
		}
	}

	<-p3
	<-p2
	<-p1
}

func startload(p chan<- int, iditem int, loadtype int) {
	loaditem3(iditem, loadtype)
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

func loaditem3(id int, loadtype int) {
	sid := Itoa(id)
	//LogPrint("load " + sid)

	//удаляем старые данные в отдельной горутине
	delete_old_item := make(chan bool, 1)
	go func() {
		if loadtype == 0 {
			query := `DELETE FROM itemtype WHERE id = ` + sid
			_, err := db.Exec(query)
			LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
		}

		query := `DELETE FROM buy_order WHERE id = ` + sid
		_, err := db.Exec(query)
		LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

		query = `DELETE FROM sell_order WHERE id = ` + sid
		_, err = db.Exec(query)
		LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

		delete_old_item <- true
	}()

	//загружаем название и сел ордера
	url := "http://eve-marketdata.com/price_check.php?region_id=-4&type=sell&type_id=" + sid
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

	skip := 0 //флаг для загрузки ордеров с другого ресурса
	sels := doc.Find(".price_check tr")
	sels_type := "eve-marketdata.com"
	if len(sels.Nodes) < 2 {
		//LogPrint("skip " + sid + ": not found sell_orders")
		skip++
	}

	//загружаем бай ордера
	url = "http://eve-marketdata.com/price_check.php?region_id=-4&type=buy&type_id=" + sid
	doc, err = goquery.NewDocument(url)
	if err != nil {
		LogPrint(Fmts("%#v", err))
		LogPrintAndExit("request send error: \n url: " + url + "\n\n")
	}
	selb := doc.Find(".price_check tr")
	selb_type := "eve-marketdata.com"
	if len(selb.Nodes) < 2 {
		//LogPrint("skip " + sid + ": not found buy_orders")
		skip++
	}

	//если каких то ордеров нет, то пробуем загрузить их с другого сайта
	if skip > 0 {
		url = "https://eve-central.com/home/quicklook.html?typeid=" + sid
		doc, err = goquery.NewDocument(url)
		if err != nil {
			LogPrint(Fmts("%#v", err))
			LogPrintAndExit("request send error: \n url: " + url + "\n\n")
		}

		if len(sels.Nodes) < 2 {
			sels = doc.Find("#sell_orders tr")
			sels_type = "eve-central.com"
			if len(sels.Nodes) < 2 {
				Print("skip " + sid + ": not found sell_orders \n")
				return
			}
		}

		if len(selb.Nodes) < 2 {
			selb = doc.Find("#buy_orders tr")
			selb_type = "eve-central.com"
			if len(selb.Nodes) < 2 {
				Print("skip " + sid + ": not found buy_orders \n")
				return
			}
		}
	}

	<-delete_old_item //ждем пока удалятся старые данные

	if loadtype == 0 {
		name = s.Replace(name, "'", "''", -1)
		query := `INSERT INTO itemtype(id,name) VALUES(` + sid + `,'` + name + `')`
		_, err = db.Exec(query)
		LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
	}

	//загружаем ордера:
	load_sell := make(chan bool, 1)
	load_buy := make(chan bool, 1)
	go loadprices3(sels_type, sels, "sell_order", sid, load_sell)
	go loadprices3(selb_type, selb, "buy_order", sid, load_buy)

	<-load_sell
	<-load_buy

	commit_db()

	info := sels_type
	if sels_type != selb_type {
		info = info + " / " + selb_type
	}

	LogPrint("load " + sid + ": " + name + ";  " + Itoa(len(sels.Nodes)) + " / " + Itoa(len(selb.Nodes)) + " " + info)
}

func commit_db() {
	query := `commit`
	_, err := db.Exec(query)
	LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)
}

func loadprices3(stype string, sel *goquery.Selection, tablename string, sid string, end_load chan bool) {
	if stype == "eve-marketdata.com" {
		for i, _ := range sel.Nodes {
			t := sel.Eq(i).Find("td")
			if len(t.Nodes) < 4 {
				continue
			}

			station := Trim(t.Eq(0).Text())

			price := s.Replace(Trim(t.Eq(2).Text()), ",", "", -1)
			price = s.Replace(price, "ISK", "", -1)
			price = s.Replace(price, "NPC", "", -1)
			//price = s.Replace(price, ".", "", -1)
			cnt := s.Replace(Trim(t.Eq(1).Text()), ",", "", -1)
			cnt = mf.StrRegexpReplace(cnt, "\\(Min: [\\d,]+\\)", "")
			expires := Trim(t.Eq(3).Text()) + " " + mf.CurTimeStr()

			query := `INSERT INTO ` + tablename + `(id,station,price,cnt,expires) 
		         VALUES(` + sid + `,'` + station + `',` + price + `*100,` + cnt + `,'` + expires + `')`
			_, err := db.Exec(query)
			LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

			if i%10 == 0 {
				commit_db()
			}
		}
	} else if stype == "eve-central.com" {
		for i, _ := range sel.Nodes {
			t := sel.Eq(i).Find("td")
			if len(t.Nodes) < 6 {
				continue
			}

			station := Trim(t.Eq(0).Text()) + " > " + s.Replace(Trim(t.Eq(1).Text()), "[-]", "", 1)

			price := s.Replace(Trim(t.Eq(2).Text()), ",", "", -1)
			//price = s.Replace(price, ".", "", -1)
			cnt := s.Replace(Trim(t.Eq(3).Text()), ",", "", -1)
			cnt = mf.StrRegexpReplace(cnt, "\\(Min: [\\d,]+\\)", "")
			expires := Trim(t.Eq(4).Text())

			query := `INSERT INTO ` + tablename + `(id,station,price,cnt,expires) 
		         VALUES(` + sid + `,'` + station + `',` + price + `*100,` + cnt + `,'` + expires + `')`
			_, err := db.Exec(query)
			LogPrintErrAndExit("ОШИБКА выполнения запроса: \n"+query+"\n\n", err)

			if i%10 == 0 {
				commit_db()
			}
		}
	} else {
		LogPrintErrAndExit("ОШИБКА нет обработки для: stype == "+stype+"\n\n", errors.New("Some problem"))
	}

	end_load <- true
}

/************************
SELECT
  (a.sell_price1 * MINVALUE(a.sell_cnt1,a.buy_cnt2)) AS sell1,
  (a.buy_price2 * MINVALUE(a.sell_cnt1,a.buy_cnt2)) AS buy2,
  MINVALUE(a.sell_cnt1,a.buy_cnt2) AS cnt,
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
WHERE a.id > 1700
) a
WHERE a.sell_price2 < a.buy_price2
  --AND a.sell_price2 * MINVALUE(a.sell_cnt2,a.buy_cnt2) < a.buy_price2 * MINVALUE(a.sell_cnt2,a.buy_cnt2)
  --AND a.sell_price2 * MINVALUE(a.sell_cnt2,a.buy_cnt2) < a.buy_price2 * MINVALUE(a.sell_cnt2,a.buy_cnt2) - 1000000*100
  AND a.sell_price1 * MINVALUE(a.sell_cnt1,a.buy_cnt1) < a.buy_price1 * MINVALUE(a.sell_cnt1,a.buy_cnt1) - 1*1000*1000
  AND a.sell_price1 * MINVALUE(a.sell_cnt1,a.buy_cnt2) < a.buy_price2 * MINVALUE(a.sell_cnt1,a.buy_cnt2) - 1*1000*1000

ORDER BY prof DESC

***********************************/
