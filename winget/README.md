# Manifiestos de winget

Manifiestos para publicar UltraCopySam en el
[Windows Package Manager](https://github.com/microsoft/winget-pkgs), de modo que
pueda instalarse con:

```powershell
winget install cahosoft2.UltraCopySam
```

Los tres manifiestos viven en `manifests/`, aislados de este documento:
`winget validate` intenta interpretar como YAML **todos** los archivos de la
carpeta que se le indica, y un `.md` la haría fallar.

| Archivo | Contenido |
| --- | --- |
| `manifests/cahosoft2.UltraCopySam.yaml` | Manifiesto de versión |
| `manifests/cahosoft2.UltraCopySam.locale.en-US.yaml` | Metadatos: descripción, licencia, etiquetas |
| `manifests/cahosoft2.UltraCopySam.installer.yaml` | URL de descarga, arquitectura y SHA256 |

## Validar antes de enviar

```powershell
winget validate --manifest .\winget\manifests
```

## Enviarlos a Microsoft

1. Haz un *fork* de [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs).
2. Copia los tres YAML de `manifests/` a
   `manifests/c/cahosoft2/UltraCopySam/<versión>/` dentro de ese repositorio.
3. Abre un *pull request*. La validación automática comprueba el hash, descarga
   el binario y lo analiza con varios antivirus.

Alternativamente, [wingetcreate](https://github.com/microsoft/winget-create)
automatiza el proceso:

```powershell
winget install Microsoft.WingetCreate
wingetcreate update cahosoft2.UltraCopySam `
    --version 0.0.4 `
    --urls https://github.com/cahosoft2/UltraCopySam/releases/download/v0.0.4/UltraCopySamV004.exe `
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
