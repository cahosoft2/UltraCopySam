# Manifiestos de winget

Manifiestos para publicar UltraCopySam en el
[Windows Package Manager](https://github.com/microsoft/winget-pkgs), de modo que
pueda instalarse con:

```powershell
winget install cahosoft2.UltraCopySam
```

| Archivo | Contenido |
| --- | --- |
| `cahosoft2.UltraCopySam.yaml` | Manifiesto de versión |
| `cahosoft2.UltraCopySam.locale.en-US.yaml` | Metadatos: descripción, licencia, etiquetas |
| `cahosoft2.UltraCopySam.installer.yaml` | URL de descarga, arquitectura y SHA256 |

## Validar antes de enviar

```powershell
winget validate --manifest .\winget
```

## Enviarlos a Microsoft

1. Haz un *fork* de [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs).
2. Copia los tres YAML a
   `manifests/c/cahosoft2/UltraCopySam/<versión>/`.
3. Abre un *pull request*. La validación automática comprueba el hash, descarga
   el binario y lo analiza con varios antivirus.

Alternativamente, [wingetcreate](https://github.com/microsoft/winget-create)
automatiza el proceso:

```powershell
winget install Microsoft.WingetCreate
wingetcreate update cahosoft2.UltraCopySam `
    --version 0.0.2 `
    --urls https://github.com/cahosoft2/UltraCopySam/releases/download/v0.0.2/UltraCopySamV002.exe `
    --submit
```

## Al publicar una versión nueva

Hay que actualizar en los tres archivos:

- `PackageVersion`
- `InstallerUrl` y `InstallerSha256` (el hash lo imprime el workflow de release)
- `ReleaseNotesUrl` y `ReleaseDate`

> [!NOTE]
> El binario no está firmado con un certificado de código. La validación de
> winget lo acepta, pero el paquete puede tardar más en aprobarse por las
> comprobaciones antivirus.
