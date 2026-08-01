<#
.SYNOPSIS
    Benchmark: UltraCopySam vs robocopy.

.DESCRIPTION
    Compara ambas herramientas sobre el mismo árbol de origen. Cada corrida
    parte de un destino limpio y se repite N veces; se reporta la mediana para
    amortiguar el ruido del sistema de archivos y de la caché.

    Mide también la segunda pasada (destino ya poblado e idéntico), que es donde
    actúan el modo incremental de UltraCopySam (-u) y el de robocopy.

.PARAMETER Origen
    Carpeta a copiar. Si se omite, se genera un árbol sintético de prueba.

.PARAMETER Escenario
    Árbol sintético a generar: pequenos (20.000 archivos de 400 B),
    grandes (8 archivos de 250 MB) o mixto (5.006 archivos, 734 MB).

.PARAMETER Repeticiones
    Número de corridas por herramienta. Por defecto 3.

.PARAMETER Workers
    Copias en paralelo: -w para UltraCopySam, /MT para robocopy.

.EXAMPLE
    .\bench.ps1 -Escenario pequenos -Workers 32

.EXAMPLE
    .\bench.ps1 -Origen "D:\mis datos" -Workers 8 -Repeticiones 5

.LINK
    https://github.com/cahosoft2/UltraCopySam
#>

[CmdletBinding()]
param(
    [string] $Origen,
    [ValidateSet('pequenos', 'grandes', 'mixto')]
    [string] $Escenario = 'mixto',
    [int]    $Repeticiones = 3,
    [int]    $Workers = 16,
    [string] $UltraCopySam = 'UltraCopySam.exe',
    [string] $CarpetaTrabajo = (Join-Path ([IO.Path]::GetTempPath()) 'ultracopysam_bench')
)

$ErrorActionPreference = 'Stop'

if (-not (Get-Command $UltraCopySam -ErrorAction SilentlyContinue)) {
    throw "No se encuentra '$UltraCopySam'. Indíquelo con -UltraCopySam o añádalo al PATH."
}

New-Item -ItemType Directory -Force $CarpetaTrabajo | Out-Null

function Borrar($ruta) {
    if (Test-Path $ruta) { [System.IO.Directory]::Delete($ruta, $true) }
}

function Mediana($valores) {
    $orden = @($valores | Sort-Object)
    $orden[[math]::Floor($orden.Count / 2)]
}

# --------------------------------------------------------------------------
# Árbol de origen
# --------------------------------------------------------------------------

if (-not $Origen) {
    $Origen = Join-Path $CarpetaTrabajo "src_$Escenario"
    if (-not (Test-Path $Origen)) {
        Write-Host "Generando árbol de prueba '$Escenario'..." -ForegroundColor Cyan
        New-Item -ItemType Directory -Force $Origen | Out-Null

        switch ($Escenario) {
            'pequenos' {
                1..40 | ForEach-Object {
                    $sub = Join-Path $Origen "dir$_"
                    New-Item -ItemType Directory -Force $sub | Out-Null
                    1..500 | ForEach-Object {
                        [System.IO.File]::WriteAllText((Join-Path $sub "f$_.txt"), ('x' * 400))
                    }
                }
            }
            'grandes' {
                1..8 | ForEach-Object {
                    $fs = [System.IO.File]::Create((Join-Path $Origen "g$_.bin"))
                    $fs.SetLength(250MB); $fs.Close()
                }
            }
            'mixto' {
                1..20 | ForEach-Object {
                    $sub = Join-Path $Origen "modulo$_"
                    New-Item -ItemType Directory -Force $sub | Out-Null
                    1..250 | ForEach-Object {
                        [System.IO.File]::WriteAllText((Join-Path $sub "src$_.js"), ('y' * 3000))
                    }
                }
                1..6 | ForEach-Object {
                    $fs = [System.IO.File]::Create((Join-Path $Origen "recurso$_.bin"))
                    $fs.SetLength(120MB); $fs.Close()
                }
            }
        }
    }
}

$archivos = @(Get-ChildItem -Recurse -File $Origen)
$totalMB  = [math]::Round(($archivos | Measure-Object -Sum Length).Sum / 1MB, 2)

Write-Host ""
Write-Host "Origen       : $Origen"
Write-Host "Contenido    : $($archivos.Count) archivos, $totalMB MB"
Write-Host "Repeticiones : $Repeticiones   Workers: $Workers"
Write-Host ""

$dstRobocopy = Join-Path $CarpetaTrabajo 'dst_robocopy'
$dstUltra    = Join-Path $CarpetaTrabajo 'dst_ultracopysam'

# --------------------------------------------------------------------------
# Primera copia: destino vacío
# --------------------------------------------------------------------------

function MedirPrimeraCopia($nombre, $accion, $destino) {
    $tiempos = @()
    for ($i = 1; $i -le $Repeticiones; $i++) {
        Borrar $destino
        [System.GC]::Collect()
        $sw = [Diagnostics.Stopwatch]::StartNew()
        & $accion $destino | Out-Null
        $sw.Stop()
        $tiempos += $sw.Elapsed.TotalSeconds
        Write-Host ("  {0,-28} corrida {1}: {2,6:N2} s" -f $nombre, $i, $sw.Elapsed.TotalSeconds)
    }
    [pscustomobject]@{
        Herramienta = $nombre
        Mediana_s   = [math]::Round((Mediana $tiempos), 2)
        Min_s       = [math]::Round(($tiempos | Measure-Object -Minimum).Minimum, 2)
        Max_s       = [math]::Round(($tiempos | Measure-Object -Maximum).Maximum, 2)
    }
}

Write-Host "PRIMERA COPIA (destino vacío)" -ForegroundColor Yellow

$primera = @()
$primera += MedirPrimeraCopia "robocopy /MT:$Workers" {
    param($d) robocopy $Origen $d /E /MT:$Workers /NFL /NDL /NJH /NJS /NC /NS /NP
} $dstRobocopy

$primera += MedirPrimeraCopia "UltraCopySam -w $Workers" {
    param($d) & $UltraCopySam -q -w $Workers $Origen $d
} $dstUltra

Write-Host ""
$primera | Format-Table -AutoSize

# --------------------------------------------------------------------------
# Segunda pasada: destino ya idéntico
# --------------------------------------------------------------------------

function MedirSegundaPasada($nombre, $accion) {
    $tiempos = @()
    for ($i = 1; $i -le $Repeticiones; $i++) {
        $sw = [Diagnostics.Stopwatch]::StartNew()
        & $accion | Out-Null
        $sw.Stop()
        $tiempos += $sw.Elapsed.TotalSeconds
    }
    [pscustomobject]@{
        Herramienta = $nombre
        Mediana_s   = [math]::Round((Mediana $tiempos), 2)
    }
}

Write-Host "SEGUNDA PASADA (destino ya idéntico)" -ForegroundColor Yellow

$segunda = @()
$segunda += MedirSegundaPasada "robocopy /MT:$Workers" {
    robocopy $Origen $dstRobocopy /E /MT:$Workers /NFL /NDL /NJH /NJS /NC /NS /NP
}
$segunda += MedirSegundaPasada "UltraCopySam -u -w $Workers" {
    & $UltraCopySam -u -q -w $Workers $Origen $dstUltra
}

Write-Host ""
$segunda | Format-Table -AutoSize

Write-Host "Los destinos quedan en $CarpetaTrabajo (bórrelos cuando termine)." -ForegroundColor DarkGray
