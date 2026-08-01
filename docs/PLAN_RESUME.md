# Plan de Diseño: Reanudación de Copia Interrumpida (`-r` / `--resume`) mediante Archivo de Estado (`.ucsam-state`)

Documento de propuesta técnica para la implementación de la opción de reanudación en **UltraCopySam**.

---

## 1. Comportamiento y Semántica del Flag `-r`

Existe dos modelos posibles para la interacción del usuario con la opción `-r`:

### Modelo Propuesto: Control Explícito mediante `-r`

1. **Cuando el flag `-r` SÍ está activado (`UltraCopySam.exe -r "src" "dst"`)**:
   - **Paso 1 (Detección)**: Busca si ya existe `dst\.ucsam-state`. Si existe y coincide con el origen, **carga el estado y reanuda la copia automáticamente** desde el punto donde se interrumpió.
   - **Paso 2 (Creación/Seguimiento)**: Si no existe el archivo (o se reanudó), mantiene el archivo `.ucsam-state` actualizado en disco durante la copia para que, si vuelve a interrumpirse, se pueda retomar.
   - **Al finalizar 100%**: Elimina el archivo `.ucsam-state`.

2. **Cuando el flag `-r` NO está activado (uso normal por defecto)**:
   - No lee ni busca `.ucsam-state`.
   - No crea `.ucsam-state` durante la copia normal.
   - Copia todo desde cero reemplazando sin preguntar (comportamiento estándar sin sobrecarga de archivos de estado).
   - *(Opcional)*: Si se presiona `Ctrl+C`, puede guardar `.ucsam-state` únicamente al cancelar como "salvavidas" para que el usuario pueda usar `-r` en el siguiente intento.

---

## 2. Ventajas del Modelo Explícito

- **Cero Sobrecarga en Copias Normales**: Las ejecuciones estándar sin `-r` no realizan escrituras adicionales ni crean archivos en el destino.
- **Predicibilidad**: El usuario decide explícitamente cuándo quiere que UltraCopySam gestione la recuperación de la sesión.

---

## 3. Estructura y Formato del Archivo `.ucsam-state`

El archivo de estado se ubica en la raíz del directorio de destino:
`\destino\.ucsam-state`

```json
{
  "version": 1,
  "source": "D:\\origen",
  "destination": "E:\\destino",
  "started_at": "2026-08-01T18:00:00Z",
  "updated_at": "2026-08-01T18:05:12Z",
  "completed_dirs": [
    "subcarpeta1",
    "subcarpeta2\\profunda"
  ]
}
```

---

## 4. Diseño del Mecanismo y Evitación de Cuellos de Botella

1. **Granularidad por Directorio**:
   - En lugar de registrar cada archivo individual (lo que generaría contención de E/S con múltiples goroutines), se registran los **directorios 100% completados**.
2. **Flush Periódico y en Cancelación (`Ctrl+C`)**:
   - Escritura atómica a disco (`.tmp` -> `rename`) cada 2 segundos y de forma inmediata en `Ctrl+C`.
3. **Limpieza Automática**:
   - Al completar la copia exitosamente al 100%, el archivo `.ucsam-state` se elimina automáticamente.

---

## 5. Salida en Consola

```text
25000 archivos, 1200 directorios, 4500.00 MB en 15.2s (296.05 MB/s)
18400 archivos omitidos por estado previo (.ucsam-state)
3 archivos parciales reemplazados
```
