package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"precio-bcv-go/models"
	"precio-bcv-go/services"
	"precio-bcv-go/utils"
)

// APIHandlers contiene las dependencias de servicio necesarias para manejar las peticiones HTTP de la API.
type APIHandlers struct {
	BCVValueService *services.BCVService
}

// NewAPIHandlers es el constructor para crear una nueva instancia de APIHandlers.
func NewAPIHandlers(bcvServiceInstance *services.BCVService) *APIHandlers {
	return &APIHandlers{
		BCVValueService: bcvServiceInstance,
	}
}

// HandleRequest maneja la ruta raíz ("/") de la API, retornando el valor actual del BCV.
func (apiHandler *APIHandlers) HandleRequest(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
	print("\nMOSTRAR EL VALOR DEL BCV")
	currentBCVValue := apiHandler.BCVValueService.GetBCV()
	
	jsonResponse := models.Response{
		BCV: currentBCVValue,
	}

	httpResponseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpResponseWriter).Encode(jsonResponse)
}

// HandlePlansRequest maneja la ruta "/plans" de la API, retornando precios de planes calculados.
func (apiHandler *APIHandlers) HandlePlansRequest(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentBCVValue := apiHandler.BCVValueService.GetBCV()
	const taxRate = 1.08 // Tasa de impuesto del 8%

	plansResponse := models.PlansResponse{
		Price20: utils.FormatFloat((currentBCVValue * 20) * taxRate),
		Price25: utils.FormatFloat((currentBCVValue * 25) * taxRate),
		Price30: utils.FormatFloat((currentBCVValue * 30) * taxRate),
	}

	httpResponseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpResponseWriter).Encode(plansResponse)
}

// HandleConvertRequest maneja la ruta "/convert" de la API, convirtiendo un monto dado.
func (apiHandler *APIHandlers) HandleConvertRequest(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
	amountToConvert, parseErr := strconv.ParseFloat(httpRequest.URL.Query().Get("amount"), 64)
	if parseErr != nil {
		http.Error(httpResponseWriter, "Invalid amount parameter", http.StatusBadRequest)
		return
	}

	currentBCVValue := apiHandler.BCVValueService.GetBCV()
	const taxRate = 1.08 // Tasa de impuesto del 8%

	conversionResult := models.ConversionResponse{
		Conversion: utils.FormatFloat((amountToConvert * currentBCVValue) * taxRate),
	}

	httpResponseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(httpResponseWriter).Encode(conversionResult)
}