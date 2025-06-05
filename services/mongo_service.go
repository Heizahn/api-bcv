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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10 segundos de timeout
	defer cancel() // Importante para liberar los recursos del contexto

	bcvRateDocument := models.BCVRate{
		Value: rateValue,
		Timestamp: recordTimestamp.UTC(),
	}

	_, insertErr := service.collection.InsertOne(ctx, bcvRateDocument)

	if insertErr != nil {
		return fmt.Errorf("error al insertar BCVRate en MongoDB: %w", insertErr)
	}
	log.Printf("BCVRate %.4f guardado en MongoDB con fecha %s (UTC).", rateValue, recordTimestamp.UTC().Format("2006-01-02")) // Log también en UTC
	return nil
}

func (service *MongoDBService) GetLatestBCVRate() (float64, error) {
	var latestBCVRecord models.BCVRate
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10 segundos de timeout
	defer cancel() // Importante para liberar los recursos del contexto
	// Ordena por timestamp descendente para obtener el documento más reciente.
	findOptions := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	decodeErr := service.collection.FindOne(ctx, bson.M{}, findOptions).Decode(&latestBCVRecord)
	
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
    
    // Crea un nuevo contexto con un timeout para esta operación específica.
    // Puedes ajustar la duración (ej. 5*time.Second, 10*time.Second) según la latencia de tu DB.
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) 
    defer cancel() // Es crucial llamar a cancel para liberar recursos del contexto.

    // Asegura que la fecha esté en la zona horaria local para una comparación precisa del día.
    // Sin embargo, para MongoDB, es mejor trabajar con UTC para las consultas.
    // Convertir startOfCurrentDay y endOfCurrentDay a UTC.
    currentTime := time.Now().In(time.Local) 
    
    // Define el inicio y fin del día actual en la zona horaria local,
    // y luego convertirlos a UTC para la consulta de MongoDB.
    startOfCurrentDayLocal := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location()) 
    endOfCurrentDayLocal := startOfCurrentDayLocal.Add(24*time.Hour).Add(-time.Nanosecond)

    // Convertir a UTC para el filtro de la base de datos
    startOfCurrentDayUTC := startOfCurrentDayLocal.UTC()
    endOfCurrentDayUTC := endOfCurrentDayLocal.UTC()


    // Crea el filtro para buscar documentos dentro del rango del día actual en UTC.
    dayFilter := bson.M{
        "timestamp": bson.M{
            "$gte": startOfCurrentDayUTC, // Usar UTC
            "$lte": endOfCurrentDayUTC,   // Usar UTC
        },
    }
    
    // Ordena por timestamp descendente para obtener el valor más reciente si hay múltiples para el mismo día.
    findOptions := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}}) 

    // Pasa el nuevo contexto 'ctx' a la operación de MongoDB.
    decodeErr := service.collection.FindOne(ctx, dayFilter, findOptions).Decode(&bcvTodayRecord) 
    if decodeErr != nil {
        if decodeErr == mongo.ErrNoDocuments {
            return 0, nil // No hay un registro para hoy, retorna 0 y sin error.
        }
        return 0, fmt.Errorf("error al obtener BCVRate para hoy de MongoDB: %w", decodeErr)
    }
    return bcvTodayRecord.Value, nil
}