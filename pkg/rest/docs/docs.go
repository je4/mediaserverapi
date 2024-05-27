// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "Jürgen Enge",
            "url": "https://ub.unibas.ch",
            "email": "juergen.enge@unibas.ch"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/cache/{collection}/{signature}/{action}/{params}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "gets cache data",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "mediaserver"
                ],
                "summary": "get cache metadata",
                "operationId": "get-collection-signature-action-params-cache",
                "parameters": [
                    {
                        "type": "string",
                        "description": "collection name",
                        "name": "collection",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "signature",
                        "name": "signature",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "action",
                        "name": "action",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "params",
                        "name": "params",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    }
                }
            }
        },
        "/collection": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "retrieves mediaserver collections",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "mediaserver"
                ],
                "summary": "gets collections data",
                "operationId": "get-collections",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/rest.HTTPCollectionResultMessage"
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    }
                }
            }
        },
        "/collection/{collection}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "retrieves mediaserver collection information",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "mediaserver"
                ],
                "summary": "gets collection data",
                "operationId": "get-collection-by-name",
                "parameters": [
                    {
                        "type": "string",
                        "description": "collection name",
                        "name": "collection",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    }
                }
            },
            "put": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "creates a new item for indexing",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "mediaserver"
                ],
                "summary": "creates new item",
                "operationId": "put-collection-item",
                "parameters": [
                    {
                        "type": "string",
                        "description": "collection name",
                        "name": "collection",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "new item to create",
                        "name": "item",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/rest.CreateItemMessage"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    }
                }
            }
        },
        "/ingest": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "gets next item for indexing",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "mediaserver"
                ],
                "summary": "next ingest item",
                "operationId": "get-ingest-item",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPIngestItemMessage"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    }
                }
            }
        },
        "/ping": {
            "get": {
                "description": "for testing if server is running",
                "produces": [
                    "text/plain"
                ],
                "tags": [
                    "mediaserver"
                ],
                "summary": "does pong",
                "operationId": "get-ping",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/storage/{storageid}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "retrieves mediaserver storage information",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "mediaserver"
                ],
                "summary": "gets storage data",
                "operationId": "get-storage-by-id",
                "parameters": [
                    {
                        "type": "string",
                        "description": "storage id",
                        "name": "storageid",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/rest.HTTPResultMessage"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "rest.CreateItemMessage": {
            "type": "object",
            "properties": {
                "ingest_type": {
                    "description": "The type of ingest\n* keep: ingest without copy of data\n* copy: ingest with copy of data\n* move: ingest with copy of data and deletion after copy\ndefault: keep",
                    "type": "string",
                    "enum": [
                        "keep",
                        "copy",
                        "move"
                    ],
                    "example": "copy"
                },
                "parent": {
                    "description": "Parent is an optional field that represents the signature of the parent item, if any.\nThis is used to establish a parent-child relationship between items.",
                    "type": "string",
                    "example": "test/10_3931_e-rara-20425_20230519T104744_gen6_ver1.zip_10_3931_e-rara-20425_export_mets.xml"
                },
                "path": {
                    "description": "Urn represents the path of the item. It is used to locate the item within the system.",
                    "type": "string",
                    "example": "vfs://test/ub-reprofiler/mets-container-doi/bau_1/2023/9940561370105504/10_3931_e-rara-20425_20230519T104744_gen6_ver1.zip/10_3931_e-rara-20425/export_mets.xml"
                },
                "public": {
                    "description": "Public is an optional field that can be used to store any public data associated with the item.",
                    "type": "boolean",
                    "example": true
                },
                "public_actions": {
                    "description": "PublicActions is an optional field that can be used to store any public actions associated with the item.",
                    "type": "string"
                },
                "signature": {
                    "description": "Signature is a unique identifier for the item within its collection.",
                    "type": "string",
                    "example": "10_3931_e-rara-20425_20230519T104744_gen6_ver1.zip_10_3931_e-rara-20425_export_mets.xml"
                }
            }
        },
        "rest.HTTPCollectionResultMessage": {
            "type": "object",
            "properties": {
                "description": {
                    "type": "string"
                },
                "identifier": {
                    "type": "string"
                },
                "jwtkey": {
                    "type": "string"
                },
                "public": {
                    "type": "string"
                },
                "secret": {
                    "type": "string"
                },
                "signature_prefix": {
                    "type": "string"
                },
                "storage": {
                    "$ref": "#/definitions/rest.HTTPStorageResultMessage"
                }
            }
        },
        "rest.HTTPIngestItemMessage": {
            "type": "object",
            "properties": {
                "collection": {
                    "type": "string"
                },
                "path": {
                    "type": "string"
                },
                "signature": {
                    "type": "string"
                }
            }
        },
        "rest.HTTPResultMessage": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "integer",
                    "example": 400
                },
                "message": {
                    "type": "string",
                    "example": "status bad request"
                }
            }
        },
        "rest.HTTPStorageResultMessage": {
            "type": "object",
            "properties": {
                "datadir": {
                    "type": "string"
                },
                "filebase": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "subitemdir": {
                    "type": "string"
                },
                "tempdir": {
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "Mediaserver API",
	Description:      "Mediaserver API for managing collections and items",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
