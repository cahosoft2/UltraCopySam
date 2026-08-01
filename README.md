# UltraCopySam

Utilitario de línea de comandos para **Windows** que copia un directorio
completo —archivos y subcarpetas— a otro destino, **reemplazando lo que exista
sin preguntar** y con el máximo rendimiento posible.

Está escrito en Go pero no usa la biblioteca estándar para copiar: llama
directamente a la API nativa de Win32, de modo que **los bytes los mueve el
kernel de Windows** y nunca pasan por buffers del programa.

```powershell
UltraCopySam.exe "D:\proyectos" "E:\backup\proyectos"
```

```text
1284 archivos | 3.41 GB | 812.44 MB/s
```

---

## Índice

- [Descarga](#descarga)
- [Windows SmartScreen y antivirus](#windows-smartscreen-y-antivirus)
- [Características](#características)
- [Instalación](#instalación)
- [Uso](#uso)
- [Opciones](#opciones)
- [Ejemplos](#ejemplos)
- [Comillas dobles y rutas con espacios](#comillas-dobles-y-rutas-con-espacios)
- [Discos externos USB](#discos-externos-usb)
- [Qué muestra durante la copia](#qué-muestra-durante-la-copia)
- [Comportamiento](#comportamiento)
- [Validaciones](#validaciones)
- [Cómo consigue la velocidad](#cómo-consigue-la-velocidad)
- [Elegir el número de workers](#elegir-el-número-de-workers)
- [Limitaciones](#limitaciones)
- [Compilación](#compilación)
- [Estructura del código](#estructura-del-código)
- [Licencia](#licencia)

---

## Descarga

Descarga la última versión desde la página de
**[Releases](https://github.com/cahosoft2/UltraCopySam/releases)**, o
directamente:

**[⬇ UltraCopySamV001.exe](https://github.com/cahosoft2/UltraCopySam/releases/download/v0.0.1/UltraCopySamV001.exe)**
— Windows 64 bits · 1,91 MB · sin instalador ni dependencias

Verifica la descarga comparando su huella SHA256:

```powershell
Get-FileHash UltraCopySamV001.exe -Algorithm SHA256
```

```text
1FF856EBE938BC3FD6F49B72FD14F8BDF9130FAAFE4CE45F355A76EC0C114FC1
```

> [!NOTE]
> Windows mostrará la advertencia *"Windows protegió su PC"* la primera vez que
> ejecutes el archivo descargado. Es normal y tiene solución en un solo comando:
> ver [Windows SmartScreen y antivirus](#windows-smartscreen-y-antivirus).

---

## Windows SmartScreen y antivirus

Al ejecutar el `.exe` recién descargado, Windows muestra una pantalla azul con
el mensaje **"Windows protegió su PC"** y solo ofrece el botón *No ejecutar*.

**Esto no significa que el programa sea peligroso.** Ocurre porque el
ejecutable no está firmado con un certificado de código, que cuesta entre 300 y
600 USD al año. Windows marca todo binario descargado de internet y sin firmar,
sin importar lo que haga.

Tienes tres formas de resolverlo, de la más rápida a la más segura.

### Opción 1 — Desbloquear el archivo (recomendada)

Al descargar cualquier archivo, Windows le añade una marca invisible llamada
*Mark of the Web* que indica "esto vino de internet". Basta con quitarla:

```powershell
Unblock-File .\UltraCopySamV001.exe
```

A partir de ahí el programa se ejecuta con normalidad, sin más advertencias.

También puedes hacerlo con el ratón: clic derecho sobre el archivo →
**Propiedades** → marcar la casilla **Desbloquear** al final de la pestaña
*General* → *Aceptar*.

### Opción 2 — Ejecutarlo de todas formas

En la propia pantalla de advertencia, pulsa **Más información** y luego el
botón **Ejecutar de todas formas** que aparece debajo. Windows recordará la
decisión para ese archivo.

### Opción 3 — Compilarlo tú mismo (la más segura)

Si prefieres no confiar en un ejecutable descargado —una postura razonable
tratándose de una herramienta que sobrescribe archivos—, compílalo desde el
código fuente. Un binario compilado en tu propia máquina **nunca dispara
SmartScreen**, porque no procede de internet:

```powershell
git clone https://github.com/cahosoft2/UltraCopySam.git
cd UltraCopySam
go build -trimpath -ldflags "-s -w" -o UltraCopySam.exe .
```

Requiere [Go](https://go.dev/dl/) instalado. Todo el código fuente son unas
1200 líneas repartidas en 6 archivos, auditables en unos minutos.

### Sobre los falsos positivos de antivirus

Los binarios de Go sin firmar generan falsos positivos con cierta frecuencia,
porque el compilador enlaza todo estáticamente en un único archivo y varios
motores heurísticos consideran ese patrón sospechoso.

Si tu antivirus bloquea el archivo, verifica primero su huella SHA256 (ver
[Descarga](#descarga)): si coincide con la publicada, el archivo es
exactamente el que se compiló y no ha sido alterado. Después, añade una
exclusión o utiliza la Opción 3.

### ¿Por qué no está firmado?

Firmar un ejecutable para Windows exige un certificado de código de una
autoridad reconocida, con verificación de identidad y renovación anual. Para
una utilidad gratuita de código abierto el coste no se justifica.

Las alternativas que existen, por si el proyecto crece:

| Vía | Coste aproximado | ¿Elimina SmartScreen? |
| --- | --- | --- |
| [Azure Trusted Signing](https://azure.microsoft.com/products/trusted-signing) | ~10 USD/mes | Sí, de inmediato |
| Certificado EV (DigiCert, Sectigo…) | 300-600 USD/año | Sí, de inmediato |
| [SignPath Foundation](https://signpath.org/) (gratis para OSS) | Gratis | Progresivamente, al acumular reputación |
| Publicar en Microsoft Store | ~19 USD, pago único | Sí (las apps de la Store no pasan por SmartScreen) |

---

## Características

- **Copia a bajo nivel** con `CopyFileExW`: el kernel transfiere los datos de un
  handle a otro sin pasar por espacio de usuario.
- **Paralelismo real**: un pool de workers recorre el árbol y copia al mismo
  tiempo, sin esperar a terminar de listar para empezar a copiar.
- **Reemplaza siempre**, sin confirmaciones. Incluso si el archivo de destino
  está marcado como *solo lectura*, *oculto* o *sistema*.
- **Sin límite de `MAX_PATH`**: soporta rutas de más de 260 caracteres mediante
  el prefijo extendido `\\?\`.
- **Tolerante a fallos**: un archivo bloqueado o sin permisos no aborta la
  copia; se reporta y el resto continúa.
- **Validación previa** de argumentos y rutas, con mensajes explicativos en
  español.
- **Un único ejecutable** sin dependencias: no requiere Go instalado, ni
  runtime, ni DLLs adicionales.

---

## Instalación

Descarga `UltraCopySam.exe` o compílalo (ver [Compilación](#compilación)) y
colócalo donde prefieras.

Para poder invocarlo como `UltraCopySam` desde cualquier carpeta, añade su
directorio al `PATH`:

```powershell
# Solo para la sesión actual
$env:PATH += ";D:\herramientas\UltraCopySam"

# Permanente, para el usuario actual
[Environment]::SetEnvironmentVariable(
    "PATH",
    [Environment]::GetEnvironmentVariable("PATH", "User") + ";D:\herramientas\UltraCopySam",
    "User")
```

Sin añadirlo al `PATH` hay que invocarlo por su ruta completa, o con `.\` si
estás situado en su carpeta (en PowerShell el `.\` es obligatorio):

```powershell
.\UltraCopySam.exe "D:\origen" "E:\destino"
```

**Requisitos:** Windows de 64 bits. No requiere permisos de administrador,
salvo que las carpetas involucradas los exijan.

---

## Uso

```text
UltraCopySam [opciones] "<directorio-origen>" "<directorio-destino>"
```

Se copia el **contenido** del origen dentro del destino, no la carpeta origen
en sí:

```text
UltraCopySam "D:\dev\old" "E:\backup"

D:\dev\old\proyecto\x.txt   ->   E:\backup\proyecto\x.txt
```

Si quieres conservar el nombre de la carpeta, inclúyelo en el destino:

```text
UltraCopySam "D:\dev\old" "E:\backup\old"

D:\dev\old\proyecto\x.txt   ->   E:\backup\old\proyecto\x.txt
```

**Códigos de salida:**

| Código | Significado |
| --- | --- |
| `0` | Todo se copió sin errores |
| `1` | La copia terminó, pero algún archivo falló |
| `2` | Uso incorrecto: faltan argumentos o alguna ruta no es válida |

---

## Opciones

| Opción | Descripción |
| --- | --- |
| `-w N` | Número de copias en paralelo. Por defecto, el doble de núcleos de CPU. |
| `-v` | Lista cada archivo copiado en lugar de mostrar la línea de progreso. |
| `-q` | Modo silencioso: sin progreso ni resumen (los errores sí se muestran). |
| `-L` | Sigue enlaces simbólicos y *junctions*. Por defecto se omiten. |

---

## Ejemplos

### Copia básica

```powershell
UltraCopySam "D:\proyectos" "E:\backup\proyectos"
```

### Rutas con espacios

Siempre entre comillas dobles.

```powershell
UltraCopySam "D:\Mis Documentos\Contabilidad" "E:\Copia de seguridad\Contabilidad"
```

### A un disco externo USB

Bajando el paralelismo, ver [Discos externos USB](#discos-externos-usb).

```powershell
UltraCopySam -w 4 "D:\dev\old" "E:\pc_caho\backup"
```

### Desde un recurso de red compartido

```powershell
UltraCopySam "\\servidor\compartido\datos" "D:\local\datos"
```

### Ver el detalle de cada archivo copiado

```powershell
UltraCopySam -v "D:\origen" "E:\destino"
```

### Modo silencioso, guardando solo los errores

```powershell
UltraCopySam -q "D:\origen" "E:\destino" 2> errores.log
```

### Guardar el resumen final en un log

El progreso va a *stderr* y el resumen a *stdout*, así que pueden separarse:

```powershell
UltraCopySam "D:\origen" "E:\destino" > resumen.log
```

### Comprobar si hubo errores desde un script

```powershell
UltraCopySam "D:\origen" "E:\destino"
if ($LASTEXITCODE -ne 0) {
    Write-Host "La copia terminó con errores" -ForegroundColor Red
}
```

### Backup diario con carpeta fechada

```powershell
$fecha = Get-Date -Format "yyyy-MM-dd"
UltraCopySam -w 4 "D:\dev" "E:\backups\$fecha\dev"
```

---

## Comillas dobles y rutas con espacios

Encierra siempre las rutas entre **comillas dobles**. Es lo que permite que una
ruta con espacios llegue como un único argumento:

```powershell
UltraCopySam "D:\Mis Datos\origen" "E:\Copia de seguridad\destino"
```

> [!WARNING]
> **No dejes una barra invertida final dentro de las comillas.**
>
> En Windows, `"D:\origen\"` hace que la `\` **escape a la comilla de cierre**,
> con lo que las dos rutas se fusionan en un solo argumento y el comando falla.
> Es una trampa clásica de la línea de comandos de Windows, no un problema de
> esta herramienta.
>
> ```text
> ✗  UltraCopySam "D:\origen\" "E:\destino\"     <- mal
> ✓  UltraCopySam "D:\origen"  "E:\destino"      <- bien
> ✓  UltraCopySam "D:\origen\\" "E:\destino\\"   <- también válido
> ```
>
> `UltraCopySam` detecta este caso concreto y explica qué ocurrió, en vez de
> fallar con un mensaje incomprensible.

También se aceptan **rutas relativas**, que se resuelven contra el directorio
actual:

```powershell
cd D:\dev
UltraCopySam ".\old" "E:\backup\old"
```

---

## Discos externos USB

Copiar a un disco USB es el escenario donde más importa la configuración. Estas
son las recomendaciones, ordenadas por el impacto real que tienen.

### 1. Activa la caché de escritura del disco (el mayor impacto)

Windows configura los discos extraíbles como *"Optimizar para extracción
rápida"*, lo que **desactiva la caché de escritura**: cada escritura viaja al
disco de inmediato. Cambiarlo suele mejorar el rendimiento más que cualquier
ajuste de la herramienta.

```text
Administrador de dispositivos
  └─ Unidades de disco
       └─ (tu disco USB) → Propiedades
            └─ pestaña "Directivas" → marcar "Mejor rendimiento"
```

> [!CAUTION]
> Con esta opción activada **debes** usar siempre *"Quitar hardware de forma
> segura"* antes de desconectar el disco. Si lo desenchufas en caliente puedes
> perder datos que aún estaban en caché.

### 2. Baja el número de workers

El valor por defecto (el doble de núcleos, típicamente 16–32) está pensado para
discos internos NVMe y es **excesivo** para USB:

```powershell
UltraCopySam -w 4 "D:\origen" "E:\destino"     # disco USB mecánico (HDD)
UltraCopySam -w 8 "D:\origen" "E:\destino"     # SSD en carcasa USB
```

En un disco mecánico solo hay un cabezal físico: demasiadas escrituras
simultáneas lo obligan a saltar de una zona a otra del plato y el rendimiento
**cae** en vez de subir. Si oyes al disco castañetear de forma continua, baja a
`-w 2`.

En un SSD externo el límite lo pone el bus USB, así que pasar de 8 no aporta
nada.

Si no sabes qué hay dentro de la carcasa, empieza con `-w 4`: en un SSD pierdes
un poco de velocidad, pero 32 workers en un mecánico sí hunden el rendimiento.

### 3. Verifica el puerto y el cable

Un puerto USB 2.0 te limita a unos **35 MB/s reales** y ningún ajuste lo
soluciona. Busca puertos USB 3.0 o superior: suelen tener el interior azul o
llevar la marca `SS` (*SuperSpeed*).

| Estándar | Velocidad real aproximada |
| --- | --- |
| USB 2.0 | 35 MB/s |
| USB 3.0 / 3.1 Gen1 | 400–450 MB/s |
| USB 3.2 Gen2 | ~1 GB/s |

Un cable de mala calidad o demasiado largo puede hacer que el disco negocie a
una velocidad inferior a la que soporta.

### 4. Formatea el destino en NTFS

| Sistema de archivos | Recomendación |
| --- | --- |
| **NTFS** | ✅ Recomendado. Sin límite práctico de tamaño, admite rutas largas `\\?\` y conserva los atributos de archivo |
| exFAT | ⚠️ Funciona, pero se degrada con muchos archivos pequeños |
| FAT32 | ❌ **No admite archivos de más de 4 GB**: fallarán, aunque el resto de la copia continúa |

### 5. Ten expectativas realistas con archivos pequeños

Si copias carpetas de desarrollo (`node_modules`, `.git`, `vendor`), el tiempo
**no** se va en mover bytes: se va en crear cada archivo —reservar su entrada en
la MFT de NTFS, confirmar el *journal*, cerrar el handle—. Son milisegundos por
archivo que el bus USB no puede paralelizar más allá de lo que ya hacen los
workers.

| Escenario | Velocidad típica en USB 3.0 |
| --- | --- |
| Archivos grandes (vídeo, ISO, backups) | 100–110 MB/s en HDD; 400+ MB/s en SSD |
| Miles de archivos pequeños | 5–15 MB/s |

Esa caída es del hardware y del sistema de archivos, no del programa. Si la
carpeta tiene mucho contenido regenerable (`node_modules`, `bin`, `obj`,
`target`), copiarla comprimida en un solo archivo suele ser más rápido que
copiar los archivos sueltos.

### 6. Evita que el disco se duerma

Windows puede suspender un disco USB durante una copia larga y provocar errores
intermitentes. En **Opciones de energía → Cambiar configuración avanzada →
Disco duro**, pon *"Apagar el disco duro tras"* en `0` (nunca) mientras dure la
copia.

---

## Qué muestra durante la copia

Mientras copia, `UltraCopySam` actualiza **una sola línea** cada 250 ms:

```text
1284 archivos | 3.41 GB | 812.44 MB/s
```

Al terminar imprime el resumen:

```text
5327 archivos, 214 directorios, 12.83 GB en 16.204s (811.55 MB/s)
```

y, si corresponde, cuántas entradas se omitieron y cuántos errores hubo.

Detalles útiles:

- El **progreso va a `stderr`** y el **resumen a `stdout`**, de modo que puedes
  redirigir el resultado a un archivo sin que se llene de líneas parpadeantes.
- Con `-v` se desactiva la línea de progreso y se imprime cada archivo copiado.
- Con `-q` no se muestra nada salvo los errores.
- En copias muy cortas (menos de 250 ms) solo verás el resumen final.
- No se muestra porcentaje ni tiempo restante: calcularlos exigiría recorrer
  todo el árbol antes de empezar a copiar, lo que retrasa el arranque y
  contradice el objetivo de máxima velocidad.

---

## Comportamiento

- El **destino se crea** si no existe, incluidos todos los directorios
  intermedios.
- Los archivos existentes en destino **se sobrescriben siempre**, sin
  preguntar. Si el archivo de destino tiene el atributo *solo lectura*,
  *oculto* o *sistema*, se limpia el atributo y se reintenta.
- **No es un espejo**: lo que ya existía en destino y no está en origen **se
  conserva**. No se borra nada.
- Los **enlaces simbólicos y junctions se omiten** por defecto, para evitar
  ciclos infinitos y copias duplicadas. Se informan en el resumen final. Con
  `-L` se siguen y se copia su contenido.
- Un **error en un archivo no aborta la copia**: se reporta por `stderr` y el
  recorrido continúa. Al final se indica cuántos fallaron y el código de salida
  es `1`.
- Se copian los **atributos** del archivo (solo lectura, oculto, etc.) y sus
  marcas de tiempo, porque `CopyFileExW` lo hace de forma nativa.
- **No** se copian los permisos NTFS (ACL) ni los flujos de datos alternativos
  (ADS).

---

## Validaciones

Nada se copia hasta que todas estas comprobaciones pasan:

| Situación | Resultado |
| --- | --- |
| Número de argumentos distinto de 2 | Error indicando cuál falta o cuántos sobran |
| Argumentos fusionados por una barra final | Error explicando la causa y la solución |
| Comillas residuales alrededor de la ruta | Se eliminan automáticamente |
| Ruta vacía o compuesta solo de espacios | Error |
| Caracteres no válidos en Windows (`< > " \| ? *`, `:` fuera de la letra de unidad, caracteres de control) | Error señalando el carácter concreto |
| Origen inexistente | Error |
| Origen es un archivo, no un directorio | Error |
| Destino existe pero es un archivo | Error |
| Origen y destino son el mismo directorio | Error |
| Destino situado dentro del origen | Error: la copia sería recursiva |

La validación de caracteres respeta la letra de unidad (`D:`), los recursos de
red (`\\servidor\share`) y el prefijo extendido (`\\?\`).

---

## Cómo consigue la velocidad

- **`CopyFileExW`**, la API nativa de Win32. El kernel mueve los bytes de un
  handle a otro: los datos **no pasan por el espacio de usuario** ni por
  buffers de Go. Windows aplica internamente doble buffering, E/S asíncrona y
  preasignación del tamaño del destino para evitar fragmentación.
- **`COPY_FILE_NO_BUFFERING`** en archivos de 32 MiB o más: evita ensuciar la
  caché de archivos de Windows con datos de un solo uso. En archivos pequeños la
  caché sí ayuda, así que ahí no se activa. Si el volumen no soporta el flag
  —algunos recursos de red—, se reintenta automáticamente sin él.
- **Pool de workers sobre una cola de trabajo compartida**: recorrer el árbol y
  copiar ocurren simultáneamente, sin una barrera entre "listar" y "copiar". Se
  usa una cola explícita en lugar de una goroutine por entrada, para que la
  memoria no crezca en árboles de millones de archivos.
- **Rutas extendidas `\\?\`**: además de eliminar el límite de 260 caracteres,
  evitan el coste de normalización de rutas que Win32 aplica en cada llamada.
- **Listado sin ordenar** (`ReadDir(-1)`) y tamaños obtenidos del propio
  `FindFirstFileW`, sin una llamada `stat` adicional por archivo.
- La cola se recorre en **LIFO**, lo que mejora la localidad al descender por el
  árbol de directorios.

---

## Elegir el número de workers

| Escenario | Recomendación |
| --- | --- |
| SSD NVMe interno → SSD NVMe interno | Valor por defecto (`NumCPU * 2`) |
| SSD interno → SSD externo USB | `-w 8` |
| Cualquier origen → **HDD externo USB** | `-w 4` (o `-w 2` si el disco castañetea) |
| HDD mecánico → HDD mecánico | `-w 2` |
| Recurso de red (SMB) | `-w 8` a `-w 16`: la latencia de red se compensa con paralelismo |
| Mismo disco físico (origen y destino) | `-w 2`: la cabeza compite consigo misma |

Regla general: **más workers solo ayudan si el dispositivo puede atender varias
operaciones a la vez.** Los discos mecánicos no pueden; los SSD sí.

---

## Limitaciones

- **Solo Windows.** Depende de la API de Win32; no compila en Linux ni macOS.
- **No es un espejo.** No borra en destino lo que ya no está en origen.
- **No reanuda archivos parciales.** Si una copia se interrumpe, el archivo que
  estaba en curso se vuelve a copiar completo en la siguiente pasada.
- **No compara antes de copiar.** Cada ejecución reescribe todos los archivos,
  aunque sean idénticos a los del destino.
- **No copia ACL ni flujos alternativos (ADS).** Si necesitas preservar
  permisos NTFS, usa `robocopy /COPYALL`.
- **No admite filtros** por extensión, patrón ni fecha.

---

## Compilación

Requiere Go 1.21 o superior.

```powershell
go build -trimpath -ldflags "-s -w" -o UltraCopySam.exe .
```

- `-trimpath` elimina las rutas de compilación del binario.
- `-ldflags "-s -w"` quita la tabla de símbolos y la información de depuración,
  reduciendo el tamaño del ejecutable.

El resultado es un binario autocontenido de menos de 2 MB, sin dependencias
externas más allá de `kernel32.dll`, que forma parte de Windows.

Verificación estática:

```powershell
go vet ./...
```

---

## Estructura del código

| Archivo | Contenido |
| --- | --- |
| `main.go` | Interfaz de línea de comandos, línea de progreso y resumen final |
| `args.go` | Saneamiento de argumentos y validación de rutas |
| `copier.go` | Motor de copia: pool de workers, recorrido y estadísticas |
| `queue.go` | Cola de trabajo concurrente con contador de pendientes |
| `winapi.go` | Enlaces directos a `kernel32.dll` (`CopyFileExW`, `CreateDirectoryW`, atributos) |
| `path.go` | Conversión a rutas extendidas `\\?\` |

Todos los archivos llevan la etiqueta de compilación `//go:build windows`.

---

## Licencia

**[BSD Zero Clause License (0BSD)](LICENSE)** — la licencia más permisiva que
existe.

Puedes usar, copiar, modificar, redistribuir y vender este software, con o sin
fines comerciales, sin ninguna condición: **ni siquiera se exige conservar el
aviso de copyright ni mencionar al autor**. Es equivalente a ponerlo en el
dominio público, pero redactado como licencia para que sea válida en cualquier
jurisdicción.

Lo único que hace la licencia es lo que toda licencia de software libre debe
hacer: dejar claro que el software se entrega **tal cual**, sin garantías, y
que el autor no responde por los daños que su uso pueda ocasionar.
