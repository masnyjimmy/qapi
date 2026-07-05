# qapi (quick api)

**qapi** is a compact, opinionated alternative syntax for describing HTTP APIs. You write a small, low-boilerplate `qapi.yaml` file, and the `qapi` CLI turns it into a standard **OpenAPI 3.1** document — or serves it directly as a live, hot-reloading API docs site.

The goal is simple: describe endpoints faster, with less repetition, and let the tool expand it into full OpenAPI for you.

> ⚠️ **Status:** qapi is still in early / unfinished development. The format and CLI may change.

---

## Why qapi?

Hand-writing OpenAPI YAML means repeating the same `content: application/json: schema: $ref: ...` structure, the same `400/401/404/500` error responses on every single path, and verbose `oneOf`-based nullability. qapi collapses all of that into short, readable expressions and lets you define shared pieces (like default error responses or paged-list parameters) once.

A qapi schema like:

```yaml
Organizator:
  id: number
  name: string
  email: string?
  phone: string?
  note: string?
  events_count: integer
```

expands into the full OpenAPI object schema (with `oneOf`-based nullable fields, `required` arrays, etc.) automatically.

---

## Installation

qapi is written in Go. For now, the only way to install it is directly via the Go toolchain:

```bash
go install github.com/masnyjimmy/qapi@latest
```

---

## Quick Start

1. Write a `.yaml` file in the qapi format (see below), e.g. `api.qapi.yaml`.
2. Compile it to standard OpenAPI:

   ```bash
   qapi compile -i api.qapi.yaml -o openapi.yaml
   ```

3. ...or just serve it directly as a live docs site, with hot reload on save:

   ```bash
   qapi serve -i api.qapi.yaml
   ```

---

## CLI Reference

### `qapi compile`

Compiles a qapi YAML file into a standard OpenAPI 3.1 YAML file.

```bash
qapi compile -i/--input <input.yaml> -o/--output <output.yaml>
```

| Flag | Short | Required | Description |
|---|---|---|---|
| `--input` | `-i` | ✅ | Path to the qapi source file |
| `--output` | `-o` | ✅ | Path to write the generated OpenAPI file |

### `qapi serve`

Serves the qapi file as a live OpenAPI documentation website. Watches the input file and hot-reloads when it changes.

```bash
qapi serve -i/--input <input.yaml>
```

| Flag | Short | Required | Description |
|---|---|---|---|
| `--input` | `-i` | ✅ | Path to the qapi source file |

Only the input is required for `serve` — qapi compiles internally and serves the result on save.

---

## The qapi Format

A qapi file is a single YAML document with these top-level keys:

```yaml
info:            # required — API metadata
servers:         # required — list of server URLs
tags:            # optional — tag descriptions
schemas:         # optional — reusable data models
traits:          # optional — reusable parameter/header snippets
defaultResponses:# optional — responses applied to every method
paths:           # the actual endpoint tree
```

### `info` (required)

```yaml
info:
  title: Miastobar Backend
  version: 0.0.1
  description: Serwer baru Miasto  # optional
```

Maps directly to OpenAPI's `info` object (`title` and `version` are required, `description` is optional).

### `servers` (required)

A list of server objects, identical to OpenAPI:

```yaml
servers:
  - url: "http://localhost:8080"
    description: Local testing server
```

### `tags` (optional)

A list of tag definitions, identical to OpenAPI:

```yaml
tags:
  - name: Auth
    description: Authorization endpoints
```

