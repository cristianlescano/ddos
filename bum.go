package main

import (
	"fmt"
	"net/http"
	"os"
)

func fetch(url string, ch chan bool, countCh chan int) {
	resp, err := http.Get(url)
	if err != nil {
		//fmt.Printf("Error al obtener la URL %s: %s\n", url, err)
		ch <- false // Indicar al canal que la solicitud ha fallado
		return
	}
	defer resp.Body.Close()

	// Incrementar el contador
	countCh <- 1

	// No almacenar el contenido, solo imprimir el estado de la respuesta
	//fmt.Printf("Descargado: %s - Estado: %s\n", url, resp.Status)
	ch <- true // Indicar al canal que la solicitud ha sido exitosa
}

func main() {
	// Definir la URL a la que se harán las solicitudes
	var url string
	fmt.Print("url: ")
	fmt.Scanf("%s\n", &url)

	// Definir la cantidad de goroutines (hilos) a abrir
	numGoroutines := 5000

	// Crear un canal para comunicarse entre las goroutines y el hilo principal
	ch := make(chan bool)

	// Crear un canal para contar la cantidad de solicitudes exitosas
	countCh := make(chan int)

	// Iniciar 10 goroutines
	for i := 0; i < numGoroutines; i++ {
		go fetch(url, ch, countCh)
	}

	// Mantener siempre 10 goroutines activas
	totalRequests := 0
	totalRequestsErr := 0
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
			go fetch(url, ch, countCh)
		case count := <-countCh:
			// Imprimir el total de solicitudes exitosas
			fmt.Fprintf(os.Stdout, "\rTotal de solicitudes exitosas: %d, Errores: %d ", totalRequests+count, totalRequestsErr)
		}
	}
}

// go build -o bum.exe bum.go
