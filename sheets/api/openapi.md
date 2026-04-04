# OpenAPI (API Specification Standard)

Define, document, and generate client/server code from a machine-readable API specification using the OpenAPI 3.x standard, formerly known as Swagger.

## Specification Structure

### Minimal OpenAPI 3.0 Document

```yaml
openapi: 3.0.3
info:
  title: My API
  version: 1.0.0
  description: Production API for widget management
  contact:
    name: API Support
    email: api@example.com
  license:
    name: MIT
servers:
  - url: https://api.example.com/v1
    description: Production
  - url: https://staging-api.example.com/v1
    description: Staging
paths: {}
```

### OpenAPI 3.1 Differences

```yaml
# 3.1 uses full JSON Schema draft 2020-12
openapi: 3.1.0
info:
  title: My API
  summary: Short summary (new in 3.1)
  version: 1.0.0
# webhooks are top-level in 3.1
webhooks:
  orderCreated:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Order'
      responses:
        '200':
          description: Webhook processed
```

## Paths and Operations

### Path Items

```yaml
paths:
  /widgets:
    get:
      operationId: listWidgets
      summary: List all widgets
      tags:
        - Widgets
      parameters:
        - name: limit
          in: query
          required: false
          schema:
            type: integer
            default: 20
            maximum: 100
        - name: cursor
          in: query
          schema:
            type: string
      responses:
        '200':
          description: Paginated list of widgets
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WidgetList'
          headers:
            X-Request-Id:
              schema:
                type: string
                format: uuid
        '429':
          $ref: '#/components/responses/TooManyRequests'
    post:
      operationId: createWidget
      summary: Create a new widget
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/WidgetCreate'
            example:
              name: "Sprocket"
              weight: 1.5
      responses:
        '201':
          description: Widget created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Widget'
```

### Path Parameters

```yaml
paths:
  /widgets/{widgetId}:
    parameters:
      - name: widgetId
        in: path
        required: true
        schema:
          type: string
          format: uuid
    get:
      operationId: getWidget
      responses:
        '200':
          description: Single widget
        '404':
          $ref: '#/components/responses/NotFound'
    delete:
      operationId: deleteWidget
      responses:
        '204':
          description: Widget deleted
```

## Components and Schemas

### Reusable Schemas

```yaml
components:
  schemas:
    Widget:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: string
          format: uuid
          readOnly: true
        name:
          type: string
          minLength: 1
          maxLength: 255
        weight:
          type: number
          format: double
          minimum: 0
        tags:
          type: array
          items:
            type: string
          maxItems: 10
        status:
          type: string
          enum:
            - active
            - retired
            - draft
          default: draft
        metadata:
          type: object
          additionalProperties:
            type: string
        createdAt:
          type: string
          format: date-time
          readOnly: true
    WidgetCreate:
      allOf:
        - $ref: '#/components/schemas/Widget'
        - type: object
          required:
            - name
    WidgetList:
      type: object
      properties:
        items:
          type: array
          items:
            $ref: '#/components/schemas/Widget'
        nextCursor:
          type: string
          nullable: true
        total:
          type: integer
```

### Polymorphism

```yaml
components:
  schemas:
    Shape:
      oneOf:
        - $ref: '#/components/schemas/Circle'
        - $ref: '#/components/schemas/Rectangle'
      discriminator:
        propertyName: shapeType
        mapping:
          circle: '#/components/schemas/Circle'
          rectangle: '#/components/schemas/Rectangle'
    Circle:
      type: object
      required: [shapeType, radius]
      properties:
        shapeType:
          type: string
        radius:
          type: number
    Rectangle:
      type: object
      required: [shapeType, width, height]
      properties:
        shapeType:
          type: string
        width:
          type: number
        height:
          type: number
```

## Security Schemes

### Authentication Definitions

```yaml
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
    OAuth2:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://auth.example.com/authorize
          tokenUrl: https://auth.example.com/token
          scopes:
            read:widgets: Read widget data
            write:widgets: Create and modify widgets
# Apply globally
security:
  - BearerAuth: []
  - OAuth2:
      - read:widgets
```

## Code Generation

