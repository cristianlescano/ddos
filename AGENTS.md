# AGENTS.md — ddos

> Instrucciones para agentes de IA que trabajan en este repositorio.

## Project Overview

**Qué es**: CLI en Go para ataques DDoS de capa 7 (HTTP/S). Su objetivo es saturar servidores objetivo mediante volumen masivo de peticiones concurrentes con fingerprints de navegador rotados para evadir protecciones (Cloudflare, WAFs, rate limiters).

**Propósito**: Tirar un servidor por agotamiento de recursos (CPU, memoria, conexiones) mediante la cantidad de peticiones simultáneas. NO es un load testing legítimo — la meta es maximizar el throughput de requests para colapsar el target.

**Stack**: Go 1.22, single-file (`main.go`), sin tests, sin CI/CD.

**Module**: `github.com/cristianlescano/ddos`

## Architecture

### Single-file structure

Todo vive en `main.go` (~850 líneas). No hay paquetes separados. Si el proyecto crece, refactorizar a:
```
internal/
  profiles/    # browser, navigation, languages, payloads
  rotation/    # header rotation logic
  curl/        # curl parsing wrapper
  metrics/     # real-time metrics display
```

### Core components

| Component | Líneas aprox. | Responsabilidad |
|-----------|---------------|-----------------|
| Structs | 23-57 | `requestConfig`, `browserProfile`, `navigationProfile`, `payloadProfile` |
| Browser profiles | 63-144 | 60+ firmas reales de navegadores |
| Navigation profiles | 150-207 | 6 contextos de navegación |
| Accept-Language pool | 213-226 | 12 variantes regionales |
| Payload profiles | 232-248 | 10 payloads POST (JSON, form, XML) |
| Cache-buster | 232, 326-360 | Parámetros anti-cache automáticos |
| Header rotation | 366-463 | `rotateHeaders()` — corazón del sistema |
| Curl parser | 469-523 | `parseCurl()` — wrapper de gcurl |
| Fetch | 529-614 | HTTP/2 request + jitter + métricas + payload rotation |
| Main | 620-800 | CLI interactiva + loop de métricas + status codes |

### Concurrency model

```
main() ──┬── goroutine 1 ──┐
         ├── goroutine 2 ──┤
         ├── ...           ├──► ch (int: status code) ──► contador por código
         └── goroutine N ──┘                 sizeMB (float64) ──► métricas de tamaño
```

- Cada goroutine llama `fetch()` de forma recursiva (se relanza al completar)
- Dos canales: `ch` (int para status code: 0=error de red, 200-599=HTTP) y `sizeMB` (float64 para tamaño en KB)
- **No hay cancelación ni graceful shutdown** — el programa corre hasta Ctrl+C

### HTTP client

- Usa `http2.Transport` con `DialTLS` custom para soportar HTTP/2 y HTTP/1.1
- **uTLS** (`github.com/refraction-networking/utls`) para spoofear fingerprints TLS reales de navegadores (JA3/JA4)
- `InsecureSkipVerify: true` para testing
- Fingerprints TLS coherentes con User-Agent: Chrome → HelloChrome_120, Firefox → HelloFirefox_120, Safari → HelloSafari_16_0

## Conventions

### Naming

- **Structs**: `camelCase` para fields, `PascalCase` para tipos
- **Funciones**: `camelCase` (no hay exported functions fuera de `main`)
- **Variables**: descriptivas, abreviaturas aceptadas (`cfg`, `nav`, `prom`)
- **Channels**: `ch` para status code, nombre del dato para otros (`sizeMB`)

### Code style

- Go estándar (`gofmt`)
- Separadores de sección con `// ---`
- Comentarios en **español** para lógica, **inglés** para structs/types
- `rand.Seed()` en `main()` — necesario para Go < 1.20

### Error handling

- **Silencioso**: la mayoría de errores se ignoran o envían `0` al canal (error de red)
- **Fatal**: solo en parsing de curl y validación de inputs (con `os.Exit(1)`)
- No hay logging estructurado

## Dependencies

