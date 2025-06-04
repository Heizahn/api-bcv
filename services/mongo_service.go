package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"precio-bcv-go/config"
	"precio-bcv-go/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBService maneja la conexión y operaciones CRUD con MongoDB.
type MongoDBService struct {
	client     *mongo.Client
	collection *mongo.Collection
	ctx        context.Context
	cancel     context.CancelFunc 
}

// NewMongoDBService inicializa un nuevo servicio de MongoDB.
// Establece una conexión con un timeout y verifica su disponibilidad con un ping.
func NewMongoDBService(appConfig *config.Config) (*MongoDBService, error) { 
	// Establece un contexto con un timeout para la conexión inicial.
	connectionCtx, cancelContext := context.WithTimeout(context.Background(), 10*time.Second)
	// Configura las opciones del cliente de MongoDB, aplicando la URI de conexión.
	clientOpts := options.Client().ApplyURI(appConfig.MongoDBURI) 
	
	// Intenta conectar a MongoDB.
	mongoClient, connectErr := mongo.Connect(connectionCtx, clientOpts) 
	if connectErr != nil {
		cancelContext() // Asegúrate de cancelar el contexto si la conexión falla.
		return nil, fmt.Errorf("error al conectar a MongoDB: %w", connectErr)
	}

	// Realiza un ping para verificar que la conexión sea funcional.
	pingErr := mongoClient.Ping(connectionCtx, nil) 
	if pingErr != nil {
		cancelContext() // Asegúrate de cancelar el contexto si el ping falla.
		return nil, fmt.Errorf("error al hacer ping a MongoDB: %w", pingErr)
	}

	log.Println("Conectado a MongoDB!")

	// Obtiene la referencia a la colección específica donde se almacenarán los datos del BCV.
	bcvCollection := mongoClient.Database(appConfig.DatabaseName).Collection(appConfig.CollectionName)

	return &MongoDBService{
		client:     mongoClient,
		collection: bcvCollection,
		ctx:        connectionCtx,
		cancel:     cancelContext,
	}, nil
}

// Disconnect cierra la conexión con MongoDB y libera los recursos del contexto.
func (service *MongoDBService) Disconnect() { 
	if service.client != nil {
		disconnectErr := service.client.Disconnect(service.ctx) 
		if disconnectErr != nil {
			log.Printf("Error al desconectar de MongoDB: %v", disconnectErr)
		}
		service.cancel() // Llama a la función de cancelación del contexto.
		log.Println("Desconectado de MongoDB.")
	}
}

// SaveBCVRate guarda un nuevo registro de la tasa BCV en MongoDB.
func (service *MongoDBService) SaveBCVRate(rateValue float64, recordTimestamp time.Time) error { 
	bcvRateDocument := models.BCVRate{
		Value:     rateValue,
		Timestamp: recordTimestamp,
	}

	_, insertErr := service.collection.InsertOne(service.ctx, bcvRateDocument) 
	if insertErr != nil {
		return fmt.Errorf("error al insertar BCVRate en MongoDB: %w", insertErr)
	}
	log.Printf("BCVRate %.4f guardado en MongoDB con fecha %s.", rateValue, recordTimestamp.Format("2006-01-02"))
	return nil
}

// GetLatestBCVRate obtiene la última tasa de BCV guardada en la colección, sin filtrar por fecha.
func (service *MongoDBService) GetLatestBCVRate() (float64, error) { 
	var latestBCVRecord models.BCVRate // Renombrado: 'bcvRate' -> 'latestBCVRecord'
	// Ordena por timestamp descendente para obtener el documento más reciente.
	findOptions := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}}) 

	decodeErr := service.collection.FindOne(service.ctx, bson.M{}, findOptions).Decode(&latestBCVRecord) 
	if decodeErr != nil {
		if decodeErr == mongo.ErrNoDocuments {
			return 0, nil // No se encontraron documentos, retorna 0 y sin error.
		}
		return 0, fmt.Errorf("error al obtener la última BCVRate de MongoDB: %w", decodeErr)
	}
	return latestBCVRecord.Value, nil
}

// GetBCVRateForToday obtiene la tasa BCV para el día actual.
// Retorna 0.0 y nil si no se encuentra un registro para hoy.
func (service *MongoDBService) GetBCVRateForToday() (float64, error) { 
	var bcvTodayRecord models.BCVRate 
	
	// Asegura que la fecha esté en la zona horaria local para una comparación precisa del día.
	currentTime := time.Now().In(time.Local) 
	
	// Define el inicio y fin del día actual.
	startOfCurrentDay := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location()) 
	endOfCurrentDay := startOfCurrentDay.Add(24*time.Hour).Add(-time.Nanosecond)

	// Crea el filtro para buscar documentos dentro del rango del día actual.
	dayFilter := bson.M{
		"timestamp": bson.M{
			"$gte": startOfCurrentDay,
			"$lte": endOfCurrentDay,
		},
	}
	
	// Ordena por timestamp descendente para obtener el valor más reciente si hay múltiples para el mismo día.
	findOptions := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}}) 

	decodeErr := service.collection.FindOne(service.ctx, dayFilter, findOptions).Decode(&bcvTodayRecord) 
	if decodeErr != nil {
		if decodeErr == mongo.ErrNoDocuments {
			return 0, nil // No hay un registro para hoy, retorna 0 y sin error.
		}
		return 0, fmt.Errorf("error al obtener BCVRate para hoy de MongoDB: %w", decodeErr)
	}
	return bcvTodayRecord.Value, nil
}