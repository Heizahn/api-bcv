package utils

import (
	"fmt"
	"strconv"
)

// FormatFloat retorna un número con 2 decimales
func FormatFloat(f float64) float64 {
	formateNum := fmt.Sprintf("%.2f", f) // Original: formateNum

	convertNum, err := strconv.ParseFloat(formateNum, 64) // Original: convertNum, err

	if err != nil {
		fmt.Printf("Error al formatear float: %v\n", err)
		return 0.0
	}
	return convertNum
}