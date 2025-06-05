package services

import (
	"crypto/tls"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// BCVService maneja la lógica para obtener, almacenar y proporcionar el valor actual del BCV.
type BCVService struct {
	bcvValueMutex sync.Mutex 
	currentBCV    float64   
	dbService     *MongoDBService 
	whatsAppService *WhatsAppService
}

// NewBCVService crea e inicializa una nueva instancia de BCVService.
func NewBCVService(mongoDBService *MongoDBService, whatsappAppService *WhatsAppService) *BCVService {
	return &BCVService{
		currentBCV: 0.0, // Inicializa el valor actual del BCV a 0.0; será actualizado en la primera llamada a UpdateBCV.
		dbService:  mongoDBService,
 		whatsAppService: whatsappAppService,
	}
}

// GetBCV obtiene el valor actual del BCV de forma segura para concurrencia.
func (service *BCVService) GetBCV() float64 { 
	service.bcvValueMutex.Lock()
	defer service.bcvValueMutex.Unlock()
	return service.currentBCV
}

// UpdateBCV actualiza el valor interno del BCV.
// Primero intenta obtener el valor de la base de datos para el día actual.
// Si no lo encuentra, realiza un scrapeo desde el BCV.
// Si el scrapeo es exitoso, guarda el nuevo valor en la base de datos.
// Si tanto la DB como el scrapeo fallan, intenta obtener el último valor conocido de la DB.
func (service *BCVService) UpdateBCV() { 
	log.Println("Iniciando actualización de BCV...")

	// Intentar obtener el BCV para el día actual de la base de datos.
	bcvTodayFromDB, dbQueryErr := service.dbService.GetBCVRateForToday() 
	if dbQueryErr != nil {
		log.Printf("Advertencia: Error al obtener BCV del día de la base de datos: %v. Intentando scrapeo o último valor conocido...\n", dbQueryErr)
	}

	var fetchedBCVValue float64 
	// time.Local es importante para que la fecha coincida con la zona horaria del servidor.
	currentDayTimestamp := time.Now().In(time.Local) 
	log.Print("Dia Actual: ", currentDayTimestamp)

	if bcvTodayFromDB > 0 {
		// Si se encontró un valor para hoy en la DB, usar ese valor.
		fetchedBCVValue = bcvTodayFromDB
		log.Printf("BCV del día actual obtenido de la base de datos: %.4f\n", fetchedBCVValue)
	} else {
		// Si no hay valor para hoy en la DB, proceder a scrapearlo.
		log.Println("No se encontró BCV para el día actual en la base de datos. Scrapeando...")
		scrapedBCV := service.fetchUSD() 

		if scrapedBCV > 0 {
			// Si el scrapeo fue exitoso, guardarlo en la DB con la fecha de hoy.
			saveErr := service.dbService.SaveBCVRate(scrapedBCV, currentDayTimestamp)
			if saveErr != nil {
				log.Printf("Advertencia: Error al guardar el BCV scrapeado en MongoDB: %v\n", saveErr)
			}
			fetchedBCVValue = scrapedBCV // Actualizar el valor que se usará.
		} else {
			// Si el scrapeo falló (scrapedBCV es 0), intentar obtener el último valor conocido de la DB.
			log.Println("Advertencia: El scrapeo de BCV falló (valor <= 0). Intentando obtener el último valor conocido de la base de datos...")

			 // --- LLAMADA AL NUEVO SERVICIO DE WHATSAPP ---
            alertMessage := "Alerta: El scrapeo del BCV falló y no se pudo obtener un valor válido. Verifique el sitio del BCV o la configuración de la aplicación."
            // Ejecutar en goroutine para no bloquear y manejar el error de forma asíncrona.
            go func(msg string) {
                if sendErr := service.whatsAppService.SendAlert(msg); sendErr != nil {
                    log.Printf("Error al enviar alerta de WhatsApp: %v\n", sendErr)
                }
            }(alertMessage)

			lastKnownBCVFromDB, lastKnownDBSearchErr := service.dbService.GetLatestBCVRate() 
			if lastKnownDBSearchErr != nil {
				log.Printf("Error al obtener el último BCV conocido de la base de datos: %v\n", lastKnownDBSearchErr)
				// fetchedBCVValue permanecerá en 0.0 si no hay ningún valor disponible.
			} else if lastKnownBCVFromDB > 0 {
				fetchedBCVValue = lastKnownBCVFromDB
				log.Printf("Usando el último BCV conocido de la base de datos: %.4f\n", fetchedBCVValue)
			} else {
				log.Println("No se pudo obtener el BCV ni por scrapeo ni de la base de datos. BCV se mantiene en 0.")
			}
		}
	}

	// Proteger la actualización de la variable interna con un mutex.
	service.bcvValueMutex.Lock()
	service.currentBCV = fetchedBCVValue // Actualiza la variable interna con el valor obtenido.
	service.bcvValueMutex.Unlock()

	log.Printf("BCV interno actualizado a: %.4f\n", service.currentBCV)
}

// fetchUSD scrapea el valor del dólar de la página del BCV.
// Retorna el valor scrapeado o 0.0 si ocurre un error o el valor no es válido.
func (service *BCVService) fetchUSD() float64 { 
	collyCollector := colly.NewCollector() 
	var scrapedUSDValue float64 = 0.0  

	// Configurar Colly para ignorar certificados TLS no válidos.
	// NOTA: 'InsecureSkipVerify: true' es SOLO para desarrollo/entornos específicos.
	// No se recomienda en producción por razones de seguridad.
	collyCollector.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

	// Define la lógica a ejecutar cuando Colly encuentra un elemento HTML con el ID "#dolar".
	collyCollector.OnHTML("#dolar", func(element *colly.HTMLElement) {
		// Extraer el texto del elemento y limpiarlo.
		usdTextRaw := element.Text[101:108]
		cleanedUSDText := strings.ReplaceAll(usdTextRaw, ",", ".") 

		// Convertir el texto limpio a un valor float.
		parsedUSD, parseError := strconv.ParseFloat(cleanedUSDText, 64) 
		if parseError != nil {
			log.Printf("Error al parsear float de BCV scrapeado '%s': %v. No se pudo obtener un valor válido.\n", cleanedUSDText, parseError)
			scrapedUSDValue = 0.0 // Asegurar que sea 0.0 si hay un error de parseo.
			return // Salir del handler OnHTML si hay un error de parseo.
		}

		// Validar que el valor sea positivo.
		if parsedUSD > 0 {
			scrapedUSDValue = parsedUSD
		} else {
			log.Println("Advertencia: Valor del dólar scrapeado es <= 0. No se pudo obtener un valor válido.")
			scrapedUSDValue = 0.0 // Asegurar que sea 0.0 si el valor no es positivo.
		}
	})

	// Visitar la URL del BCV para iniciar el proceso de scrapeo.
	visitErr := collyCollector.Visit("https://www.bcv.org.ve/") 
	if visitErr != nil {
		log.Printf("Error al visitar BCV para scrapeo: %v. No se pudo obtener el valor.\n", visitErr)
		return 0.0 // Retornar 0.0 para indicar que el scrapeo falló en la visita.
	}

	return scrapedUSDValue
}