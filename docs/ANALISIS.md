# Análisis técnico de UltraCopySam

Documento orientado a desarrolladores: explica **qué hace el proyecto, cómo está
construido y por qué se tomó cada decisión**, incluyendo las que se probaron y
se descartaron. Está pensado para que alguien que llega nuevo pueda modificar el
código sin repetir callejones sin salida ya explorados.

Para el uso de la herramienta, ver el [README](../README.md).

---

## Índice

- [1. Qué es y qué no es](#1-qué-es-y-qué-no-es)
- [2. Panorama del código](#2-panorama-del-código)
- [3. Arquitectura del motor](#3-arquitectura-del-motor)
- [4. Modelo de concurrencia](#4-modelo-de-concurrencia)
- [5. Gestión de memoria](#5-gestión-de-memoria)
- [6. La capa Win32](#6-la-capa-win32)
- [7. Rutas: el problema de MAX_PATH](#7-rutas-el-problema-de-max_path)
- [8. Modo incremental (`-u`)](#8-modo-incremental--u)
- [9. Validación de entrada](#9-validación-de-entrada)
- [10. Manejo de errores](#10-manejo-de-errores)
- [11. Rendimiento: metodología y resultados](#11-rendimiento-metodología-y-resultados)
- [12. Hipótesis descartadas](#12-hipótesis-descartadas)
- [13. Trampas de Windows aprendidas](#13-trampas-de-windows-aprendidas)
- [14. Distribución y CI](#14-distribución-y-ci)
- [15. Limitaciones y deuda técnica](#15-limitaciones-y-deuda-técnica)
- [16. Cómo extenderlo](#16-cómo-extenderlo)

---

## 1. Qué es y qué no es

**Es** una utilidad de línea de comandos para Windows que copia el contenido de
un directorio a otro, sobrescribiendo siempre y sin preguntar, optimizada para
velocidad y para árboles con millones de archivos.

**No es** un sustituto general de `robocopy`. Deliberadamente **no** implementa:

- Espejo (borrar en destino lo que ya no está en origen).
- Filtros por extensión, patrón, fecha o atributo.
- Copia de ACL de NTFS ni de flujos de datos alternativos (ADS).
- Reintentos automáticos ni reanudación de archivos parciales.

El alcance reducido es una decisión, no una carencia pendiente: la propuesta de
valor es *dos rutas y ya*, frente a los ~80 modificadores de `robocopy`.

### Restricciones de diseño

| Restricción | Consecuencia |
| --- | --- |
| Solo Windows | Todos los `.go` llevan `//go:build windows`; se usa `syscall` directo, no `x/sys` |
| Sin dependencias externas | `go.mod` no tiene `require`; por eso no existe `go.sum` y el CI desactiva la caché de módulos |
| Nunca borra en destino | Elimina toda una clase de fallos catastróficos |
| Un solo binario | Se distribuye como `.exe` autocontenido, sin instalador obligatorio |

---

## 2. Panorama del código

Unas 1.300 líneas de Go en 6 archivos, todos en `package main`.

| Archivo | Responsabilidad | Depende de |
| --- | --- | --- |
| [`main.go`](../main.go) | CLI, flags, orquestación, progreso, resumen | todos |
| [`args.go`](../args.go) | Saneamiento y validación de argumentos y rutas | `path.go` |
| [`copier.go`](../copier.go) | Motor: pools, recorrido, copia, estadísticas, `-u` | `queue.go`, `winapi.go`, `path.go` |
| [`queue.go`](../queue.go) | Cola de directorios y tipos de trabajo | — |
| [`winapi.go`](../winapi.go) | Enlaces directos a `kernel32.dll` | — |
| [`path.go`](../path.go) | Conversión a rutas extendidas `\\?\` | — |

Fuera del código Go:

| Archivo | Para qué |
| --- | --- |
| [`install.ps1`](../install.ps1) | Instalador/desinstalador; descarga, desbloquea y gestiona el `PATH` |
| [`bench/bench.ps1`](../bench/bench.ps1) | Benchmark reproducible contra `robocopy` |
| [`winget/manifests/`](../winget/manifests/) | Manifiestos para Windows Package Manager |
| [`.github/workflows/`](../.github/workflows/) | CI y publicación automática de releases |
| [`.gitattributes`](../.gitattributes) | Fija LF en los `.go` (ver [sección 13](#13-trampas-de-windows-aprendidas)) |

---

## 3. Arquitectura del motor

El flujo completo de una ejecución:

```
main()
  ├─ flag.Parse()
  ├─ validarNumeroDeArgumentos()      args.go
  ├─ sanitizeArg() x2                 args.go   -> rutas absolutas y limpias
  ├─ validarDirectorios()             args.go   -> comprobaciones semánticas
  ├─ extendedPath() x2                path.go   -> prefijo \\?\
  ├─ mkdirAll(destino)                main.go
  ├─ sistemaDeArchivos(destino)       winapi.go -> "NTFS" | "exFAT" | ...
  ├─ newCopier(...)                   copier.go
  ├─ startProgress()                  main.go   -> goroutine, ticker 250 ms
  ├─ copier.run(origen, destino)      copier.go <- aquí ocurre todo
  └─ resumen + os.Exit(0|1)
```

Dentro de `run` conviven **dos pools de workers sobre dos colas distintas**:

```
                    ┌───────────────────────────┐
   raíz ──push──▶   │  colaDirs  (sin límite)   │ ◀──push── subdirectorios
                    │  LIFO, mutex + sync.Cond  │              ▲
                    └────────────┬──────────────┘              │
                                 │ pop                         │
                          ┌──────▼───────┐                     │
                          │  4 walkers   │─────────────────────┘
                          │  walkDir()   │
                          └──────┬───────┘
                                 │ send (bloquea si está llena)
                    ┌────────────▼──────────────┐
                    │ chan archivoJob (4096)    │  <- el techo de memoria
                    └────────────┬──────────────┘
                                 │ range
                          ┌──────▼───────┐
                          │ -w copiers   │
                          │  copyOne()   │
                          └──────────────┘
```

La separación en dos pools es **la decisión estructural más importante del
proyecto**, y resuelve dos problemas a la vez (memoria y velocidad). Se explica
en las secciones [4](#4-modelo-de-concurrencia) y [5](#5-gestión-de-memoria).

---

## 4. Modelo de concurrencia

### Las dos colas

**`colaDirs`** ([queue.go](../queue.go)) es una cola con `sync.Mutex` +
`sync.Cond` y contador de pendientes:

- **LIFO** (`items[len-1]`): al sacar el último directorio insertado, el
  recorrido tiende a profundizar en una rama antes de saltar a otra, lo que
  mejora la localidad de los metadatos que NTFS ya tiene en caché.
- **Sin límite**: los directorios son dos órdenes de magnitud menos numerosos
  que los archivos (~200.000 frente a ~2.900.000 en un caso real).
- **Terminación por contador**: `push` incrementa `pending`, `done` lo
  decrementa; al llegar a cero se marca `closed` y se hace `Broadcast`. Así los
  walkers salen solos cuando el árbol está agotado, sin necesidad de que nadie
  sepa de antemano cuántos directorios hay.

**El canal de archivos** es un `chan archivoJob` con capacidad `-cola` (4.096).
Se eligió un canal y no otra cola con `Cond` porque aquí sí hace falta
**bloqueo al escribir**, que es justo lo que un canal con buffer ofrece de
serie.

### Por qué no puede haber bloqueo mutuo

Es la propiedad crítica del diseño, y conviene tenerla presente antes de tocar
nada:

> Los workers de copia **solo consumen** del canal de archivos y **nunca
> escriben** en él.

De ahí se sigue que siempre hay alguien capaz de drenar el canal, así que un
walker bloqueado en un envío acabará desbloqueándose. **Con un único pool esto
no se cumpliría**: todos los workers podrían quedar bloqueados intentando
encolar, sin que quedara nadie para vaciar la cola, y el programa se pararía
para siempre.

Verificado empíricamente en el escenario más hostil posible, `-cola 1 -w 1`
(capacidad 1, un solo copiador): termina correctamente.

### Secuencia de cierre

```go
recorrido.Wait()   // todos los walkers han salido -> no habrá más archivos
close(archivos)    // los copiadores terminan su range al drenar
copia.Wait()       // todo copiado
```

El orden importa: cerrar el canal antes de que los walkers hayan terminado
provocaría un *panic* por envío sobre canal cerrado.

### Estado compartido

- **Estadísticas**: `atomic.Int64` ([copier.go](../copier.go), `stats`). Sin
  mutex: son contadores independientes y nunca se leen en conjunto de forma
  transaccional.
- **Errores**: `errMu sync.Mutex` protege solo el contador de errores mostrados,
  para no entrelazar líneas en la salida.
- **`*carpeta`**: se comparte entre goroutines pero es **inmutable tras su
  creación**, así que no necesita sincronización.

### Elección de `walkersPorDefecto = 4`

Listar directorios es unas 300 veces más rápido que copiar (0,04 s frente a
12 s para 20.000 archivos). Cuatro walkers saturan de sobra la cola de archivos;
más solo añadirían contención sobre `colaDirs` sin aportar nada.

---

## 5. Gestión de memoria

### El problema

Sin freno, el recorrido —300× más rápido que la copia— vacía el árbol entero en
la cola antes de que se copie nada. El consumo pasa a ser proporcional al número
de archivos. Con 2,9 millones de archivos, medido y extrapolado, eso son cientos
de MB de RAM para nada.

### Las dos medidas

**a) Compartir el prefijo de ruta.** Antes cada entrada guardaba las dos rutas
absolutas completas. Ahora:

```go
type carpeta struct {          // una por directorio
    src string
    dst string
}

type archivoJob struct {       // una por archivo
    dir  *carpeta              // 8 bytes, compartido
    name string                // solo "index.js"
    size int64
}
```

Las rutas completas se componen al copiar con `join(j.dir.src, j.name)`. En un
árbol de 2,9 millones de archivos en 200.000 directorios, el texto de las rutas
se almacena 200.000 veces en lugar de 2,9 millones.

**b) Acotar la cola.** El canal tiene capacidad fija; cuando se llena, los
walkers se bloquean. Ese es el techo real.

### Resultado medido

Pico de *working set* del proceso, copiando el mismo árbol:

| Archivos | Antes | Después |
| --- | --- | --- |
| 20.000 | 15,5 MB | 15,7 MB |
| 300.000 | 22,7 MB | **16,0 MB** |
| Factor al multiplicar ×15 los archivos | ×1,5 | **×1,0** |

Los ~15 MB de base son el *runtime* de Go y el propio binario, no la cola.

---

## 6. La capa Win32

[`winapi.go`](../winapi.go) enlaza con `kernel32.dll` mediante
`syscall.NewLazyDLL`, sin `golang.org/x/sys/windows`, para mantener el proyecto
sin dependencias externas.

| Función | Uso |
| --- | --- |
| `CopyFileExW` | La copia en sí |
| `CreateDirectoryW` | Crear directorios; devuelve si **ya existía** |
| `GetFileAttributesW` / `SetFileAttributesW` | Limpiar *solo lectura*/oculto/sistema |
| `GetVolumeInformationW` | Detectar el sistema de archivos del destino |

### Por qué `CopyFileExW` y no `ReadFile`/`WriteFile`

Porque el kernel mueve los bytes de un handle a otro: **no pasan por espacio de
usuario ni por buffers de Go**. Windows aplica internamente doble buffering, E/S
asíncrona y preasignación del tamaño del destino para evitar fragmentación.
Además copia atributos y marcas de tiempo de forma nativa —esto último es
imprescindible para que `-u` funcione.

Se probó la alternativa manual y **empata** (ver [sección 12](#12-hipótesis-descartadas)).

### Los flags y su cascada de reintentos

```go
if size >= noBufferingThreshold {        // 32 MiB
    flags |= copyFileNoBuffering
}
flags |= copyFileAllowDecryptedDest
```

- **`COPY_FILE_NO_BUFFERING` (0x1000)** solo en archivos grandes: evita ensuciar
  la caché de Windows con datos de un solo uso. En archivos pequeños la caché sí
  ayuda, así que ahí no se activa. **Es la razón de que ganemos 1,65× a
  `robocopy` con archivos grandes.**
- **`COPY_FILE_ALLOW_DECRYPTED_DESTINATION` (0x8)**: permite copiar un archivo
  cifrado con EFS a un destino que no soporta cifrado, en vez de fallar.

`copyOne` implementa dos reintentos, en este orden:

1. Si falló con `NO_BUFFERING` y el error sugiere que no está soportado
   (`ERROR_INVALID_PARAMETER`, `ERROR_NOT_SUPPORTED`, `ERROR_INVALID_FUNCTION`
   —típico de recursos de red), se reintenta sin ese flag.
2. Si falló con `ERROR_ACCESS_DENIED`, se limpian los atributos del destino y se
   reintenta. Esto implementa el requisito de *reemplazar sin preguntar* frente a
   archivos de solo lectura.

### `createDirectory` devuelve `yaExistia`

```go
func createDirectory(path string) (yaExistia bool, err error)
```

Parece un detalle, pero es lo que hace que `-u` **no penalice la primera copia**:
un directorio recién creado está vacío por definición, así que no hace falta
listarlo para buscar archivos que no pueden existir.

---

## 7. Rutas: el problema de MAX_PATH

Las funciones Win32 con nombre de ruta están limitadas a 260 caracteres, salvo
que se use el prefijo `\\?\`, que además **desactiva toda normalización**: la
cadena llega al sistema de archivos tal cual.

[`path.go`](../path.go) resuelve la conversión:

```go
\\?\D:\ruta            // unidad local
\\?\UNC\servidor\recurso   // recurso de red: \\servidor\... -> \\?\UNC\servidor\...
```

**Consecuencia importante para quien toque el código:** bajo `\\?\` no se
admiten `/`, ni `.`, ni `..`, ni barras duplicadas. Por eso `sanitizeArg` aplica
`filepath.Abs` + `Clean` **antes** de añadir el prefijo, y por eso existe `join`
en lugar de `filepath.Join`:

```go
func join(dir, name string) string {   // concatena y punto
    if strings.HasSuffix(dir, `\`) { return dir + name }
    return dir + `\` + name
}
```

`filepath.Join` volvería a limpiar la ruta en cada llamada —trabajo inútil en el
punto más caliente del recorrido— y podría alterar el prefijo.

`displayPath` hace el camino inverso para que los mensajes al usuario no muestren
el `\\?\`.

---

## 8. Modo incremental (`-u`)

Salta los archivos cuyo **tamaño y fecha de modificación** ya coinciden en
destino.

### Cómo evita costar tiempo

- Los datos del **origen** ya vienen gratis: `ReadDir` los entrega desde el mismo
  `FindFirstFileW` que hace el recorrido.
- Los del **destino** se obtienen con **un `ReadDir` por carpeta**, no un `stat`
  por archivo (`leerDestino`). Una llamada al sistema por directorio.
- Si la carpeta acaba de crearse, **no se lista** (ver `yaExistia` arriba).

El mapa se indexa con `strings.ToLower(nombre)` porque el sistema de archivos de
Windows no distingue mayúsculas.

### La precisión de la comparación: un fallo real que hubo que corregir

La primera implementación aplicaba **siempre** una tolerancia de 2 segundos, con
el argumento de que FAT/exFAT redondean las fechas. Estaba mal, y la prueba lo
destapó de inmediato:

> Se modificó un archivo 1,1 s después de copiarlo, conservando el tamaño. `-u`
> lo dio por idéntico y **no lo copió**: el destino se quedó con la versión
> vieja. Pérdida de datos silenciosa.

La corrección: la comparación es **exacta** por defecto (NTFS guarda 100 ns), y
la tolerancia de 2 s se aplica **solo** si el destino es FAT/exFAT, detectado
con `GetVolumeInformationW` al arrancar.

```go
func (c *copier) sinCambios(destino entradaDestino, tamano int64, mod time.Time) bool {
    if destino.modificacion.IsZero() || destino.tamano != tamano {
        return false            // no existe o cambió de tamaño -> copiar
    }
    if c.tolerancia == 0 {
        return destino.modificacion.Equal(mod)   // NTFS: exacto
    }
    // FAT/exFAT: margen de 2 s
    ...
}
```

**Principio general del modo `-u`: ante cualquier duda, copiar.** El valor cero
de `entradaDestino` (archivo ausente) devuelve `false`; si `leerDestino` no puede
listar la carpeta devuelve `nil` y se copia todo. Copiar de más es inocuo; no
copiar cuando había que hacerlo, no.

### Limitación conocida

Un archivo que cambie conservando **tamaño y fecha exactos** no se detecta. Es
prácticamente imposible por accidente, pero está documentado en el README. La
alternativa —comparar contenido con un hash— exigiría leer ambos archivos
completos, más caro que copiar.

---

## 9. Validación de entrada

Toda en [`args.go`](../args.go), antes de tocar el disco. Merece la pena por una
razón concreta: es una herramienta que **sobrescribe sin preguntar**, así que un
error de interpretación de argumentos puede ser destructivo.

| Comprobación | Motivo |
| --- | --- |
| Exactamente 2 argumentos | — |
| Comillas residuales | Llegan así desde `.bat` o al pegar rutas del Explorador |
| Comilla suelta en medio | Diagnostica la trampa de la barra final (ver abajo) |
| Caracteres inválidos `< > " \| ? *`, `:` fuera de la unidad, control | Fallarían más tarde con un error críptico |
| Origen existe y es directorio | — |
| Destino no es un archivo | — |
| Origen ≠ destino | — |
| Destino no está dentro del origen | **Evita una copia recursiva infinita** |

La validación de caracteres es consciente de los prefijos: salta `\\?\UNC\`,
`\\?\`, `\\` y la letra de unidad antes de buscar caracteres prohibidos.

---

## 10. Manejo de errores

**Filosofía: un archivo que falla no aborta la copia.** En un árbol de millones
de archivos, abortar por uno bloqueado sería inaceptable.

- `reportErr` incrementa el contador y escribe en `stderr`.
- A partir de 50 errores deja de imprimir (salvo `-v`), para no ahogar la
  consola. Avisa de que lo está haciendo.
- El código de salida final es `1` si hubo algún error, `0` si no, `2` por uso
  incorrecto. Esto permite comprobar `$LASTEXITCODE` desde un script.

Los errores de Win32 llegan como `syscall.Errno`; se comparan por número
(`errorAccessDenied = 5`, etc.) porque el texto depende del idioma del sistema.

---

## 11. Rendimiento: metodología y resultados

### Metodología

[`bench/bench.ps1`](../bench/bench.ps1), publicado para que los resultados sean
reproducibles:

- Destino **borrado antes de cada corrida** (si no, se mide el modo incremental).
- **Mediana** de varias corridas, no la media: una corrida atípica no distorsiona.
- Corridas **alternadas** entre herramientas, para que la deriva térmica o el
  estado de la caché no beneficie sistemáticamente a una.
- Escenarios separados: muchos pequeños, pocos grandes, y mixto. **Agrupar todo
  en un solo número oculta que cada herramienta gana en un régimen distinto.**

### Resultados actuales (v0.0.3)

| Escenario | robocopy `/MT` | UltraCopySam |
| --- | --- | --- |
| 20.000 archivos pequeños (7,6 MB) | 10,70 s | **10,44 s** |
| 8 archivos grandes (2.000 MB) | 1,44 s | **0,87 s** |
| Mixto: 5.006 archivos (734 MB) | **2,90 s** | 3,17 s |
| Segunda pasada sin cambios | 0,03 s | 0,02 s |

### La lección de rendimiento del proyecto

La versión 0.0.2 era **21% más lenta** que `robocopy` con archivos pequeños. Se
probaron tres explicaciones plausibles y **todas resultaron falsas**. La causa
real apareció mientras se resolvía un problema de memoria: **un único pool hacía
recorrido y copia**, así que un worker ocupado listando no estaba copiando.

Separar los pools cerró la diferencia y, de paso, redujo el tiempo a la mitad en
árboles grandes (300.000 archivos: 127 s → 60 s).

> **Moraleja para quien optimice esto:** mide antes de reescribir. Las tres
> hipótesis "evidentes" eran incorrectas, y la causa real estaba en la
> arquitectura de concurrencia, no en la API ni en las estructuras de datos.

### Dónde está hoy el límite

Con archivos pequeños, el coste es **por archivo** y no por byte: crear cada
entrada en la MFT de NTFS y confirmar el *journal*. El recorrido de 20.000
archivos toma 0,04 s frente a ~10 s de copia. Ninguna optimización de nuestro
código cambia eso; el límite lo pone el sistema de archivos.

---

## 12. Hipótesis descartadas

Documentadas para que nadie las vuelva a explorar sin datos nuevos. Todas se
midieron con un prototipo aparte que copiaba los mismos 20.000 archivos.

| Hipótesis | Resultado | Conclusión |
| --- | --- | --- |
| `CopyFileExW` pesa demasiado en archivos pequeños; `ReadFile`/`WriteFile` manual sería más rápido | 10,27 s vs 10,32 s | **Empate.** No compensa reimplementar la copia |
| Las rutas extendidas `\\?\` cuestan más | 2% *más rápidas* | **Falsa.** Además son necesarias para MAX_PATH |
| La cola con `sync.Cond` es más lenta que un canal de Go | La cola ganó al canal | **Falsa** |
| Un análisis previo del árbol aceleraría la copia | El escaneo de 2,9 M archivos cuesta 118 s | **Contraproducente.** Rompería el solapamiento actual y añadiría minutos de espera para ahorrar segundos |

La última merece énfasis: **un índice previo no acelera nada**, porque el cuello
de botella no es saber qué copiar. Lo que sí ahorra tiempo de verdad es *no
copiar* (modo `-u`).

---

## 13. Trampas de Windows aprendidas

Problemas reales que costaron depuración. Si vas a tocar el proyecto, léelos.

### La barra invertida final escapa la comilla

```powershell
UltraCopySam "D:\origen\" "E:\destino\"
```

La `\` final **escapa la comilla de cierre**, y ambas rutas llegan fusionadas en
un único argumento. Es comportamiento del intérprete de comandos de Windows, no
del programa. `validarNumeroDeArgumentos` detecta el caso (un solo argumento que
contiene una comilla) y explica la causa en lugar de fallar de forma críptica.

### `gofmt` y los finales de línea CRLF

El CI falló al comprobar el formato. Causa: Git convierte a CRLF al hacer
*checkout* en Windows, y **`gofmt` considera que un archivo con CRLF no está
formateado**. En local no se veía porque los archivos estaban en LF.

Solución: [`.gitattributes`](../.gitattributes) con `*.go text eol=lf`.

### `SetEnvironmentVariable` corrompe el `PATH`

`[Environment]::SetEnvironmentVariable('PATH', ..., 'User')` **expande las
variables** del valor y cambia el tipo de registro de `REG_EXPAND_SZ` a
`REG_SZ`. A un usuario con `%USERPROFILE%\bin` en su `PATH` se lo dejaría
convertido en ruta fija para siempre.

`install.ps1` escribe la clave del registro directamente conservando el tipo, y
notifica el cambio con `WM_SETTINGCHANGE`.

### El *Mark of the Web* y SmartScreen

Todo archivo descargado lleva una marca que hace que Windows bloquee su
ejecución si no está firmado. `install.ps1` ejecuta `Unblock-File` tras instalar,
que es lo que evita el aviso sin necesidad de certificado.

### `winget validate` parsea todo el directorio

Intenta interpretar como YAML **todos** los archivos de la carpeta indicada. Un
`README.md` con un `:` a mitad de línea la hace fallar. Por eso los manifiestos
viven en `winget/manifests/` y la documentación se queda fuera.

### `$LASTEXITCODE` y los pipelines de PowerShell

`& $exe | Select-Object -First 1` **corta el pipeline** antes de que
`$LASTEXITCODE` llegue a asignarse. Hay que capturar toda la salida y luego leer
el código. Afectaba a la comprobación final de `install.ps1`.

### Las opciones van antes de las rutas

El paquete `flag` de Go deja de interpretar opciones en cuanto encuentra el
primer argumento posicional. `UltraCopySam "D:\a" "E:\b" -u` **no** activa `-u`;
el programa lo detecta y avisa.

---

## 14. Distribución y CI

### Flujo de publicación

```
git tag v0.0.4 && git push origin v0.0.4
        │
        ▼
.github/workflows/release.yml
   ├─ compila en un runner de Windows
   ├─ deriva el nombre: v0.0.4 -> UltraCopySamV004.exe
   ├─ calcula el SHA256
   └─ gh release create con notas que incluyen el hash y el commit
```

Que la compilación sea pública y trazable importa especialmente en un `.exe` sin
firmar: cualquiera puede comprobar de qué commit salió el binario que descarga.

### CI

[`ci.yml`](../.github/workflows/ci.yml) ejecuta `go vet`, comprueba `gofmt` y
corre una **prueba de humo que verifica el contenido copiado**, no solo que el
programa no falle: copia un árbol, comprueba los bytes en destino, valida que la
segunda pasada con `-u` no copia nada, y que un archivo modificado **que conserva
el tamaño** sí se copia (la regresión de la [sección 8](#8-modo-incremental--u)).

### Manual de versión nueva

1. `go build` y actualizar el SHA256 en `README.md`, `README.es.md` y los
   manifiestos de winget.
2. Etiquetar y empujar; el workflow publica el release.
3. **No sobrescribir un release ya publicado**: quien lo descargó tendría un
   archivo distinto con el mismo número de versión y un hash que ya no coincide.

---

## 15. Limitaciones y deuda técnica

### Limitaciones asumidas

Ver [sección 1](#1-qué-es-y-qué-no-es). Son alcance, no deuda.

### Deuda técnica real

| Asunto | Estado |
| --- | --- |
| **Sin pruebas unitarias en Go** | Toda la verificación es manual o vía la prueba de humo del CI. `sinCambios`, `esFAT`, `extendedPath`, `join` y `sanitizeArg` son funciones puras, ideales para tests de tabla. **Es la deuda más importante.** |
| **Sin progreso intra-archivo** | El contador suma al completar cada archivo. Con un solo archivo de 50 GB, el avance se queda en `0.00 MB` hasta el final. `CopyFileExW` admite un callback de progreso; se descartó por su coste por bloque, pero podría activarse solo por encima de cierto tamaño |
| **Sin porcentaje ni ETA** | Requeriría recorrer el árbol antes de copiar. Decisión consciente (ver [sección 12](#12-hipótesis-descartadas)) |
| **La diferencia del 9% en el escenario mixto** | Sin causa identificada. Requeriría *profiling* con `pprof` |
| **`main.go` mezcla responsabilidades** | CLI, progreso y formateo conviven. A este tamaño es manejable, pero si crece conviene separar la presentación |
| **Mensajes solo en español** | El README es bilingüe, pero los mensajes del programa no. Para un proyecto orientado a la comunidad internacional es una barrera |

---

## 16. Cómo extenderlo

### Antes de tocar el motor

1. **Ejecuta el benchmark antes y después.** `bench/bench.ps1` con los tres
   escenarios. Una mejora en archivos grandes puede ser una regresión en
   pequeños.
2. **Mide también la memoria**, no solo el tiempo: muestrea `WorkingSet64` del
   proceso durante una copia de al menos 300.000 archivos.
3. **Prueba el bloqueo mutuo** con `-cola 1 -w 1`. Cualquier cambio en el
   flujo de encolado puede introducir un interbloqueo que solo aparece bajo
   presión.
4. **Repite la batería de `-u`**, en particular el caso del archivo modificado
   que conserva el tamaño.

### Cambios de bajo riesgo

- **Filtros por patrón**: se aplicarían en `walkDir`, antes de encolar.
- **Modo simulación (`--dry-run`)**: recorrer sin copiar; encaja de forma natural
  porque el recorrido ya está separado de la copia.
- **Mensajes en inglés**: extraer las cadenas a un mapa y elegir por
  `GetUserDefaultUILanguage`.

### Cambios de riesgo medio

- **Modo espejo (`--mirror`)**: hay que borrar en destino lo que no está en
  origen. `leerDestino` ya da la lista de la carpeta; lo delicado es que **es la
  única funcionalidad capaz de destruir datos del usuario**, así que debería
  exigir confirmación explícita o un flag adicional.
- **Progreso intra-archivo**: pasar un `LPPROGRESS_ROUTINE` a `CopyFileExW`.
  Mide el coste: el callback se invoca por bloque.
- **Copiar ACL**: requiere `GetNamedSecurityInfo`/`SetNamedSecurityInfo`, y
  privilegios que el proceso puede no tener.

### Cambios que ya se descartaron

No los reintentes sin datos nuevos: ver [sección 12](#12-hipótesis-descartadas).

### Compilar y verificar

```powershell
go build -trimpath -ldflags "-s -w" -o UltraCopySam.exe .
go vet ./...
gofmt -l .        # debe no imprimir nada
```

El CI ejecuta exactamente eso, más la prueba de humo. Si `gofmt` se queja de
archivos que tú no tocaste, comprueba tus finales de línea (ver
[sección 13](#13-trampas-de-windows-aprendidas)).
