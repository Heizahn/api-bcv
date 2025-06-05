package main

import (
	"fmt"
	"log"
	"net/http"

	// Importaciones de tus módulos
	"precio-bcv-go/config"
	"precio-bcv-go/handlers"
	"precio-bcv-go/services"

	// Dependencias externas
	gorillaHandlers "github.com/gorilla/handlers" // Alias para el paquete gorilla/handlers
	"github.com/robfig/cron"
)

func main() {
	// --- 1. Cargar Configuración de la Aplicación ---
	// Carga la configuración desde variables de entorno o el archivo .env.
	// La función retornará la configuración o un error fatal si falta algo esencial.
	appConfig, configLoadErr := config.LoadConfig() 
	if configLoadErr != nil {
		log.Fatalf("Error crítico al cargar la configuración de la aplicación: %v", configLoadErr)
	}
	log.Printf("Configuración cargada: Puerto=%s", appConfig.Port,)

	// --- 2. Inicializar Servicio de Base de Datos MongoDB ---
	// Crea una instancia del servicio que gestiona la conexión y operaciones con MongoDB,
	// pasándole la configuración necesaria (URI, nombres de DB/Colección).
	mongoService, mongoServiceInitErr := services.NewMongoDBService(appConfig) 
	if mongoServiceInitErr != nil {
		// Si la conexión a MongoDB falla (ej. servidor no disponible, credenciales incorrectas),
		// la aplicación no puede operar, por lo que se termina.
		log.Fatalf("Error crítico: No se pudo inicializar el servicio de MongoDB: %v", mongoServiceInitErr)
	}
	// Asegura que la conexión a MongoDB se cierre de forma segura cuando la función main() finalice.
	defer mongoService.Disconnect()
	log.Println("Servicio de MongoDB inicializado y conexión establecida.")

	// --- 3. Inicializar Servicio de Tasa de Cambio BCV ---
	// Crea una instancia del servicio que se encarga de obtener y mantener el valor del BCV.
	// Se le inyecta el 'mongoService' para que pueda guardar los valores en la base de datos.
	whatsAppService := services.NewWhatsAppService(appConfig) // Pasa la configuración
	log.Println("Servicio de WhatsApp inicializado.")

	bcvPriceService := services.NewBCVService(mongoService, whatsAppService) // Renombrado: 'bcvService' -> 'bcvPriceService'
	log.Println("Servicio de BCV inicializado.")

	// --- 4. Realizar la Primera Actualización de la Tasa BCV al Arrancar el Servidor ---
	// Esto asegura que tengamos un valor inicial del BCV disponible antes de que lleguen
	// las primeras peticiones HTTP, consultando primero la base de datos o scrapeando.
	bcvPriceService.UpdateBCV()
	log.Printf("Valor inicial del BCV establecido: %.4f\n", bcvPriceService.GetBCV())

	// --- 5. Configurar Tarea Programada (Cron) para la Actualización Diaria del BCV ---
	dailyPriceScheduler := cron.New()
	dailyPriceScheduler.AddFunc("0 30 1 * * *", bcvPriceService.UpdateBCV) 
	dailyPriceScheduler.Start()
	log.Println("Con Activado")

	// --- 6. Configurar CORS (Cross-Origin Resource Sharing) para la API ---
	// Define las políticas de seguridad para permitir solicitudes de diferentes dominios.
	// Por ahora, se permite cualquier origen, cabecera y método para simplificar el desarrollo.
	// En producción, es crucial restringir esto a dominios específicos.
	corsAllowedOrigins := gorillaHandlers.AllowedOrigins([]string{"*"})  
	corsAllowedHeaders := gorillaHandlers.AllowedHeaders([]string{"Content-Type"}) 
	corsAllowedMethods := gorillaHandlers.AllowedMethods([]string{"GET", "OPTIONS"}) 
	log.Println("Configuración de CORS aplicada (permitiendo todos los orígenes para desarrollo).")

	// --- 7. Inicializar Manejadores de Rutas API ---
	// Crea una instancia de los manejadores HTTP que procesarán las solicitudes a las rutas de la API.
	// Se le inyecta el 'bcvPriceService' para que los manejadores puedan acceder al valor del BCV.
	apiRoutesHandlers := handlers.NewAPIHandlers(bcvPriceService) 
	log.Println("Manejadores de API inicializados.")

	// --- 8. Configurar Rutas HTTP y sus Manejadores ---
	// Asigna cada URL de la API a su función manejadora correspondiente.
	// Cada manejador es un método de la instancia 'apiRoutesHandlers'.
	http.HandleFunc("/", apiRoutesHandlers.HandleRequest)
	http.HandleFunc("/plans", apiRoutesHandlers.HandlePlansRequest)
	http.HandleFunc("/convert", apiRoutesHandlers.HandleConvertRequest)
	log.Println("Rutas HTTP configuradas.")

	// --- 9. Iniciar Servidor HTTP ---
	// Comienza a escuchar en el puerto configurado y a procesar las solicitudes entrantes.
	// Se aplica la configuración de CORS a todas las rutas usando el multiplexor HTTP por defecto.
	fmt.Printf("Servidor iniciado y escuchando en el puerto %s\n", appConfig.Port)
	http.ListenAndServe(":"+appConfig.Port, gorillaHandlers.CORS(corsAllowedOrigins, corsAllowedHeaders, corsAllowedMethods)(http.DefaultServeMux))
	// log.Fatal es una función que, si ListenAndServe retorna un error (ej. el puerto ya está en uso),
	// imprime el error y termina la aplicación.
}