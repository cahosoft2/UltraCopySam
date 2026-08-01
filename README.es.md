# UltraCopySam

[![CI](https://github.com/cahosoft2/UltraCopySam/actions/workflows/ci.yml/badge.svg)](https://github.com/cahosoft2/UltraCopySam/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/cahosoft2/UltraCopySam)](https://github.com/cahosoft2/UltraCopySam/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/cahosoft2/UltraCopySam/total)](https://github.com/cahosoft2/UltraCopySam/releases)
[![License: 0BSD](https://img.shields.io/badge/license-0BSD-blue.svg)](LICENSE)
[![Platform: Windows](https://img.shields.io/badge/platform-Windows%20x64-0078D6)](https://github.com/cahosoft2/UltraCopySam/releases/latest)

*[English version](README.md)*

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
1284 archivos | 3491.84 MB | 812.44 MB/s
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
- [Copias incrementales](#copias-incrementales)
- [Comparativa con robocopy](#comparativa-con-robocopy)
- [¿Usar esto en lugar de robocopy?](#usar-esto-en-lugar-de-robocopy)
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

**[⬇ UltraCopySamV002.exe](https://github.com/cahosoft2/UltraCopySam/releases/download/v0.0.2/UltraCopySamV002.exe)**
— Windows 64 bits · 1,91 MB · sin instalador ni dependencias

Verifica la descarga comparando su huella SHA256:

```powershell
Get-FileHash UltraCopySamV002.exe -Algorithm SHA256
```

```text
E22891936853ECBB3EB244669AFAA16232BFBE78B3E8247764E7630E05D233B6
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
Unblock-File .\UltraCopySamV002.exe
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

### Instalación automática (recomendada)

Ejecuta esto en PowerShell y listo:

```powershell
irm https://raw.githubusercontent.com/cahosoft2/UltraCopySam/main/install.ps1 | iex
```

El instalador se encarga de todo:

1. Descarga la última versión desde GitHub Releases.
2. Muestra su huella SHA256 para que puedas verificarla.
3. La instala en `%LOCALAPPDATA%\Programs\UltraCopySam`.
4. La **desbloquea** para que SmartScreen no interfiera (`Unblock-File`).
5. La agrega al `PATH` del usuario.
6. Comprueba que el ejecutable responde.

**No requiere permisos de administrador**: todo queda en tu perfil de usuario.
Abre una terminal nueva al terminar para que el `PATH` surta efecto.

> [!TIP]
> Si prefieres revisar el script antes de ejecutarlo —buena costumbre con
> cualquier instalador de internet—, ábrelo primero:
> [install.ps1](install.ps1). También puedes descargarlo y ejecutarlo aparte:
>
> ```powershell
> irm https://raw.githubusercontent.com/cahosoft2/UltraCopySam/main/install.ps1 -OutFile install.ps1
> notepad install.ps1
> .\install.ps1
> ```

#### Opciones del instalador

| Parámetro | Descripción |
| --- | --- |
| `-Version v0.0.2` | Instala una versión concreta en lugar de la última |
| `-InstallDir "D:\herramientas"` | Cambia la carpeta de instalación |
| `-FromFile ".\UltraCopySam.exe"` | Instala desde un archivo local, sin descargar |
| `-NoPath` | No modifica el `PATH` |
| `-Uninstall` | Desinstala: borra la carpeta y limpia el `PATH` |

```powershell
# Instalar en otra carpeta, sin tocar el PATH
.\install.ps1 -InstallDir "D:\herramientas\UltraCopySam" -NoPath

# Desinstalar
.\install.ps1 -Uninstall
```

> [!NOTE]
> Si PowerShell bloquea la ejecución del script por su directiva de
> ejecución, ejecútalo así (afecta solo a ese proceso, no cambia la
> configuración de tu sistema):
>
> ```powershell
> powershell -ExecutionPolicy Bypass -File .\install.ps1
> ```

### Instalación manual

Descarga `UltraCopySam.exe` de [Releases](https://github.com/cahosoft2/UltraCopySam/releases)
o compílalo (ver [Compilación](#compilación)), desbloquéalo y colócalo donde
prefieras:

```powershell
Unblock-File .\UltraCopySamV002.exe
```

Para poder invocarlo como `UltraCopySam` desde cualquier carpeta, añade su
directorio al `PATH`:

```powershell
# Solo para la sesión actual
$env:PATH += ";D:\herramientas\UltraCopySam"
```

Para hacerlo permanente, lo más seguro es usar el instalador con `-FromFile`,
que escribe el `PATH` sin alterar las variables que contenga:

```powershell
.\install.ps1 -FromFile ".\UltraCopySamV002.exe" -InstallDir "D:\herramientas\UltraCopySam"
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
| `-u` | No copia los archivos cuyo tamaño y fecha ya coincidan en destino. Ver [Copias incrementales](#copias-incrementales). |
| `-w N` | Número de copias en paralelo. Por defecto, el doble de núcleos de CPU. |
| `-v` | Lista cada archivo copiado en lugar de mostrar la línea de progreso. |
| `-q` | Modo silencioso: sin progreso ni resumen (los errores sí se muestran). |
| `-L` | Sigue enlaces simbólicos y *junctions*. Por defecto se omiten. |

> [!IMPORTANT]
> Las opciones van **antes** de las rutas. `UltraCopySam -u "D:\a" "E:\b"`
> funciona; `UltraCopySam "D:\a" "E:\b" -u` no, y el programa avisa de ello.

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

## Copias incrementales

Con `-u`, antes de copiar cada archivo se comprueba si en el destino ya hay uno
con **el mismo tamaño y la misma fecha de modificación**. Si coinciden, se
omite.

```powershell
UltraCopySam -u -w 4 "D:\dev" "E:\backup\dev"
```

```text
0 archivos, 41 directorios, 0.00 MB en 20ms (0.00 MB/s)
20000 archivos sin cambios, 7.63 MB que no hubo que reescribir
```

### Cuándo usarlo

Siempre que **repitas** una copia sobre un destino ya poblado: respaldos
periódicos, sincronizar un disco externo, o reanudar una copia que se
interrumpió. La ganancia es enorme porque el coste real de copiar no está en
mover los bytes, sino en crear cada archivo en el destino.

Medición sobre un árbol de 20.000 archivos:

| Escenario | Sin `-u` | Con `-u` |
| --- | --- | --- |
| Primera copia (destino vacío) | 15,7 s | 14,3 s |
| Segunda pasada (todo igual) | 10,3 s | **0,05 s** |

En la primera copia **no penaliza**: cuando una carpeta de destino acaba de
crearse, está vacía por definición y no se consulta. En la segunda pasada es
unas **200 veces más rápido**.

### Cómo lo comprueba sin perder tiempo

Los datos del origen ya vienen gratis con el recorrido. Los del destino se
obtienen con **un solo listado por carpeta**, no con una consulta por archivo:
una llamada al sistema por directorio, integrada en el mismo recorrido
solapado, sin ninguna fase de análisis previa.

### Precisión de la comparación

En **NTFS la comparación de fechas es exacta** (resolución de 100 ns). Solo si
el destino usa FAT o exFAT —que redondean la hora a 2 segundos— se aplica un
margen de esa magnitud, porque de lo contrario ningún archivo coincidiría
jamás. El sistema de archivos se detecta automáticamente.

> [!WARNING]
> `-u` da por idéntico un archivo que conserve **tamaño y fecha**. Si un
> programa modifica un archivo manteniendo ambos exactamente iguales, el cambio
> no se detecta. Es prácticamente imposible por accidente, pero si necesitas la
> certeza absoluta de que el destino queda igual al origen, ejecuta sin `-u`:
> el modo normal reescribe siempre.

---

## Comparativa con robocopy

Cifras honestas, incluyendo dónde UltraCopySam pierde. Medido sobre un único SSD
NVMe, con el destino borrado antes de cada corrida, frente a `robocopy /E /MT`
(su modo multihilo). Mediana de 3 corridas, salvo el escenario de archivos
pequeños, que usa 7 corridas alternadas por ser donde ambas herramientas están
más igualadas.

| Escenario | robocopy `/MT` | UltraCopySam | Ganador |
| --- | --- | --- | --- |
| 20.000 archivos pequeños (7,6 MB) | **10,56 s** | 12,77 s | robocopy, por un 21% |
| 8 archivos grandes (2.000 MB) | 1,49 s | **0,92 s** | UltraCopySam, **1,6× más rápido** |
| Mixto: 5.006 archivos (734 MB) | **2,74 s** | 3,41 s | robocopy, por un 20% |
| Segunda pasada, sin cambios | 0,03 s | 0,03 s | Empate |

**Cómo leer los resultados:**

- Con **archivos grandes** UltraCopySam gana con claridad, gracias a
  `COPY_FILE_NO_BUFFERING` en archivos de 32 MiB o más.
- Con **muchos archivos pequeños** robocopy gana por un 21%. El resultado es
  consistente: en 7 corridas alternadas los rangos ni siquiera se solapan
  (robocopy 10,08-11,29 s; UltraCopySam 11,73-13,21 s).
- En **respaldos repetidos** ambos empatan, porque ninguno reescribe lo que no
  ha cambiado.

**De dónde *no* viene esa diferencia.** Se probaron tres hipótesis con un
prototipo aparte y las tres quedaron descartadas:

| Hipótesis | Resultado |
| --- | --- |
| `CopyFileExW` pesa demasiado en archivos pequeños; con `ReadFile`/`WriteFile` manual sería más rápido | **Descartada**: quedan a un 0,5% una de otra |
| Las rutas extendidas `\\?\` cuestan más | **Descartada**: son ligeramente *más* rápidas |
| La cola con `sync.Cond` es más lenta que un channel de Go | **Descartada**: la cola resultó más rápida que el channel |

El coste por archivo tampoco está en el recorrido: recorrer esos mismos 20.000
archivos en modo `-u` toma **0,04 s**. La causa exacta de la diferencia restante
aún no está identificada y haría falta un *profiling* real para localizarla. Se
documenta aquí en lugar de ocultarse.

Puedes reproducirlas con el script de [bench/bench.ps1](bench/bench.ps1).

---

## ¿Usar esto en lugar de robocopy?

`robocopy` viene incluido en Windows, está más que probado y, como muestran las
cifras anteriores, es más rápido con archivos pequeños. Es una gran herramienta
y no deberías cambiarla solo por velocidad.

Dónde compensa UltraCopySam:

- **Archivos grandes**: es medible y consistentemente más rápido.
- **Simplicidad**: dos rutas y listo, frente a los ~80 modificadores de
  robocopy. No hay que recordar ningún `/E /MT:32 /NFL /NDL /NJH /NJS`.
- **Mensajes de error claros** en lugar de códigos numéricos que hay que
  consultar. Te dice qué falló y cómo arreglarlo, incluida la trampa clásica de
  Windows con la barra invertida final dentro de las comillas.
- **Código legible**: unas 1.300 líneas de Go auditables, en vez de un binario
  cerrado.
- **Comportamiento predecible**: nunca borra nada en el destino. No existe un
  `/MIR` capaz de vaciar una carpeta por error.

Si necesitas espejo, filtros, reintentos o copia de permisos NTFS, usa robocopy:
esta herramienta deliberadamente no cubre eso.

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
1284 archivos | 3491.84 MB | 812.44 MB/s
```

Al terminar imprime el resumen:

```text
5327 archivos, 214 directorios, 13137.92 MB en 16.204s (811.55 MB/s)
```

y, si corresponde, cuántas entradas se omitieron y cuántos errores hubo.

Detalles útiles:

- Los volúmenes se expresan **siempre en MB**, sin escalar a GB. Una unidad fija
  evita que la cifra salte de unidad a mitad de la copia, que es justo lo que
  impide comparar de un vistazo si el ritmo sube o baja.
- El **progreso va a `stderr`** y el **resumen a `stdout`**, de modo que puedes
  redirigir el resultado a un archivo sin que se llene de líneas parpadeantes.
- Con `-v` se desactiva la línea de progreso y se imprime cada archivo copiado.
- Con `-q` no se muestra nada salvo los errores.
- En copias muy cortas (menos de 250 ms) solo verás el resumen final.
- El contador **suma cada archivo al completarse**, no mientras se copia. Con un
  único archivo muy grande la cifra permanece en `0.00 MB` hasta que termina.
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
  estaba en curso se vuelve a copiar completo en la siguiente pasada (con `-u`,
  los que sí terminaron se saltan).
- **La comparación de `-u` es por tamaño y fecha**, no por contenido. Ver
  [Copias incrementales](#copias-incrementales).
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
| `install.ps1` | Instalador y desinstalador para PowerShell |
| `bench/bench.ps1` | Script de benchmark usado para las cifras publicadas |
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