Tags can also be attached directly to a group of paths (see [Paths](#paths) below), which is usually more convenient than listing every path's tags individually.

---

### `schemas`

Reusable data models, referenced elsewhere with `<SchemaName>`. Each schema is either:

- an **object**, defined with plain key/value pairs (see [Schema expressions](#schema-expressions) for the value syntax), or
- a **schema expression** itself (a bare type), e.g.:

  ```yaml
  File: "string($binary)"
  ```

Object example:

```yaml
Organizator:
  id: number
  name: string
  email: string?
  phone: string?
  note: string?
  events_count: integer
```

Object fields can themselves be nested objects — the format supports arbitrary nesting, not just flat field lists.

A field name suffixed with `?` (e.g. `name?: string`) marks that **field as optional** (i.e. not included in the compiled `required` array). This is independent from the value being nullable — see below.

#### Schema expressions

Every schema value (a field type, a param schema, a response body, etc.) is one of:

**1. A primitive**, with optional nullability and format:

```
<type>[?][(<format>)]
```

- `type` is one of `boolean`, `string`, `integer`, `number`
- an optional trailing `?` makes the value **nullable** — compiles to `oneOf: [{type: "null"}, {type: <type>}]`
- an optional `(...)` sets a format-like modifier, e.g.:
  - `string($errorCode)` → `type: string, format: errorCode`
  - `integer($imageId)` → `type: integer, format: imageId`
  - `string($date-time)` → `type: string, format: date-time`

  The `(...)` block can also carry other compiled attributes (defaults, maximums, etc.) — see [Traits](#traits) below, where this is used to parameterize things like default/maximum values.

**2. A reference**, optionally as an array:

```
<SchemaName>[?][[]...]
```

- `<Category>` references the `Category` schema defined under `schemas`
- `<Category>[]` compiles to an array: `type: array, items: { $ref: '#/components/schemas/Category' }`

Examples from a real file:

```yaml
categoryId: integer
category: string
images: <ImageInfo>[]
cursor: string?
```

---

### `traits`

Traits are reusable, parameterized bundles of `params`/`headers`, so you don't have to redefine the same pagination (or similar) parameters on every endpoint.

Definition — the trait name can declare parameter placeholders in parentheses:

```yaml
traits:
  paged(Def,Max):
    params:
      - name: cursor
        schema: "string"
      - name: limit
        schema: "integer(#Def,<#Max)"
```

Inside the trait body, `#Def` and `#Max` refer to the values the trait is invoked with. Based on the compiled output, within the parenthesized modifier block:

- a bare value sets the compiled **default**
- a value prefixed with `<` sets the compiled **maximum**

So `integer(#Def,<#Max)` invoked as `paged(20,100)` compiles to:

```yaml
type: integer
default: 20
maximum: 100
```

Usage on a method:

```yaml
get:
  id: ListImages
  traits: ["paged(20,100)"]
```

This injects the trait's `params` (and `headers`, if defined) into that method, with `Def=20` and `Max=100` substituted.

---

### `defaultResponses`

Response definitions that are automatically merged into **every** method's `responses`, so you don't have to repeat the same `400/401/404/500` error shapes on every single endpoint:

```yaml
defaultResponses:
  400:
    description: Bad Request
    application/json:
      <DefaultError>
  401:
    description: Unauthorized
    application/json:
      <DefaultError>
  404:
    description: Not found
    application/json:
      <DefaultError>
  500:
    description: Internal Server Error
    application/json:
      <DefaultError>
```

A method only needs to declare its "success"-path responses (`200`, `201`, `204`, ...); the default error responses are appended automatically during compilation.

---

### `paths`

`paths` is a **nested tree** of path segments rather than a flat map of full paths (as in vanilla OpenAPI). Each key is a path segment (or `{param}` placeholder), and its value can contain further nested segments, a `tags` list, and/or HTTP method definitions (`get`, `post`, `put`, `patch`, `delete`).

```yaml
paths:
  /api/v1:
    /resources:
      tags: [Resources]
      /image:
        put:
          id: UploadImage
          ...
        get:
          id: ListImages
          ...
      /image/{imageId}:
        get:
          id: GetImage
          ...
        delete:
          id: DeleteImage
          ...
```

This compiles the final path by **concatenating segments down the tree** — e.g. `/api/v1` + `/resources` + `/image` → `/api/v1/resources/image`. A single key can also combine multiple segments at once, like `/image/{imageId}`.

`tags` declared at any level in the tree apply to every method nested beneath it, so you only need to state a tag once per group of related endpoints instead of on every method.

#### Method fields

Each HTTP method (`get`/`post`/`put`/`patch`/`delete`) supports:

| Field | Description |
|---|---|
| `id` | Operation ID (maps to OpenAPI `operationId`) |
| `description` | Human-readable summary |
| `traits` | List of trait invocations to merge in, e.g. `["paged(20,100)"]` |
| `params` | Query/path parameters — list of `{ name, schema, required? }` (`required` defaults to `true`) |
| `headers` | Header parameters — same shape as `params` |
| `body` | Request body, keyed by content type → schema expression |
| `responses` | Status-code-keyed responses, each with a `description` and content-type → schema mappings |

Example:

```yaml
put:
  id: add_event
  body:
    application/json:
      <EventIn>
  responses:
    201:
      description: Created event
      application/json: <Event>
```

Response status codes may also use a two-`X` wildcard shorthand (e.g. `4XX`) per the schema, in addition to exact codes like `200`/`204`.

Multipart uploads are expressed the same way, just with a different content type:

```yaml
put:
  id: UploadImage
  body:
    multipart/form-data:
      image: <File>
  responses:
    200:
      description: Uploaded image
      application/json:
        <ImageInfo>
```

---

## Full Example

A minimal end-to-end slice — qapi source:

```yaml
info:
  title: Example API
  version: 0.0.1

servers:
  - url: "http://localhost:8080"

schemas:
  DefaultError:
    code: string($errorCode)
    details: string

  Category:
    id: integer
    name: string
    color: string
    events_count: integer

  CategoryIn:
    name: string
    color: string

defaultResponses:
  400:
    description: Bad Request
    application/json:
      <DefaultError>

paths:
  /api/v1:
    /category:
      tags: [Category]
      get:
        id: list_categories
        responses:
          200:
            description: Categories list
            application/json:
              <Category>[]
      put:
        id: put_category
        body:
          application/json:
            <CategoryIn>
        responses:
          200:
            description: Category sent succesfully
            application/json:
              <Category>
```

Compiling this with `qapi compile -i example.qapi.yaml -o openapi.yaml` produces a full OpenAPI 3.1 document with `components.schemas.Category`/`CategoryIn`/`DefaultError` fully expanded, `required` arrays computed from your `?` markers, and the `400` response attached to both `GET /api/v1/category` and `PUT /api/v1/category` automatically.

---

## License

*(Not yet specified.)*