| Package | Versión | Uso | Crítico |
|---------|---------|-----|---------|
| `gcurl` | v1.2.1 | Parseo de comandos curl | Sí |
| `accounting` | v1.0.0 | Formato de números (separador miles) | Sí |
| `golang.org/x/net/http2` | (de go.mod) | Soporte HTTP/2 | Sí |
| `utls` | v1.8.2 | TLS fingerprint spoofing (JA3/JA4) | Sí |
| `requests` | v1.50.0 | Transitive (de gcurl) | No |
| `brotli` | v1.0.5 | Transitive (de requests) | No |
| `gjson` | v1.12.0 | Transitive (de gcurl) | No |

**No agregar dependencias sin justificación**. El proyecto es minimalista.

## How to extend

### Agregar un nuevo navegador

1. Agregar entrada a `browserProfiles` con los 4 fields:
   ```go
   {"User-Agent string", `"sec-ch-ua"`, `"Platform"`, "?0 o ?1"},
   ```
2. El `sec-ch-ua` debe ser coherente con el User-Agent

### Agregar un navigation profile

1. Agregar entrada a `navigationProfiles` con todos los fields
2. El `acceptHeader` debe coincidir con el `secFetchDest`

### Agregar un payload profile

1. Agregar entrada a `payloadProfiles` con los 3 fields:
   ```go
   {"nombre", `template con %d para IDs`, "content-type"},
   ```
2. Usar `%d` para valores aleatorios (soporta 1, 2 o 3 placeholders)

### Agregar un header rotatable

1. Agregar nombre a `rotatableHeaders`
2. Implementar lógica en `rotateHeaders()`

### Agregar métrica nueva

1. Crear canal en `main()`
2. Enviar dato desde `fetch()`
3. Leer en el `select` del loop principal
4. Agregar al `fmt.Fprintf` de la línea ~790

## What NOT to do

- **NO** eliminar `Cookie` ni `Authorization` de los headers — `rotateHeaders()` los preserva explícitamente
- **NO** cambiar el formato del canal `sizeMB` — está en KB, no MB
- **NO** cambiar el canal `ch` a bool — ahora es `chan int` con status codes
- **NO** agregar `context.Context` sin discutir — el diseño actual es fire-and-forget
- **NO** crear paquetes nuevos sin necesidad real — mantener single-file hasta que sea insostenible
- **NO** modificar los perfiles de navegador existentes — solo agregar nuevos
- **NO** cambiar la lógica de jitter — está calibrada para ±25%
- **NO** quitar `InsecureSkipVerify` — es necesario para testing con certs self-signed
- **NO** desacoplar el TLS fingerprint del User-Agent — deben ser coherentes (Chrome UA → HelloChrome_120)

## Build & Run

```sh
# Desarrollo
go run main.go

# Binario Linux/macOS
go build -o ddos

# Binario Windows
GOOS=windows GOARCH=amd64 go build -o ddos.exe

# Con ícono Windows (requiere rsrc)
rsrc -arch amd64 -ico img/icon1.ico -o rsrc_windows_amd64.syso
go build -o ddos.exe
```

## Git conventions

- Commits en español, estilo convencional: `fix:`, `feat:`, `docs:`, `refactor:`
- Mensajes cortos y descriptivos
- NO agregar "Co-Authored-By" ni atribución de IA

## OpenCode context

Este proyecto se trabaja con **OpenCode**. No hay `.cursorrules`, `.windsurfrules` ni configuración específica de IDE.

### Skills relevantes

- **go-testing**: Si se agregan tests (actualmente no hay)
- **sdd-***: Si se usa Spec-Driven Development para features grandes

### Reglas de trabajo

1. **Leer antes de modificar** — siempre leer el archivo completo antes de editar
2. **Preservar convenciones existentes** — no imponer estilos nuevos
3. **Preguntar antes de refactorizar** — si el cambio es estructural, proponer primero
4. **Documentar cambios** — actualizar este archivo si se agregan patrones o convenciones

## Known limitations

- No hay graceful shutdown (Ctrl+C mata sin cleanup)
- No hay proxy support (single IP = rate limited por Cloudflare/WAF)
- No hay manejo de JS challenge (Cloudflare 503 → cf_clearance)
- No hay logging estructurado
- Las métricas se pierden al cerrar (no hay persistencia)
- No hay latency metrics (P50, P95, P99)
- Alta concurrencia desde una sola IP trigger rate limiting (429)
