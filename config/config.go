package config

import (
	"fmt" // Importa fmt para usar fmt.Errorf
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

// Config guarda las variables de configuración de la aplicación
type Config struct {
	Port           string
	MongoDBURI     string
	DatabaseName   string
	CollectionName string
	WhatsAppAPIURL string 
	WhatsAppToNumber string
}

// LoadConfig carga las variables de entorno desde un archivo .env.
// Retorna un puntero a Config si todo es exitoso, o un error si alguna variable requerida falta.
func LoadConfig() (*Config, error) {
	// Obtener la ruta completa del archivo actual (config.go)
	_, currentFilePath, _, _ := runtime.Caller(0) 
	configDir := filepath.Dir(currentFilePath)

	// Construir la ruta al archivo .env, asumiendo que está en la raíz del proyecto (un nivel arriba de 'config' dir).
	envFilePath := filepath.Join(configDir, "..", ".env")

	log.Printf("Intentando cargar .env desde: %s\n", envFilePath)

	loadEnvErr := godotenv.Load(envFilePath) // Renombrado: 'err' -> 'loadEnvErr'
	if loadEnvErr != nil {
		log.Printf("Advertencia: No se pudo cargar el archivo .env desde %s. Usando variables de entorno del sistema o valores por defecto. Error: %v", envFilePath, loadEnvErr)
	} else {
		log.Println(".env cargado exitosamente.")
	}

	// Obtener cada variable de entorno requerida.
	// Si alguna variable no se encuentra o está vacía, se retorna un error fatal.
	appPort, getPortErr := getRequiredEnv("PORT") 
	if getPortErr != nil {
		return nil, fmt.Errorf("variable de entorno faltante: %w", getPortErr)
	}

	dbURI, getDBURIErr := getRequiredEnv("MONGODB_URI")
	if getDBURIErr != nil {
		return nil, fmt.Errorf("variable de entorno faltante: %w", getDBURIErr)
	}

	dbName, getDBNameErr := getRequiredEnv("DATABASE_NAME")
	if getDBNameErr != nil {
		return nil, fmt.Errorf("variable de entorno faltante: %w", getDBNameErr)
	}

	collectionName, getCollectionNameErr := getRequiredEnv("COLLECTION_NAME") 
	if getCollectionNameErr != nil {
		return nil, fmt.Errorf("variable de entorno faltante: %w", getCollectionNameErr)
	}

	// --- OBTENER LAS NUEVAS VARIABLES DE ENTORNO ---
	whatsAppAPIURL, getWhatsAppAPIURLErr := getRequiredEnv("WHATSAPP_API_URL")
	if getWhatsAppAPIURLErr != nil {
		// Considera si quieres que esto sea un error fatal o solo una advertencia.
		// Para una alerta, probablemente quieras que sea requerido.
		return nil, fmt.Errorf("variable de entorno faltante: %w", getWhatsAppAPIURLErr)
	}

	whatsAppToNumber, getWhatsAppToNumberErr := getRequiredEnv("WHATSAPP_TO_NUMBER")
	if getWhatsAppToNumberErr != nil {
		return nil, fmt.Errorf("variable de entorno faltante: %w", getWhatsAppToNumberErr)
	}

	// Si todas las variables requeridas se encuentran y tienen valor, se retorna la configuración final.
	return &Config{
		Port:           appPort,
		MongoDBURI:     dbURI,
		DatabaseName:   dbName,
		CollectionName: collectionName,
		// --- ASIGNAR LAS NUEVAS VARIABLES ---
		WhatsAppAPIURL: whatsAppAPIURL,
		WhatsAppToNumber: whatsAppToNumber,
	}, nil // Retorna nil para el error, indicando éxito.
}

// getRequiredEnv obtiene el valor de una variable de entorno especificada por 'key'.
// Retorna el valor de la variable o un error si la variable no existe o está vacía.
func getRequiredEnv(key string) (string, error) {
	envValue, exists := os.LookupEnv(key) 
	if !exists || envValue == "" {
		return "", fmt.Errorf("'%s' no encontrada o vacía. Es una variable de entorno requerida", key)
	}
	return envValue, nil
}