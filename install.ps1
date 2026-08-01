<#
.SYNOPSIS
    Instalador de UltraCopySam para Windows.

.DESCRIPTION
    Descarga la última versión publicada en GitHub Releases (o usa un archivo
    local), la instala en la carpeta del usuario, la desbloquea para que
    Windows SmartScreen no interfiera y la agrega al PATH.

    No requiere permisos de administrador: todo se instala en el perfil del
    usuario actual.

.PARAMETER Version
    Etiqueta de la versión a instalar, por ejemplo "v0.0.1".
    Por defecto instala la más reciente.

.PARAMETER InstallDir
    Carpeta de instalación.
    Por defecto: %LOCALAPPDATA%\Programs\UltraCopySam

.PARAMETER FromFile
    Instala desde un .exe que ya tienes en disco, sin descargar nada.

.PARAMETER NoPath
    No modifica la variable PATH.

.PARAMETER Uninstall
    Desinstala: elimina la carpeta de instalación y la quita del PATH.

.EXAMPLE
    .\install.ps1
    Instala la última versión.

.EXAMPLE
    irm https://raw.githubusercontent.com/cahosoft2/UltraCopySam/main/install.ps1 | iex
    Instala directamente desde internet, sin clonar el repositorio.

.EXAMPLE
    .\install.ps1 -Uninstall
    Desinstala UltraCopySam.

.LINK
    https://github.com/cahosoft2/UltraCopySam
#>

[CmdletBinding()]
param(
    [string] $Version    = 'latest',
    [string] $InstallDir = "$env:LOCALAPPDATA\Programs\UltraCopySam",
    [string] $FromFile,
    [switch] $NoPath,
    [switch] $Uninstall
)

$ErrorActionPreference = 'Stop'

$repo    = 'cahosoft2/UltraCopySam'
$appName = 'UltraCopySam'
$exeName = 'UltraCopySam.exe'

function Write-Paso  { param($m) Write-Host "==> $m" -ForegroundColor Cyan }
function Write-Ok    { param($m) Write-Host "    $m" -ForegroundColor Green }
function Write-Aviso { param($m) Write-Host "    $m" -ForegroundColor Yellow }

# --------------------------------------------------------------------------
# PATH del usuario (nunca el del sistema: así no hace falta ser administrador)
#
# Se lee y escribe el registro directamente en lugar de usar
# [Environment]::SetEnvironmentVariable, porque ese método expande las
# variables del valor (%USERPROFILE% pasaría a ser una ruta fija) y cambia el
# tipo de REG_EXPAND_SZ a REG_SZ, corrompiendo el PATH de quien las use.
# --------------------------------------------------------------------------

