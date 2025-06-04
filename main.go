package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/gorilla/handlers"
	"github.com/robfig/cron"
)

var (
	bcvMutex sync.Mutex
	bcv      float64
)

func main() {
	// Iniciar cron para la conversión
	shedule := cron.New()
	shedule.AddFunc("0 30 6 * * *", updateBCV)
	shedule.Start()

	//Cors
	corsOrigin := handlers.AllowedOrigins([]string{"*"})
	corsHeader := handlers.AllowedHeaders([]string{"Content-Type"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "OPTIONS"})

	// Iniciar servidor
	updateBCV()
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	http.HandleFunc("/", handleRequest)
	http.HandleFunc("/plans", handlePlansRequest)
	http.HandleFunc("/convert", handleConvertRequest)
	fmt.Println("Servidor iniciado")
	http.ListenAndServe(":8080", handlers.CORS(corsOrigin, corsHeader, corsMethods)(http.DefaultServeMux))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	bcvMutex.Lock()
	defer bcvMutex.Unlock()

	response := Response{
		BCV: bcv,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handlePlansRequest(w http.ResponseWriter, r *http.Request) {
	bcvMutex.Lock()
	defer bcvMutex.Unlock()

	response := PlansResponse{
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
		usdText := element.Text[101:108]
		cleanedText := strings.ReplaceAll(usdText, ",", ".")

		d, err := strconv.ParseFloat(cleanedText, 64)
		if err != nil {
			d = 97.4194
			fmt.Println("Error:", err)
			return
		}

		if d > 0 {
			usd = d
		} else {
			fmt.Println("Error: No se pudo obtener el valor del dólar")
		}
	})

	err := c.Visit("https://www.bcv.org.ve/")
	if err != nil {
		fmt.Println("Error:", err)
		return 97.4194
	}

	return usd
}

type Response struct {
	BCV float64 `json:"bcv"`
}

type PlansResponse struct {
	Price20 float64 `json:"price_20"`
	Price25 float64 `json:"price_25"`
	Price30 float64 `json:"price_30"`
}

type ConversionResponse struct {
	Conversion float64 `json:"conversion"`
}

func formatFloat(f float64) float64 {
	//Retornar un numero con 4 decimales
	formateNum := fmt.Sprintf("%.2f", f)

	convertNum, err := strconv.ParseFloat(formateNum, 64)

	if err != nil {
		fmt.Println("Error: ", err)
		return 0.0
	}

	return convertNum
}


