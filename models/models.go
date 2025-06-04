package models

import "time" // Necesario para el campo Timestamp de BCVRate

// Response para la ruta principal
type Response struct {
	BCV float64 `json:"bcv"`
}

// PlansResponse para la ruta /plans
type PlansResponse struct {
	Price20 float64 `json:"price_20"`
	Price25 float64 `json:"price_25"`
	Price30 float64 `json:"price_30"`
}

// ConversionResponse para la ruta /convert
type ConversionResponse struct {
	Conversion float64 `json:"conversion"`
}

// BCVRate representa el documento que se guardará en MongoDB
type BCVRate struct {
	ID        string    `json:"id,omitempty" bson:"_id,omitempty"` // Opcional para MongoDB, usa ObjectID
	Value     float64   `json:"value" bson:"value"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}