function Get-UserPathEntry {
    $clave = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment')
    try {
        if (-not $clave) {
            return @{ Valor = ''; Tipo = [Microsoft.Win32.RegistryValueKind]::ExpandString }
        }
        # DoNotExpandEnvironmentNames conserva los %...% tal cual están escritos.
        $valor = $clave.GetValue('Path', '',
                    [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        $tipo  = try { $clave.GetValueKind('Path') }
                 catch { [Microsoft.Win32.RegistryValueKind]::ExpandString }
        return @{ Valor = [string]$valor; Tipo = $tipo }
    } finally {
        if ($clave) { $clave.Close() }
    }
}

function Set-UserPathEntry {
    param([string] $Valor, $Tipo)

    $clave = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $true)
    try {
        $clave.SetValue('Path', $Valor, $Tipo)
    } finally {
        if ($clave) { $clave.Close() }
    }

    # Avisar a Windows del cambio; si no, las aplicaciones ya abiertas
    # (incluido el Explorador) seguirían viendo el PATH anterior.
    if (-not ('Win32.NativeMethods' -as [type])) {
        Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition @'
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam,
    string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
    }
    $resultado = [UIntPtr]::Zero
    [void][Win32.NativeMethods]::SendMessageTimeout(
        [IntPtr]0xFFFF, 0x1A, [UIntPtr]::Zero, 'Environment', 2, 5000, [ref]$resultado)
}

function Add-ToUserPath {
    param([string] $Dir)

    $entrada = Get-UserPathEntry
    $partes  = @($entrada.Valor -split ';' | Where-Object { $_ })

    if ($partes -contains $Dir) {
        Write-Ok "Ya estaba en el PATH"
        return
    }

    Set-UserPathEntry ((@($partes) + $Dir) -join ';') $entrada.Tipo
    $env:PATH = "$env:PATH;$Dir"       # disponible ya en esta misma sesión
    Write-Ok "Agregado al PATH del usuario"
}

function Remove-FromUserPath {
    param([string] $Dir)

    $entrada = Get-UserPathEntry
    $partes  = @($entrada.Valor -split ';' | Where-Object { $_ })

    if ($partes -notcontains $Dir) {
        Write-Aviso "No estaba en el PATH"
        return
    }

    Set-UserPathEntry (($partes | Where-Object { $_ -ne $Dir }) -join ';') $entrada.Tipo
    Write-Ok "Eliminado del PATH del usuario"
}

# --------------------------------------------------------------------------
# Desinstalación
# --------------------------------------------------------------------------

if ($Uninstall) {
    Write-Paso "Desinstalando $appName"

    if (Test-Path $InstallDir) {
        Remove-Item -Recurse -Force $InstallDir
        Write-Ok "Carpeta eliminada: $InstallDir"
    } else {
        Write-Aviso "No estaba instalado en $InstallDir"
    }

    Remove-FromUserPath $InstallDir

    Write-Host ""
    Write-Host "$appName desinstalado." -ForegroundColor Green
    return
}

# --------------------------------------------------------------------------
# Comprobaciones previas
# --------------------------------------------------------------------------

if ($env:PROCESSOR_ARCHITECTURE -notin @('AMD64', 'ARM64')) {
    throw "$appName requiere Windows de 64 bits (detectado: $env:PROCESSOR_ARCHITECTURE)."
}

# PowerShell 5.1 no negocia TLS 1.2 por defecto y GitHub lo exige.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("ultracopysam_" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp -Force | Out-Null

try {
    # ----------------------------------------------------------------------
    # Obtener el ejecutable
    # ----------------------------------------------------------------------

    if ($FromFile) {
        Write-Paso "Instalando desde archivo local"

        if (-not (Test-Path $FromFile)) {
            throw "No se encuentra el archivo: $FromFile"
        }
        $origen = (Resolve-Path $FromFile).Path
        Write-Ok (Split-Path $origen -Leaf)

    } else {
        Write-Paso "Consultando la última versión publicada"

        $api = if ($Version -eq 'latest') {
            "https://api.github.com/repos/$repo/releases/latest"
        } else {
            "https://api.github.com/repos/$repo/releases/tags/$Version"
        }

        try {
            $release = Invoke-RestMethod -Uri $api -Headers @{ 'User-Agent' = $appName }
        } catch {
            throw "No se pudo consultar la versión '$Version' en GitHub. $($_.Exception.Message)"
        }

        $asset = $release.assets | Where-Object { $_.name -like '*.exe' } | Select-Object -First 1
        if (-not $asset) {
            throw "La versión $($release.tag_name) no tiene ningún ejecutable adjunto."
        }

        Write-Ok "$($release.tag_name) — $($asset.name) ($([math]::Round($asset.size/1MB, 2)) MB)"

        Write-Paso "Descargando"
        $origen = Join-Path $tmp $asset.name
        Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $origen -UseBasicParsing
        Write-Ok "Descarga completada"

        Write-Paso "Verificando integridad"
        $hash = (Get-FileHash $origen -Algorithm SHA256).Hash
        Write-Ok "SHA256: $hash"
    }

    # ----------------------------------------------------------------------
    # Instalar
    # ----------------------------------------------------------------------

    Write-Paso "Instalando en $InstallDir"

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $destino = Join-Path $InstallDir $exeName

    # Si ya está instalado y en ejecución, Windows impide sobrescribirlo.
    if (Test-Path $destino) {
        try {
            Copy-Item $origen $destino -Force
        } catch {
            throw "No se pudo reemplazar $destino. ¿Hay una copia en ejecución? Ciérrela y reintente."
        }
    } else {
        Copy-Item $origen $destino -Force
    }

    # Quita el "Mark of the Web": sin esto SmartScreen bloquea el ejecutable.
    Unblock-File $destino
    Write-Ok "Instalado y desbloqueado"

    if (-not $NoPath) {
        Write-Paso "Configurando el PATH"
        Add-ToUserPath $InstallDir
    }

    # ----------------------------------------------------------------------
    # Comprobación final
    # ----------------------------------------------------------------------

    Write-Paso "Comprobando la instalación"

    # Sin argumentos el programa imprime la ayuda y sale con código 2.
    # Se captura toda la salida antes de leer $LASTEXITCODE: cortar el
    # pipeline (por ejemplo con Select-Object -First) lo dejaría sin asignar.
    $salida = & $destino 2>&1
    $codigo = $LASTEXITCODE

    if ($codigo -eq 2) {
        Write-Ok "El ejecutable responde correctamente"
    } else {
        Write-Aviso "Respuesta inesperada (código $codigo): $($salida | Select-Object -First 1)"
    }

    Write-Host ""
    Write-Host "$appName instalado correctamente." -ForegroundColor Green
    Write-Host ""
    Write-Host "  Ubicación : $destino"
    Write-Host "  Uso       : UltraCopySam `"D:\origen`" `"E:\destino`""
    Write-Host "  Ayuda     : UltraCopySam"
    Write-Host ""

    if (-not $NoPath) {
        Write-Host "  Abre una terminal nueva para que el PATH surta efecto." -ForegroundColor Yellow
        Write-Host ""
    }

    # La comprobación anterior dejó $LASTEXITCODE en 2; el instalador debe
    # terminar en 0 para no confundir a quien lo llame desde un script.
    $global:LASTEXITCODE = 0
}
finally {
    if (Test-Path $tmp) {
        Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }
}
