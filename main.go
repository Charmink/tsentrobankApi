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
	"sync"
	"time"
)

type Valute struct { // Структура валюты для декодирования json
	Name  string
	Value string
}

type ValuteInformation struct { // Структура для хранения информации о валюте
	Valute *Valute
	Date   string
}

func action(url string, valutesCourse map[string][]ValuteInformation, mt *sync.Mutex) { // Функция принимает
	// сгенерированный url и делает запрос к Api центробанка, после чего достает инфориацию о валюте и записывает в мапу
	response, _ := http.Get(url)
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	reader := strings.NewReader(string(body))
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel
	date := ""
	for { // парсим json
		token, err := decoder.Token()
		if err != nil && err != io.EOF {
			panic(err)
		}
		if token == nil {
			break
		}
		switch tp := token.(type) {
		case xml.StartElement:
			if tp.Name.Local == "ValCurs" {
				for _, atr := range tp.Attr { // достаем дату из полученного json
					if atr.Name.Local == "Date" {
						date = atr.Value
					}
				}
			} else if tp.Name.Local == "Valute" { // достаем информацию о валюте
				var v Valute
				decoder.DecodeElement(&v, &tp)
				mt.Lock()                                             // блокируем горутины чтобы записать информацию в мапу
				valutesCourse[v.Name] = append(valutesCourse[v.Name], // записываем информацию о валюте в мапу
					ValuteInformation{&v, date})
				mt.Unlock() // разблокирываем горутины

			}
		}
	}
}

type ValuteStats struct { // структура для удобной записи результата по каждой валюте
	Name    string
	Min     float64
	Max     float64
	Mid     float64
	MaxDate string
	MinDate string
}

func countStats(valuesInfo []ValuteInformation) ValuteStats { // Функция вычисляет данные о максимальном, минимальном и
	// среднем значениях за данный период времени
	valStat := ValuteStats{Name: valuesInfo[0].Valute.Name, Min: math.MaxInt64, Max: math.MinInt64, Mid: 0}
	for _, valute := range valuesInfo {
		value, _ := strconv.ParseFloat(strings.Replace(valute.Valute.Value, ",", ".", -1), 64)
		if value < valStat.Min {
			valStat.Min = value
			valStat.MinDate = valute.Date
		}
		if value > valStat.Max {
			valStat.Max = value
			valStat.MaxDate = valute.Date
		}
		valStat.Mid += value
	}
	valStat.Mid /= float64(len(valuesInfo))
	return valStat
}

func main() {
	currentTime := time.Now()
	var mt sync.Mutex
	var wg sync.WaitGroup
	valutesCourse := make(map[string][]ValuteInformation)
	url := "http://www.cbr.ru/scripts/XML_daily_eng.asp?date_req="
	for i := 0; i <= 90; i++ { // генерируем дату для обращения к Api и запускаем отдельный поток для
		// получения информации из api
		oldDate := currentTime.AddDate(0, 0, -i)
		stringDate := oldDate.Format("02/01/2006")
		wg.Add(1)
		go func(url string) {
			action(url+stringDate, valutesCourse, &mt)
			wg.Done()

		}(url)
	}
	wg.Wait()                             // ожидаем горутины получающие данные из Api
	for _, value := range valutesCourse { // вычисляем статистику валюты за данный период в многопоточном режиме
		wg.Add(1)
		go func(value []ValuteInformation) {
			result := countStats(value)
			mt.Lock()
			fmt.Printf("Valute: %s\n", result.Name)
			fmt.Printf("Max Value: %f Date: %s\n", result.Max, result.MaxDate)
			fmt.Printf("Min Value: %f Date: %s\n", result.Min, result.MinDate)
			fmt.Printf("Mid Value: %f\n", result.Mid)
			fmt.Println("--------------------------------------")
			mt.Unlock()
			wg.Done()
		}(value)
	}
	wg.Wait()
}
