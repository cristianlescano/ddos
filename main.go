package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/leekchan/accounting"
)

func fetch(url string, ch chan bool, sleep int, sizeMB chan float64) {
	resp, err := http.Get(url + "?random=" + strconv.Itoa(rand.Intn(1000000000)))
	if err != nil {
		//fmt.Printf("Error al obtener la URL %s: %s\n", url, err)
		ch <- false // Indicar al canal que la solicitud ha fallado
		return
	}
	defer resp.Body.Close()
	time.Sleep(time.Duration(sleep) * time.Millisecond)

	body, _ := io.ReadAll(resp.Body)

	size := len(body)                // tamaño en bytes
	sizeKB := float64(size) / 1024.0 // tamaño en kilobytes
	sizeMB <- (sizeKB / 1024.0)      // tamaño en megabytes

	if resp.StatusCode != 200 {
		ch <- false // Indicar al canal que la solicitud ha fallado
	} else {
		ch <- true // Indicar al canal que la solicitud ha sido exitosa
	}

}

func main() {
	// Definir la URL a la que se harán las solicitudes
	var urlScan string
	fmt.Print("url: ")
	fmt.Scanf("%s\n", &urlScan)

	// Definir la cantidad de goroutines (hilos) a abrir
	var numGoroutinesInput string
	fmt.Print("nro rutinas: ")
	fmt.Scanf("%s\n", &numGoroutinesInput)

	numGoroutines, _ := strconv.Atoi(numGoroutinesInput)

	if numGoroutines < 1 {
		fmt.Println("El número de goroutines debe ser al menos 1")
		os.Exit(1)
	}

	var numSleepInput string
	fmt.Print("sleep: ")
	fmt.Scanf("%s\n", &numSleepInput)

	numSleep, _ := strconv.Atoi(numSleepInput)

	// Crear un canal para comunicarse entre las goroutines y el hilo principal
	ch := make(chan bool)

	// Crear un canal mostrar el tamaño de la respuesta
	sizeMB := make(chan float64)

	// Iniciar 10 goroutines
	for i := 0; i < numGoroutines; i++ {
		go fetch(urlScan, ch, numSleep, sizeMB)
	}

	ac := accounting.Accounting{
		Symbol:    "",  //El símbolo
		Precision: 0,   // ¿Cuántos "centavos" queremos? (también llamado precisión)
		Thousand:  ".", //Separador de miles
		Decimal:   "",  //Separador de decimales

	}
	// Mantener siempre 10 goroutines activas
	totalRequests := 0
	totalRequestsErr := 0
	totalSize := 0.0
	sizeProm := 0.0
	for {
		select {
		case success := <-ch:
			if success {
				// Incrementar el contador de solicitudes exitosas
				totalRequests++
			} else {
				totalRequestsErr++
			}
			// Lanzar una nueva goroutine para reemplazarla
			go fetch(urlScan, ch, numSleep, sizeMB)

		case size := <-sizeMB:
			totalSize += size
			sizeProm = size * 1024
		}
		sTotalRquests := ac.FormatMoney(totalRequests)
		sTotalRquestsErr := ac.FormatMoney(totalRequestsErr)
		fmt.Fprintf(os.Stdout, "\rTotal de solicitudes exitosas: %s | Errores: %s | Tamaño:  %.2f KB | Transferido: %.2f MB ", sTotalRquests, sTotalRquestsErr, sizeProm, totalSize)
	}
}

// update .syso
// $GOPATH/bin/rsrc -arch 386 -ico img/icon1.ico
// $GOPATH/bin/rsrc -arch amd64 -ico img/icon1.ico

// go build
