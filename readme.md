# ddos

> CLI en Go para ataques DDoS de capa 7 (HTTP/S). Su objetivo es saturar servidores objetivo mediante volumen masivo de peticiones concurrentes con fingerprints de navegador rotados para evadir protecciones (Cloudflare, WAFs, rate limiters).

**Propósito**: Tirar un servidor por agotamiento de recursos (CPU, memoria, conexiones) mediante la cantidad de peticiones simultáneas. NO es un load testing legítimo — la meta es maximizar el throughput de requests para colapsar el target.

## Descripción

Realiza múltiples solicitudes HTTP concurrentes a una o más URLs, manteniendo un número constante de requests activos. Cada request utiliza un perfil de navegador realista diferente para evitar detección por WAF/CDN. Soporta HTTP/1.1 y HTTP/2 automáticamente.

## Características

### Modos de operación

- **Manual**: Ingresás URL y cookies a mano
- **Auto**: Pegás un comando `curl` completo desde el DevTools del navegador y lo parsea automáticamente (incluye método, headers, cookies y body)

### Browser fingerprint rotation

Cada request rota automáticamente los siguientes headers usando perfiles de navegadores reales:

- **User-Agent** — 60+ firmas reales (Chrome 146-148, Firefox 138-140, Edge, Safari, Brave, Opera, Vivaldi, Samsung Internet)
- **Plataformas** — Windows, macOS, Linux, Android, iOS
- **sec-ch-ua / sec-ch-ua-mobile / sec-ch-ua-platform** — coherentes con el User-Agent
- **Accept-Language** — 12 variantes regionales (es-AR, es-MX, en-US, pt-BR, etc.)
- **Accept-Encoding** — gzip/deflate/br/zstd variado
- **Sec-Fetch-\*** — headers de navegación coherentes (Dest, Mode, Site, User)
- **Referer** — generado automáticamente según el contexto (same-origin, cross-site, o ninguno)
- **sec-gpc / DNT** — presencia aleatoria como navegadores reales

### Navigation profiles

Cada request simula un contexto de navegación diferente:

| Perfil | Sec-Fetch-Dest | Sec-Fetch-Mode | Uso |
|--------|---------------|----------------|-----|
| page_navigate | document | navigate | Carga de página principal |
| page_reload | document | navigate | Recarga de página |
| ajax_api | empty | cors | Llamadas a API |
| image_load | image | no-cors | Carga de imágenes |
| script_load | script | no-cors | Carga de scripts |
| stylesheet | style | no-cors | Carga de CSS |

### Cache-busting

Agrega automáticamente parámetros de cache-busting (`_`, `t`, `cb`, `nocache`, etc.) con timestamps o valores aleatorios:
- 20%: sin cache-buster
- 65%: 1 parámetro
- 15%: 2 parámetros

### Jitter en delays

El sleep configurado tiene una variación de ±25% para simular comportamiento humano y evitar patrones detectables.

### Métricas en tiempo real

Muestra en la terminal:

```
Exitosas: 2.278 | Errores: 10.804 | Acierto: 17.4% | Prom: 32.30 KB | Transferido: 71.85 MB | RL CF:6.500 Origin:280 | 429:6.780 (51.8%) | 200:2.278 (17.4%) | 503:270 (2.1%)
```

| Métrica | Descripción |
|---------|-------------|
| **Exitosas** | Requests con status 2xx |
| **Errores** | Requests con status != 2xx (incluye 0 = error de red) |
| **Acierto** | Porcentaje de exitosas sobre el total |
| **Prom** | Tamaño promedio de respuesta en KB |
| **Transferido** | Total de datos descargados (auto-escala KB / MB / GB) |
| **RL CF** | Rate limits (429) que vienen de **Cloudflare** (header `Server: cloudflare`) |
| **RL Origin** | Rate limits (429) que vienen del **servidor de origen** (nginx, apache, la app) |
| **Status codes** | Desglose por código HTTP con cantidad y porcentaje `(x%)` |

### TLS fingerprint spoofing (uTLS)

Usa la librería `github.com/refraction-networking/utls` para spoofear fingerprints TLS reales de navegadores (JA3/JA4):
- **Chrome** → `HelloChrome_120`
- **Firefox** → `HelloFirefox_120`
- **Safari** → `HelloSafari_16_0`

El fingerprint TLS es coherente con el User-Agent seleccionado, haciendo la conexión indistinguible de un navegador real a nivel de handshake.

### HTTP/2 support

El cliente HTTP soporta HTTP/2 y HTTP/1.1 mediante `http2.Transport` con `DialTLS` custom que usa uTLS para el handshake TLS:
- `InsecureSkipVerify` para testing con certs self-signed
- Negociación ALPN (`h2`, `http/1.1`)

### Payload rotation (POST)

Para requests POST/PUT sin body definido, rota automáticamente entre 10 payloads predefinidos:
- **JSON**: login, search, API call, comment, form
- **Form-encoded**: login, search, contact, register
- **XML**: SOAP envelope

Cada payload usa `%d` como placeholder para IDs aleatorios, simulando datos reales.

### Multi-target

Podés atacar múltiples URLs simultáneamente:
- **Manual**: ingresá URLs una por línea (línea vacía termina)
- **Auto**: pegá un curl principal y luego agregá más targets con curls adicionales

Cada request selecciona un target aleatorio del pool, distribuyendo la carga.

## Uso

```sh
go run main.go
```

### Modo Manual
```
modo (manual/auto): manual
URLs (una por línea, línea vacía para terminar):
  target 1> https://ejemplo.com/api/data
  target 2> https://ejemplo.com/api/other
  target 3>
cookies (presiona Enter si no hay): session=abc123
nro rutinas: 50
sleep (ms): 100
```

### Modo Auto (curl)
```
modo (manual/auto): auto
Modo automático: pegá el comando curl completo.
curl> curl 'https://ejemplo.com/api/data' -H 'Cookie: session=abc123' -H 'Authorization: Bearer xyz'
¿Agregar más targets? (Enter para saltar, o pegá otro curl)
curl> curl 'https://ejemplo.com/api/other' -X POST -d '{"test":1}'
curl>
URLs: 2 targets
Method: POST
Headers: 3
nro rutinas: 50
sleep (ms): 100
```

### Parámetros

| Parámetro | Descripción |
|-----------|-------------|
| `nro rutinas` | Cantidad de goroutines concurrentes (mínimo 1) |
| `sleep (ms)` | Delay base entre requests en milisegundos (0 = sin delay) |

## Compilación

```sh
# Ejecutar directamente
go run main.go

# Compilar binario
go build

# Compilar con ícono (Windows)
rsrc -arch 386 -ico img/icon1.ico
rsrc -arch amd64 -ico img/icon1.ico
go build
```

## Dependencias

| Paquete | Uso |
|---------|-----|
| `github.com/474420502/gcurl` | Parseo de comandos curl |
| `github.com/leekchan/accounting` | Formato de números con separadores de miles |
| `github.com/refraction-networking/utls` | TLS fingerprint spoofing (JA3/JA4) |
| `golang.org/x/net/http2` | Soporte HTTP/2 |

## Instalación

```sh
git clone https://github.com/cristianlescano/ddos.git
cd ddos
go mod download
```

## Limitaciones conocidas

- No hay graceful shutdown (Ctrl+C mata sin cleanup)
- No hay proxy support (single IP = rate limited por Cloudflare/WAF)
- No hay manejo de JS challenge (Cloudflare 503 → cf_clearance)
- No hay logging estructurado
- Las métricas se pierden al cerrar (no hay persistencia)
- Alta concurrencia desde una sola IP trigger rate limiting (429)