### openapi-generator-cli

```bash
# Install
npm install -g @openapitools/openapi-generator-cli

# Generate Go client
openapi-generator-cli generate \
  -i openapi.yaml \
  -g go \
  -o ./gen/go-client \
  --additional-properties=packageName=widgetapi

# Generate Python server stub
openapi-generator-cli generate \
  -i openapi.yaml \
  -g python-flask \
  -o ./gen/python-server

# Generate TypeScript Axios client
openapi-generator-cli generate \
  -i openapi.yaml \
  -g typescript-axios \
  -o ./gen/ts-client

# List all available generators
openapi-generator-cli list

# Validate spec
openapi-generator-cli validate -i openapi.yaml

# Generate with custom templates
openapi-generator-cli generate \
  -i openapi.yaml \
  -g go \
  -t ./custom-templates/ \
  -o ./gen/go-client
```

### Other Generation Tools

```bash
# oapi-codegen (Go-specific, lightweight)
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
oapi-codegen -package api -generate types,server openapi.yaml > api.gen.go

# Swagger UI via Docker
docker run -p 8080:8080 \
  -e SWAGGER_JSON=/spec/openapi.yaml \
  -v $(pwd):/spec \
  swaggerapi/swagger-ui

# Redocly CLI — lint and bundle
npm install -g @redocly/cli
redocly lint openapi.yaml
redocly bundle openapi.yaml -o bundled.yaml
redocly preview-docs openapi.yaml
```

## Validation and Linting

### Spectral (Stoplight)

```bash
npm install -g @stoplight/spectral-cli

# Lint with default OpenAPI ruleset
spectral lint openapi.yaml

# Custom ruleset (.spectral.yaml)
# extends: spectral:oas
# rules:
#   operation-operationId: error
#   info-description: warn

spectral lint openapi.yaml --ruleset .spectral.yaml
```

### Spec Diff and Breaking Changes

```bash
# oasdiff — detect breaking changes
go install github.com/tufin/oasdiff@latest
oasdiff breaking base.yaml revision.yaml
oasdiff changelog base.yaml revision.yaml
oasdiff diff base.yaml revision.yaml --format yaml
```

## Versioning Strategies

### URL-Based Versioning

```yaml
servers:
  - url: https://api.example.com/v1
  - url: https://api.example.com/v2
```

### Header-Based Versioning

```yaml
parameters:
  - name: API-Version
    in: header
    required: false
    schema:
      type: string
      default: "2024-01-01"
```

## Tips

- Always set `operationId` on every operation -- generated code uses it for method names and missing IDs produce ugly defaults
- Use `$ref` aggressively to keep specs DRY; put shared schemas, parameters, and responses in `components/`
- Run `spectral lint` in CI to catch spec drift before it reaches code generation
- Prefer `oneOf` with `discriminator` over untyped `object` for polymorphic responses -- clients get proper type unions
- Pin your spec to a specific OpenAPI version (3.0.3 or 3.1.0) and do not mix conventions between them
- Set `readOnly: true` on server-generated fields like `id` and `createdAt` so generators exclude them from create models
- Use `format: date-time` for timestamps -- it maps to proper datetime types in every language generator
- Run `oasdiff breaking` in CI/CD pipelines to block PRs that introduce breaking API changes
- Keep a separate `openapi.yaml` per major version -- do not try to serve v1 and v2 from one spec
- Add `example` values to schemas so Swagger UI renders useful try-it-out payloads
- Use `tags` to group operations logically -- generators use tags to organize code into modules or controllers

## See Also

- rest-api, graphql, grpc, json-schema, swagger, api-gateway

## References

- [OpenAPI 3.0.3 Specification](https://spec.openapis.org/oas/v3.0.3)
- [OpenAPI 3.1.0 Specification](https://spec.openapis.org/oas/v3.1.0)
- [OpenAPI Generator](https://openapi-generator.tech/)
- [Spectral Linter](https://stoplight.io/open-source/spectral)
- [Redocly CLI](https://redocly.com/docs/cli/)
- [oasdiff - Breaking Change Detection](https://github.com/Tufin/oasdiff)
