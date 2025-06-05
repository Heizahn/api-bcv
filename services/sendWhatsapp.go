// services/whatsapp_service.go
package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"precio-bcv-go/config" // Importar la configuración para acceder a las URL y números
)

// WhatsAppService maneja el envío de alertas vía WhatsApp a través de una API interna.
type WhatsAppService struct {
	apiURL   string
	toNumber string
	client   *http.Client // Cliente HTTP para hacer las solicitudes
}

// NewWhatsAppService crea e inicializa una nueva instancia de WhatsAppService.
// Recibe la configuración de la aplicación para obtener la URL de la API y el número de destino.
func NewWhatsAppService(appConfig *config.Config) *WhatsAppService {
	// Puedes configurar el cliente HTTP aquí, por ejemplo, con un timeout.
	httpClient := &http.Client{Timeout: 10 * time.Second}

	return &WhatsAppService{
		apiURL:   appConfig.WhatsAppAPIURL,
		toNumber: appConfig.WhatsAppToNumber,
		client:   httpClient,
	}
}

// SendAlert envía un mensaje de alerta a través de la API interna de WhatsApp.
// Retorna un error si la solicitud falla o la API devuelve un estado no exitoso.
func (ws *WhatsAppService) SendAlert(message string) error {
	if ws.apiURL == "" || ws.toNumber == "" {
		log.Println("Advertencia: WhatsApp API URL o número de destino no configurados. No se puede enviar alerta.")
		return fmt.Errorf("configuración de WhatsApp API incompleta")
	}

	// Estructura del cuerpo de la solicitud JSON para tu API de WhatsApp
	// ¡Ajusta esto según cómo espere los datos tu API interna!
	requestBody, err := json.Marshal(map[string]string{
		"tlf":      ws.toNumber,
		"body": message,
	})
	if err != nil {
		return fmt.Errorf("error al serializar cuerpo de la solicitud de WhatsApp: %w", err)
	}

	req, err := http.NewRequest("POST", ws.apiURL + "/send-text", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("error al crear solicitud HTTP para WhatsApp API: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ws.client.Do(req)
	if err != nil {
		return fmt.Errorf("error al enviar solicitud a WhatsApp API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("error al enviar alerta por WhatsApp. Código de estado: %d, Respuesta: %s", resp.StatusCode, string(responseBody))
	}

	log.Println("Alerta de WhatsApp enviada exitosamente.")
	return nil
}
