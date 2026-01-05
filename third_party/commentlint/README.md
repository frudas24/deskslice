# commentlint

Linter Go simple que exige un comentario de documentación para **todas las funciones** (exportadas o no). Se pensó para integrarse con `golangci-lint`, pero en este repositorio se ejecuta como paso independiente (`make commentlint`).

## Instalación / prerequisitos

- Go 1.24 o superior.
- Este directorio ya contiene el código; no requiere build previo (se ejecuta vía `go run`).

## Uso

```bash
# revisar todo el repo (respetando .golangci.yml)
go run ./third_party/commentlint ./...

# revisar sólo los paquetes de synapse
make commentlint internal/synapse/...
```

Por defecto se revisa todo (`./...`). Se respetan las exclusiones y límites que se definen en `.golangci.yml`.

## Integración con `.golangci.yml`

`commentlint` lee la sección `issues` para replicar el comportamiento de GolangCI:

```yaml
issues:
  max-issues-per-linter: 10   # límite de reportes (0 = sin límite)
  exclude-dirs:
    - dist
    - vendor
    - internal/rpc/proto
  exclude-files:
    - ".*_grpc\\.pb\\.go"
    - ".*\\.pb\\.go"
```

Si el archivo no existe, se usan valores por defecto (sin exclusiones ni límite). Las rutas se consideran relativas a la raíz del repo.

## Integración con el flujo

- El target `make commentlint` ejecuta el linter antes del lint habitual.
- `make build` no lo invoca todavía (para evitar ruido). Si querés hacerlo obligatorio, añadí `commentlint` como dependencia del target `lint` o del `build`.

## Limitaciones conocidas

- No se integra todavía como plugin real de `golangci-lint` (se ejecuta como comando separado).
- No ignora funciones generadas salvo que el archivo contenga la cabecera estándar `// Code generated... DO NOT EDIT.`
