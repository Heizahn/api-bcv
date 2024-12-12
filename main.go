package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/robfig/cron"
)

var (
	bcvMutex sync.Mutex
	bcv      float64
)

func main() {
	// Iniciar cron para la conversión
	shedule := cron.New()
	shedule.AddFunc("30 6 * * *", updateBCV)
	shedule.Start()

	// Iniciar servidor
	updateBCV()
	http.HandleFunc("/", handleResquest)
	http.HandleFunc("/plans", handlePlansRequest)
	http.HandleFunc("/convert", handleConvertRequest)
	fmt.Println("Servidor iniciado en http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func handleResquest(w http.ResponseWriter, r *http.Request) {
	bcvMutex.Lock()
	defer bcvMutex.Unlock()

	response := Response{
		BCV: formatFloat(bcv),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handlePlansRequest(w http.ResponseWriter, r *http.Request) {
	bcvMutex.Lock()
	defer bcvMutex.Unlock()

	response := PlnsResponse{
		Price20: formatFloat((bcv * 20) * 1.08),
		Price25: formatFloat((bcv * 25) * 1.08),
		Price30: formatFloat((bcv * 30) * 1.08),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleConvertRequest(w http.ResponseWriter, r *http.Request) {
	bcvMutex.Lock()
	defer bcvMutex.Unlock()

	amount, err := strconv.ParseFloat(r.URL.Query().Get("amount"), 64)
	if err != nil {
		http.Error(w, "Invalid amount parameter", http.StatusBadRequest)
		return
	}

	response := ConversionResponse{
		Conversion: formatFloat((amount * bcv) * 1.08),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func updateBCV() {
	newBCV := fetchUSD()

	bcvMutex.Lock()
	bcv = newBCV
	bcvMutex.Unlock()

	fmt.Println("BCV actualizado:", bcv)
}

func fetchUSD() float64 {
	c := colly.NewCollector()
	var usd float64

	c.OnHTML("#dolar", func(element *colly.HTMLElement) {
		usdText := element.Text[101:106]
		cleanedText := strings.ReplaceAll(usdText, ",", ".")

		d, err := strconv.ParseFloat(cleanedText, 64)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		usd = d
	})

	err := c.Visit("https://www.bcv.org.ve/")
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}

	return usd
}

type Response struct {
	BCV float64 `json:"bcv"`
}

type PlnsResponse struct {
	Price20 float64 `json:"price_20"`
	Price25 float64 `json:"price_25"`
	Price30 float64 `json:"price_30"`
}

type ConversionResponse struct {
	Conversion float64 `json:"conversion"`
}

func formatFloat(f float64) float64 {
	return roundFloat(f, 2)
}

func roundFloat(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return round(f*shift) / shift
}

func round(f float64) float64 {
	return math.Floor(f + 0.5)
}
