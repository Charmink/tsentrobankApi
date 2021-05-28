package main

import (
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Valute struct {
	Name string
	Value string
}

type ValuteInformation struct {
	Valute *Valute
	Date string
}

func action(url string, valutesCourse map[string][]ValuteInformation){
	response, _ := http.Get(url)
	body, _ := io.ReadAll(response.Body)
	reader := strings.NewReader(string(body))
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel
	date := ""
	for{
		token, err := decoder.Token()
		if err != nil && err != io.EOF{
			panic(err)
		}
		if token == nil{
			break
		}
		switch tp := token.(type) {
		case xml.StartElement:
			if tp.Name.Local == "ValCurs"{
				for _, atr := range tp.Attr{
					if atr.Name.Local == "Date"{
						date = atr.Value
					}
				}
			}else if tp.Name.Local == "Valute"{
				var v Valute
				decoder.DecodeElement(&v, &tp)
				valutesCourse[v.Name] = append(valutesCourse[v.Name],
					ValuteInformation{&v, date})

			}
		}
	}
}

type ValuteStats struct {
	Name string
	Min float64
	Max float64
	Mid float64
	MaxDate string
	MinDate string
}

func countStats(valuesInfo []ValuteInformation) ValuteStats{
	valStat := ValuteStats{Name: valuesInfo[0].Valute.Name, Min: math.MaxInt64, Max: math.MinInt64, Mid: 0}
	for _, valute := range valuesInfo{
		value, _ := strconv.ParseFloat(strings.Replace(valute.Valute.Value, ",", ".", -1), 64)
		if value < valStat.Min {
			valStat.Min = value
			valStat.MinDate = valute.Date
		}
		if value > valStat.Max{
			valStat.Max = value
			valStat.MaxDate = valute.Date
		}
		valStat.Mid += value
	}
	valStat.Mid /= float64(len(valuesInfo))
	return valStat
}

func main()  {
	currentTime := time.Now()
	valutesCourse := make(map[string][]ValuteInformation)
	url := "http://www.cbr.ru/scripts/XML_daily_eng.asp?date_req="
	for i := 0; i <= 90; i ++{
		oldDate := currentTime.AddDate(0,0,-i)
		stringDate := oldDate.Format("02/01/2006")
		action(url + stringDate, valutesCourse)
	}
	for _, value := range valutesCourse{
		result := countStats(value)
		fmt.Printf("Valute: %s\n", result.Name)
		fmt.Printf("Max Value: %f Date: %s\n", result.Max, result.MaxDate)
		fmt.Printf("Min Value: %f Date: %s\n", result.Min, result.MinDate)
		fmt.Printf("Mid Value: %f\n", result.Mid)
		fmt.Println("--------------------------------------")
	}
}
