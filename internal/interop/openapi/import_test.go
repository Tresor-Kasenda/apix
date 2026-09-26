package openapi

import (
	"encoding/json"
	"testing"
)

const specV3 = `
openapi: 3.0.3
info:
  title: Shop API
  version: "1.0"
servers:
  - url: http://localhost:{port}/api/v1
    variables:
      port:
        default: "8000"
paths:
  /users:
    get:
      operationId: listUsers
      parameters:
        - name: page
          in: query
          required: true
          schema: { type: integer, default: 1 }
        - name: q
          in: query
          schema: { type: string }
    post:
      operationId: createUser
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewUser'
      responses:
        201:
          description: created
  /users/{userId}:
    parameters:
      - name: X-Tenant
        in: header
        required: true
        schema: { type: string }
    delete:
      responses:
        204:
          description: deleted
components:
  schemas:
    NewUser:
      type: object
      properties:
        id: { type: integer, readOnly: true }
        email: { type: string, format: email }
        role: { type: string, enum: [admin, member] }
        tags:
          type: array
          items: { type: string }
        manager:
          $ref: '#/components/schemas/NewUser'
`

func TestParseOpenAPI3(t *testing.T) {
	res, err := Parse([]byte(specV3), Options{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if res.Title != "Shop API" || res.Server != "http://localhost:8000/api/v1" || res.BasePath != "/api/v1" {
		t.Fatalf("unexpected metadata: %+v", res)
	}
	if len(res.Requests) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(res.Requests))
	}

	list := res.Requests[0]
	if list.Name != "listUsers" || list.Method != "GET" || list.Path != "/users" {
		t.Fatalf("unexpected list request: %+v", list)
	}
	if list.Query["page"] != "1" || len(list.Query) != 1 {
		t.Fatalf("expected only the required query param with its default, got %v", list.Query)
	}

	create := res.Requests[1]
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(create.Body), &body); err != nil {
		t.Fatalf("body is not JSON: %v\n%s", err, create.Body)
	}
	if body["email"] != "user@example.com" || body["role"] != "admin" {
		t.Fatalf("unexpected generated body: %v", body)
	}
	if _, ok := body["id"]; ok {
		t.Fatalf("readOnly properties must be skipped: %v", body)
	}

	del := res.Requests[2]
	if del.Method != "DELETE" || del.Path != "/users/${userId}" {
		t.Fatalf("unexpected delete request: %+v", del)
	}
	if del.Headers["X-Tenant"] != "${X_Tenant}" {
		t.Fatalf("expected shared required header placeholder, got %v", del.Headers)
	}
}

func TestParseOpenAPIIncludeBasePath(t *testing.T) {
	res, err := Parse([]byte(specV3), Options{IncludeBasePath: true})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if res.Requests[0].Path != "/api/v1/users" {
		t.Fatalf("expected base path prefix, got %q", res.Requests[0].Path)
	}
}

func TestParseSwagger2JSON(t *testing.T) {
	spec := `{
	  "swagger": "2.0",
	  "info": {"title": "Legacy", "version": "0.1"},
	  "host": "api.example.com",
	  "basePath": "/v2",
	  "schemes": ["https"],
	  "paths": {
	    "/login": {
	      "post": {
	        "parameters": [{"in": "body", "name": "body", "schema": {"$ref": "#/definitions/Login"}}]
	      }
	    }
	  },
	  "definitions": {
	    "Login": {"type": "object", "properties": {"email": {"type": "string", "example": "a@b.c"}}}
	  }
	}`

	res, err := Parse([]byte(spec), Options{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if res.Server != "https://api.example.com/v2" {
		t.Fatalf("unexpected server %q", res.Server)
	}
	if len(res.Requests) != 1 || res.Requests[0].Method != "POST" {
		t.Fatalf("unexpected requests: %+v", res.Requests)
	}
	if res.Requests[0].Body != "{\n  \"email\": \"a@b.c\"\n}" {
		t.Fatalf("unexpected body: %s", res.Requests[0].Body)
	}
}

func TestParseRejectsNonSpec(t *testing.T) {
	if _, err := Parse([]byte(`{"hello": "world"}`), Options{}); err == nil {
		t.Fatal("expected an error for a non-OpenAPI document")
	}
}
