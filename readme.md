## Descripción
Este programa en Go realiza múltiples solicitudes HTTP a una URL específica y mantiene un número constante de solicitudes activas. Cuando una solicitud se completa (ya sea con éxito o con un error), se lanza una nueva solicitud para mantener el número constante de solicitudes activas. Además, se puede establecer un retraso entre las solicitudes.

## Funcionamiento
El programa consta de dos funciones principales: fetch y main.

La función fetch realiza una solicitud HTTP a una URL dada. Si la solicitud es exitosa, envía un true al canal ch y incrementa el contador de solicitudes exitosas enviando un 1 al canal countCh. Si la solicitud falla, envía un false al canal ch. Después de cada solicitud, la función fetch duerme durante un tiempo especificado antes de realizar la próxima solicitud.

La función main lanza un número especificado de goroutines, cada una de las cuales realiza solicitudes HTTP utilizando la función fetch. La función main también lee continuamente de los canales ch y countCh para rastrear el número de solicitudes exitosas y fallidas.

## Uso
Para usar este programa, necesitarás tener Go instalado en tu máquina. Puedes compilar y ejecutar el programa utilizando el comando go run main.go en la terminal. Para compilar el programa en un ejecutable, puedes usar el comando go build main.go.

## Instalación y Ejecución
Para usar este programa, necesitarás tener Go instalado en tu máquina. Si no lo tienes, puedes descargarlo e instalarlo desde la página oficial de Go.

Una vez que tienes Go instalado, puedes clonar este repositorio a tu máquina local utilizando Git:
```sh
git clone https://github.com/cristianlescano/ddos.git
```

Navega al directorio del proyecto:
```sh
cd ddos
```

Para compilar y ejecutar el programa, utiliza el comando go run main.go en la terminal:
```sh
go run main.go
```

Para compilar el programa en un ejecutable, puedes usar el comando go build main.go:
```sh
go build
```


## Nota
Este programa es solo para fines educativos y no debe usarse para realizar ataques DDoS u otras actividades malintencionadas.